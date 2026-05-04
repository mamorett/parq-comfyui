package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/trithemius/parq-comfyui/internal/collector"
	"github.com/trithemius/parq-comfyui/internal/extractor"
	"github.com/trithemius/parq-comfyui/internal/parquet"
	"github.com/trithemius/parq-comfyui/internal/progress"
)

var (
	inputFlag         = flag.String("input", "", "Input: directory, single file, or glob pattern")
	inputShortFlag    = flag.String("i", "", "Input: directory, single file, or glob pattern (shorthand)")
	fileListFlag      = flag.String("file-list", "", "Text file containing list of image paths")
	fileListShortFlag = flag.String("f", "", "Text file containing list of image paths (shorthand)")
	directoryFlag     = flag.String("directory", "", "[DEPRECATED] Use --input instead")
	directoryShortFlag = flag.String("d", "", "[DEPRECATED] Use --input instead (shorthand)")
	databaseFlag      = flag.String("database", "", "Path to Parquet database file (required)")
	dbFlag            = flag.String("db", "", "Path to Parquet database file (shorthand)")
	overrideFlag      = flag.Bool("override", false, "Override existing entries in database")
	useParametersFlag = flag.Bool("use-parameters", false, "Use A1111/parameters-style extraction")
	recursiveFlag     = flag.Bool("recursive", false, "Recursively search for images")
	recursiveShortFlag = flag.Bool("r", false, "Recursively search for images (shorthand)")
)

func main() {
	flag.Parse()

	input := *inputFlag
	if input == "" {
		input = *inputShortFlag
	}
	if input == "" {
		input = *directoryFlag
	}
	if input == "" {
		input = *directoryShortFlag
	}

	fileList := *fileListFlag
	if fileList == "" {
		fileList = *fileListShortFlag
	}

	database := *databaseFlag
	if database == "" {
		database = *dbFlag
	}

	recursive := *recursiveFlag || *recursiveShortFlag

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

	fmt.Println("ComfyUI Prompt Extractor (Go)")
	fmt.Printf("Input: %s\n", input)
	if fileList != "" {
		fmt.Printf("File list: %s\n", fileList)
	}
	fmt.Printf("Parquet database: %s\n", database)
	fmt.Printf("Existing entries in database: %d\n", len(db.Entries))
	fmt.Printf("Override existing: %v\n", *overrideFlag)
	fmt.Println("\n💡 Tip: Press Ctrl-C anytime to save progress and exit gracefully")
	fmt.Println(strings.Repeat("-", 60))

	fmt.Printf("Found %d PNG file(s) total\n", len(allImageFiles))

	var imagesToProcess []string
	existingEntries := make(map[string]parquet.Entry)

	for _, img := range allImageFiles {
		if db.Exists(img) {
			if *overrideFlag {
				entry, _ := db.GetEntry(img)
				existingEntries[img] = entry
				imagesToProcess = append(imagesToProcess, img)
			}
		} else {
			imagesToProcess = append(imagesToProcess, img)
		}
	}

	skippedCount := len(allImageFiles) - len(imagesToProcess)
	if skippedCount > 0 {
		fmt.Printf("Skipping %d image(s) already in database\n", skippedCount)
	}

	if len(imagesToProcess) == 0 {
		fmt.Println("\n✓ No images to process. All images already exist in database.")
		return
	}

	fmt.Printf("Processing %d image(s)\n\n", len(imagesToProcess))

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	var newEntries []parquet.Entry
	successCount := 0
	errorCount := 0
	noPromptCount := 0

	pb := progress.NewProgressBar(len(imagesToProcess), "Processing images")

	go func() {
		<-sigChan
		fmt.Println("\n\n⚠ Interrupt received (Ctrl-C). Saving progress...")
		if len(newEntries) > 0 {
			db.AddEntries(newEntries, *overrideFlag)
			if err := db.Save(database); err != nil {
				fmt.Printf("✗ Error saving database: %v\n", err)
			} else {
				fmt.Printf("✓ Progress saved: %d new entries added\n", len(newEntries))
			}
		}
		os.Exit(0)
	}()

	for _, imagePath := range imagesToProcess {
		result, err := extractor.Extract(imagePath, *useParametersFlag)
		if err != nil {
			errorCount++
			pb.Describe(fmt.Sprintf("✗ %s: %v", filepath.Base(imagePath), err))
			pb.UpdateWithStatus(fmt.Sprintf("✗ %s", filepath.Base(imagePath)))
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
			noPromptCount++
		}

		now := time.Now()
		entry := parquet.Entry{
			ImagePath:   imagePath,
			Prompt:      "",
			Description: promptText,
			CreatedAt:   now,
		}

		if existing, ok := existingEntries[imagePath]; ok {
			entry.CreatedAt = existing.CreatedAt
			entry.ModifiedAt = &now
		}

		newEntries = append(newEntries, entry)
		successCount++
		pb.UpdateWithStatus(fmt.Sprintf("✓ %s", filepath.Base(imagePath)))
	}

	pb.Finish()

	fmt.Println("\nSaving results to database...")
	db.AddEntries(newEntries, *overrideFlag)
	if err := db.Save(database); err != nil {
		fmt.Printf("✗ Error saving database: %v\n", err)
	} else {
		fmt.Printf("✓ Database updated: %s\n", database)
		fmt.Printf("  New entries added: %d\n", len(newEntries))
		fmt.Printf("  Total entries in database: %d\n", len(db.Entries))
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Processing complete!")
	fmt.Printf("✓ Successfully processed: %d\n", successCount)
	if noPromptCount > 0 {
		fmt.Printf("⚠ Images with no prompts found: %d\n", noPromptCount)
	}
	if errorCount > 0 {
		fmt.Printf("✗ Errors: %d\n", errorCount)
	}
	if skippedCount > 0 {
		fmt.Printf("⊘ Skipped (already in database): %d\n", skippedCount)
	}
}
