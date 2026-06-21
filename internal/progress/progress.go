package progress

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressBar wrapper for terminal progress
type ProgressBar struct {
	bar              *progressbar.ProgressBar
	currentFile      string
	lastRedraw       time.Time
	throttleDuration time.Duration
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, description string) *ProgressBar {
	bar := progressbar.NewOptions(total,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription("[cyan]⚙ " + description + "...[reset]"),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("img/s"),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[cyan]█[reset]",
			SaucerHead:    "[cyan]█[reset]",
			SaucerPadding: "░",
			BarStart:      "[cyan]▕[reset]",
			BarEnd:        "[cyan]▏[reset]",
		}),
		progressbar.OptionThrottle(65 * time.Millisecond),
	)
	return &ProgressBar{
		bar:              bar,
		lastRedraw:       time.Unix(0, 0),
		throttleDuration: 65 * time.Millisecond,
	}
}

// UpdateWithStatus updates the progress bar with a status message and increments
func (pb *ProgressBar) UpdateWithStatus(status string) {
	if status != "" {
		_ = pb.bar.Clear()
		fmt.Printf("\n\033[2K\033[A\r")
		fmt.Println(status)
	}
	_ = pb.bar.Add(1)
	fmt.Printf("\n\033[2K\033[90m⚙ Processing: %s\033[0m\033[A\r", pb.currentFile)
	pb.lastRedraw = time.Now()
}

// Increment just increments the progress bar
func (pb *ProgressBar) Increment() {
	_ = pb.bar.Add(1)
	if time.Since(pb.lastRedraw) >= pb.throttleDuration {
		fmt.Printf("\n\033[2K\033[90m⚙ Processing: %s\033[0m\033[A\r", pb.currentFile)
		pb.lastRedraw = time.Now()
	}
}

// Describe sets the description without incrementing
func (pb *ProgressBar) Describe(desc string) {
	pb.currentFile = desc
}

// Finish finishes the progress bar
func (pb *ProgressBar) Finish() {
	_ = pb.bar.Finish()
	fmt.Printf("\n\033[2K\033[A\r")
}
