package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/trithemius/parq-comfyui/internal/collector"
	"github.com/trithemius/parq-comfyui/internal/extractor"
	"github.com/trithemius/parq-comfyui/internal/parquet"
	"github.com/trithemius/parq-comfyui/internal/progress"
)

var (
	input         string
	fileList      string
	database      string
	override      bool
	useParameters bool
	usePrompt     bool
	recursive     bool
)

func init() {
	flag.StringVar(&input, "input", "", "Input: directory, single file, or glob pattern")
	flag.StringVar(&input, "i", "", "Input (shorthand)")
	flag.StringVar(&fileList, "file-list", "", "Text file containing list of image paths")
	flag.StringVar(&fileList, "f", "", "Text file (shorthand)")
	flag.StringVar(&database, "database", "", "Path to Parquet database file (required)")
	flag.StringVar(&database, "db", "", "Database (shorthand)")
	flag.BoolVar(&override, "override", false, "Override existing entries in database")
	flag.BoolVar(&useParameters, "use-parameters", false, "Use A1111/parameters-style extraction")
	flag.BoolVar(&usePrompt, "use-prompt", false, "Use ComfyUI prompt/workflow extraction (default)")
	flag.BoolVar(&recursive, "recursive", false, "Recursively search for images")
	flag.BoolVar(&recursive, "r", false, "Recursive (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of parq-comfyui:\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -i, --input <path>      Input: directory, single file, or glob pattern\n")
		fmt.Fprintf(os.Stderr, "  -f, --file-list <path>  Text file containing list of image paths\n")
		fmt.Fprintf(os.Stderr, "  -db, --database <path>  Path to Parquet database file (required)\n")
		fmt.Fprintf(os.Stderr, "  -r, --recursive         Recursively search for images\n")
		fmt.Fprintf(os.Stderr, "  --override              Override existing entries in database\n")
		fmt.Fprintf(os.Stderr, "  --use-parameters        A1111-style parameters extraction\n")
		fmt.Fprintf(os.Stderr, "  --use-prompt            ComfyUI-style extraction (default)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  parq-comfyui -i ./renders -db prompts.parquet\n")
		fmt.Fprintf(os.Stderr, "  parq-comfyui -i \"*.png\" --db prompts.parquet --use-parameters\n")
	}
}

type appState struct {
	db              *parquet.ParquetDB
	newEntries      []parquet.Entry
	mu              sync.Mutex
	successCount    int
	errorCount      int
	skippedCount    int
	noPromptCount   int
	skippedNoParam  int
	databasePath    string
	overrideEnabled bool
}

