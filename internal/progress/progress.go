package progress

import (
	"fmt"
	"os"
	"github.com/schollz/progressbar/v3"
)

// ProgressBar wrapper for terminal progress
type ProgressBar struct {
	bar *progressbar.ProgressBar
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, description string) *ProgressBar {
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stdout), // Standard output
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	return &ProgressBar{bar: bar}
}

// UpdateWithStatus updates the progress bar with a status message
func (pb *ProgressBar) UpdateWithStatus(status string) {
	_ = pb.bar.Add(1)
	if status != "" {
		pb.bar.Describe(status)
	}
}

// Increment just increments the progress bar
func (pb *ProgressBar) Increment() {
	_ = pb.bar.Add(1)
}

// Describe sets the description
func (pb *ProgressBar) Describe(desc string) {
	pb.bar.Describe(desc)
}

// Finish finishes the progress bar
func (pb *ProgressBar) Finish() {
	_ = pb.bar.Finish()
	fmt.Println()
}
