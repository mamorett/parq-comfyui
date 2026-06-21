package progress

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// ProgressBar wrapper for terminal progress
type ProgressBar struct {
	bar              *progressbar.ProgressBar
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
		progressbar.OptionSetWidth(10), // Short bar to prevent line wrapping
		progressbar.OptionThrottle(65 * time.Millisecond),
	)
	return &ProgressBar{
		bar:              bar,
		lastRedraw:       time.Unix(0, 0),
		throttleDuration: 65 * time.Millisecond,
	}
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // fallback
	}
	return width
}

func (pb *ProgressBar) updateDescription(desc string) {
	if time.Since(pb.lastRedraw) >= pb.throttleDuration {
		termWidth := getTerminalWidth()
		maxDescLen := termWidth - 55
		if maxDescLen < 20 {
			maxDescLen = 20
		}

		truncated := desc
		if len(desc) > maxDescLen {
			prefix := ""
			content := desc
			if strings.HasPrefix(desc, "✓ ") {
				prefix = "✓ "
				content = desc[2:]
			} else if strings.HasPrefix(desc, "SKIP ") {
				prefix = "SKIP "
				content = desc[5:]
			} else if strings.HasPrefix(desc, "✗ ") {
				prefix = "✗ "
				content = desc[2:]
			} else if strings.HasPrefix(desc, "Processing ") {
				prefix = "Processing "
				content = desc[11:]
			}

			adjMaxLen := maxDescLen - len(prefix)
			if adjMaxLen < 10 {
				adjMaxLen = 10
			}

			if len(content) > adjMaxLen {
				half := (adjMaxLen - 3) / 2
				truncated = prefix + content[:half] + "..." + content[len(content)-half:]
			} else {
				truncated = prefix + content
			}
		}

		colorized := truncated
		if strings.HasPrefix(truncated, "✓ ") {
			colorized = "[green]✓[reset] [cyan]" + truncated[2:] + "[reset]"
		} else if strings.HasPrefix(truncated, "SKIP ") {
			colorized = "[yellow]SKIP[reset] [cyan]" + truncated[5:] + "[reset]"
		} else if strings.HasPrefix(truncated, "✗ ") {
			colorized = "[red]✗[reset] [cyan]" + truncated[2:] + "[reset]"
		} else {
			colorized = "[cyan]" + truncated + "[reset]"
		}

		pb.bar.Describe(colorized)
		pb.lastRedraw = time.Now()
	}
}

// UpdateWithStatus updates the progress bar with a status message and increments
func (pb *ProgressBar) UpdateWithStatus(status string) {
	if status != "" {
		_ = pb.bar.Clear()
		fmt.Println(status)
	}
	_ = pb.bar.Add(1)
	pb.lastRedraw = time.Now()
}

// Increment just increments the progress bar
func (pb *ProgressBar) Increment() {
	_ = pb.bar.Add(1)
}

// IncrementWithStatus updates description and increments the progress bar
func (pb *ProgressBar) IncrementWithStatus(status string) {
	pb.updateDescription(status)
	_ = pb.bar.Add(1)
}

// Describe sets the description with rate-limiting and smart truncation
func (pb *ProgressBar) Describe(desc string) {
	pb.updateDescription(desc)
}

// Finish finishes the progress bar
func (pb *ProgressBar) Finish() {
	_ = pb.bar.Finish()
}
