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

var rawAsciiArt = `  ____   _    ____   ___       ____ ___  __  __ _____ __   __ _   _ ___ 
 |  _ \ / \  |  _ \ / _ \     / ___/ _ \|  \/  |  ___|\ \ / /| | | |_ _|
 | |_) / _ \ | |_) | | | |   | |  | | | | |\/| | |_    \ V / | | | || | 
 |  __/ ___ \|  _ <| |_| |   | |__| |_| | |  | |  _|    | |  | |_| || | 
 |_| /_/   \_\_| \_\\__\_\    \____\___/|_|  |_|_|      |_|   \___/|___|`

func printLogo() {
	fmt.Printf("\033[36m%s\033[0m\n\n", rawAsciiArt)
}

var (
	input         string
	fileList      string
	database      string
	override      bool
	useParameters bool
	usePrompt     bool
	recursive     bool
	clean         bool
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
	flag.BoolVar(&clean, "clean", false, "Remove entries whose image files no longer exist on disk")

	flag.Usage = func() {
		printLogo()
		fmt.Fprintf(os.Stderr, "\033[1;36mUsage of parq-comfyui:\033[0m\n\n")
		fmt.Fprintf(os.Stderr, "\033[1mOptions:\033[0m\n")
		fmt.Fprintf(os.Stderr, "  \033[36m-i, --input\033[0m \033[33m<path>\033[0m      Input: directory, single file, or glob pattern\n")
		fmt.Fprintf(os.Stderr, "  \033[36m-f, --file-list\033[0m \033[33m<path>\033[0m  Text file containing list of image paths\n")
		fmt.Fprintf(os.Stderr, "  \033[36m-db, --database\033[0m \033[33m<path>\033[0m  Path to Parquet database file (\033[1;31mrequired\033[0m)\n")
		fmt.Fprintf(os.Stderr, "  \033[36m-r, --recursive\033[0m         Recursively search for images\n")
		fmt.Fprintf(os.Stderr, "  \033[36m--override\033[0m              Override existing entries in database\n")
		fmt.Fprintf(os.Stderr, "  \033[36m--use-parameters\033[0m        A1111-style parameters extraction\n")
		fmt.Fprintf(os.Stderr, "  \033[36m--use-prompt\033[0m            ComfyUI-style extraction (default)\n")
		fmt.Fprintf(os.Stderr, "  \033[36m--clean\033[0m                 Remove stale database entries (files no longer exist on disk)\n\n")
		fmt.Fprintf(os.Stderr, "\033[1mExamples:\033[0m\n")
		fmt.Fprintf(os.Stderr, "  \033[90mparq-comfyui -i ./renders -db prompts.parquet\033[0m\n")
		fmt.Fprintf(os.Stderr, "  \033[90mparq-comfyui -i \"*.png\" --db prompts.parquet --use-parameters\033[0m\n")
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
		fmt.Println("\033[31m✗ Error: --database is required\033[0m")
		flag.Usage()
		os.Exit(1)
	}

	if useParameters && usePrompt {
		fmt.Println("\033[31m✗ Error: --use-parameters and --use-prompt are mutually exclusive\033[0m")
		os.Exit(1)
	}

	// Validate --clean requires either file list or input path
	if clean && input == "" && fileList == "" {
		fmt.Println("\033[31m✗ Error: --clean requires either --input or --file-list\033[0m")
		flag.Usage()
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

	if clean {
		removed := db.RemoveMissingEntries()
		if removed > 0 {
			fmt.Printf("♻ Removed %d stale entries (files no longer exist)\n", removed)
			if err := db.Save(); err != nil {
				fmt.Printf("✗ Error saving after clean: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("✓ No stale entries found")
		}

		if input == "" && fileList == "" {
			fmt.Printf("Total entries in database: %d\n", len(db.Entries))
			return
		}
	}

	state := &appState{
		db:              db,
		databasePath:    database,
		overrideEnabled: override,
	}

	printLogo()
	fmt.Printf("\033[1;36m🎨 parq-comfyui (Go)\033[0m\n")
	fmt.Printf("\033[90mInput:\033[0m %s\n", input)
	if fileList != "" {
		fmt.Printf("\033[90mFile list:\033[0m %s\n", fileList)
	}
	fmt.Printf("\033[90mParquet database:\033[0m %s\n", database)
	fmt.Printf("\033[90mExisting entries in database:\033[0m \033[32m%d\033[0m\n", len(db.Entries))
	fmt.Printf("\033[90mOverride existing:\033[0m %v\n", override)
	fmt.Println("\n💡 \033[33mTip:\033[0m Press \033[1;33mCtrl-C\033[0m anytime to save progress and exit gracefully")
	fmt.Println("\033[90m" + strings.Repeat("-", 60) + "\033[0m")

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
		baseName := filepath.Base(imagePath)
		pb.Describe("Processing " + baseName)

		result, err := extractor.Extract(imagePath, useParameters)
		if err != nil {
			state.mu.Lock()
			state.errorCount++
			state.mu.Unlock()
			pb.UpdateWithStatus(fmt.Sprintf("✗ %s: %v", baseName, err))
			continue
		}

		if useParameters && len(result.PositivePrompts) == 0 {
			state.mu.Lock()
			state.skippedNoParam++
			state.mu.Unlock()
			pb.IncrementWithStatus(fmt.Sprintf("SKIP %s", baseName))
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

		pb.IncrementWithStatus(fmt.Sprintf("✓ %s", baseName))
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
			fmt.Printf("\033[31m✗ Error saving database: %v\033[0m\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("\033[32m✓ Database updated:\033[0m %s\n", s.databasePath)
			fmt.Printf("  New entries added: \033[32m%d\033[0m\n", len(s.newEntries))
			fmt.Printf("  Total entries in database: \033[32m%d\033[0m\n", len(s.db.Entries))
		}
	} else {
		fmt.Println("\033[90m⊘ No new entries to save.\033[0m")
	}

	fmt.Println("\033[90m" + strings.Repeat("-", 60) + "\033[0m")
	fmt.Println("\033[1;36mProcessing complete!\033[0m")
	fmt.Printf("\033[32m✓ Successfully processed:\033[0m %d\n", s.successCount)
	if s.noPromptCount > 0 {
		fmt.Printf("\033[33m⚠ Images with no prompts found:\033[0m %d\n", s.noPromptCount)
	}
	if s.skippedNoParam > 0 {
		fmt.Printf("\033[32m✓ Images skipped (no parameters found):\033[0m %d\n", s.skippedNoParam)
	}
	if s.errorCount > 0 {
		fmt.Printf("\033[31m✗ Errors:\033[0m %d\n", s.errorCount)
	}
	if s.skippedCount > 0 {
		fmt.Printf("\033[90m⊘ Skipped (already in database):\033[0m %d\n", s.skippedCount)
	}
	
	os.Exit(0)
}