func main() {
	flag.Parse()

	if database == "" {
		fmt.Println("✗ Error: --database is required")
		flag.Usage()
		os.Exit(1)
	}

	if input == "" && fileList == "" {
		fmt.Println("✗ Error: --input or --file-list is required")
		flag.Usage()
		os.Exit(1)
	}

	if useParameters && usePrompt {
		fmt.Println("✗ Error: --use-parameters and --use-prompt are mutually exclusive")
		os.Exit(1)
	}

	// Gather files
	allImageFiles, err := collector.CollectImages(input, recursive, fileList)
	if err != nil {
		fmt.Printf("✗ Error collecting images: %v\n", err)
		os.Exit(1)
	}

	if len(allImageFiles) == 0 {
		fmt.Println("✗ No PNG files found")
		return
	}

	// Load DB
	db, err := parquet.LoadParquetDB(database)
	if err != nil {
		fmt.Printf("✗ Error loading database: %v\n", err)
		os.Exit(1)
	}

	state := &appState{
		db:              db,
		databasePath:    database,
		overrideEnabled: override,
	}

	fmt.Println("🎨 parq-comfyui (Go)")
	fmt.Printf("Input: %s\n", input)
	if fileList != "" {
		fmt.Printf("File list: %s\n", fileList)
	}
	fmt.Printf("Parquet database: %s\n", database)
	fmt.Printf("Existing entries in database: %d\n", len(db.Entries))
	fmt.Printf("Override existing: %v\n", override)
	fmt.Println("\n💡 Tip: Press Ctrl-C anytime to save progress and exit gracefully")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Printf("Found %d PNG file(s) total\n", len(allImageFiles))

	var imagesToProcess []string
	existingEntriesMap := make(map[string]parquet.Entry)

	for _, img := range allImageFiles {
		if db.Exists(img) {
			if override {
				entry, _ := db.GetEntry(img)
				existingEntriesMap[img] = entry
				imagesToProcess = append(imagesToProcess, img)
			}
		} else {
			imagesToProcess = append(imagesToProcess, img)
		}
	}

	state.skippedCount = len(allImageFiles) - len(imagesToProcess)
	if state.skippedCount > 0 {
		fmt.Printf("Skipping %d image(s) already in database\n", state.skippedCount)
	}

	if len(imagesToProcess) == 0 {
		fmt.Println("\n✓ No images to process. All images already exist in database.")
		return
	}

	fmt.Printf("Processing %d image(s)\n\n", len(imagesToProcess))

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	pb := progress.NewProgressBar(len(imagesToProcess), "Processing images")

	go func() {
		<-sigChan
		fmt.Println("\n\n⚠ Interrupt received (Ctrl-C). Saving progress...")
		state.saveAndExit()
	}()

	for _, imagePath := range imagesToProcess {
		result, err := extractor.Extract(imagePath, useParameters)
		if err != nil {
			state.mu.Lock()
			state.errorCount++
			state.mu.Unlock()
			pb.Describe(fmt.Sprintf("✗ %s: %v", filepath.Base(imagePath), err))
			pb.Increment()
			continue
		}

		if useParameters && len(result.PositivePrompts) == 0 {
			state.mu.Lock()
			state.skippedNoParam++
			state.mu.Unlock()
			pb.UpdateWithStatus(fmt.Sprintf("SKIP %s", filepath.Base(imagePath)))
			continue
		}

		var promptText string
		if len(result.PositivePrompts) > 0 {
			var texts []string
			for _, p := range result.PositivePrompts {
				texts = append(texts, p.Text)
			}
			promptText = strings.Join(texts, "\n---\n")
		}

		if promptText == "" {
			state.mu.Lock()
			state.noPromptCount++
			state.mu.Unlock()
		}

		now := time.Now()
		entry := parquet.Entry{
			ImagePath:   imagePath,
			Prompt:      "",
			Description: promptText,
			CreatedAt:   now,
		}

		if existing, ok := existingEntriesMap[imagePath]; ok {
			entry.CreatedAt = existing.CreatedAt
			entry.ModifiedAt = &now
		}

		state.mu.Lock()
		state.newEntries = append(state.newEntries, entry)
		state.successCount++
		state.mu.Unlock()
		
		pb.UpdateWithStatus(fmt.Sprintf("✓ %s", filepath.Base(imagePath)))
	}

	pb.Finish()
	fmt.Println("\nSaving results to database...")
	state.saveAndExit()
}

func (s *appState) saveAndExit() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.newEntries) > 0 {
		s.db.AddEntries(s.newEntries, s.overrideEnabled)
		if err := s.db.Save(); err != nil {
			fmt.Printf("✗ Error saving database: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("✓ Database updated: %s\n", s.databasePath)
			fmt.Printf("  New entries added: %d\n", len(s.newEntries))
			fmt.Printf("  Total entries in database: %d\n", len(s.db.Entries))
		}
	} else {
		fmt.Println("⊘ No new entries to save.")
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Processing complete!")
	fmt.Printf("✓ Successfully processed: %d\n", s.successCount)
	if s.noPromptCount > 0 {
		fmt.Printf("⚠ Images with no prompts found: %d\n", s.noPromptCount)
	}
	if s.skippedNoParam > 0 {
		fmt.Printf("✓ Images skipped (no parameters found): %d\n", s.skippedNoParam)
	}
	if s.errorCount > 0 {
		fmt.Printf("✗ Errors: %d\n", s.errorCount)
	}
	if s.skippedCount > 0 {
		fmt.Printf("⊘ Skipped (already in database): %d\n", s.skippedCount)
	}
	
	os.Exit(0)
}
