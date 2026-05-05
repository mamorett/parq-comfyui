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
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("img/s"),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetPredictTime(false),
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
		progressbar.OptionFullWidth(),
	)
	return &ProgressBar{bar: bar}
}

// UpdateWithStatus updates the progress bar with a status message and increments
func (pb *ProgressBar) UpdateWithStatus(status string) {
	if status != "" {
		pb.bar.Describe(status)
	}
	_ = pb.bar.Add(1)
}

// Increment just increments the progress bar
func (pb *ProgressBar) Increment() {
	_ = pb.bar.Add(1)
}

// Describe sets the description without incrementing
func (pb *ProgressBar) Describe(desc string) {
	pb.bar.Describe(desc)
}

// Finish finishes the progress bar
func (pb *ProgressBar) Finish() {
	_ = pb.bar.Finish()
}
