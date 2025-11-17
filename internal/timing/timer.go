package timing

import (
	"fmt"
	"strings"
	"time"
)

// Stage represents a single compilation stage with its timing information
type Stage struct {
	Name     string
	Duration time.Duration
}

// Timer tracks compilation stage timings
type Timer struct {
	stages       []Stage
	currentStage string
	startTime    time.Time
}

// NewTimer creates a new compilation timer
func NewTimer() *Timer {
	return &Timer{
		stages: make([]Stage, 0),
	}
}

// Start begins timing a new stage
func (t *Timer) Start(stageName string) {
	// If there's a stage already running, end it first
	if t.currentStage != "" {
		t.End()
	}

	t.currentStage = stageName
	t.startTime = time.Now()
}

// End stops timing the current stage and records its duration
func (t *Timer) End() {
	if t.currentStage == "" {
		return
	}

	duration := time.Since(t.startTime)
	t.stages = append(t.stages, Stage{
		Name:     t.currentStage,
		Duration: duration,
	})
	t.currentStage = ""
}

// Total returns the total compilation time
func (t *Timer) Total() time.Duration {
	var total time.Duration
	for _, stage := range t.stages {
		total += stage.Duration
	}
	return total
}

// Summary returns a formatted string with timing information for all stages
func (t *Timer) Summary() string {
	// End any running stage
	t.End()

	if len(t.stages) == 0 {
		return ""
	}

	var sb strings.Builder
	total := t.Total()

	sb.WriteString("\n")
	sb.WriteString("Compilation Summary:\n")
	sb.WriteString(strings.Repeat("-", 50) + "\n")

	// Find the longest stage name for alignment
	maxLen := 0
	for _, stage := range t.stages {
		if len(stage.Name) > maxLen {
			maxLen = len(stage.Name)
		}
	}

	// Print each stage with aligned timing and percentage
	for _, stage := range t.stages {
		percentage := float64(stage.Duration) / float64(total) * 100
		padding := strings.Repeat(" ", maxLen-len(stage.Name))

		sb.WriteString(fmt.Sprintf("  %s:%s %8s  (%5.1f%%)\n",
			stage.Name,
			padding,
			formatDuration(stage.Duration),
			percentage,
		))
	}

	sb.WriteString(strings.Repeat("-", 50) + "\n")
	sb.WriteString(fmt.Sprintf("  Total:%s %8s\n",
		strings.Repeat(" ", maxLen-5),
		formatDuration(total),
	))

	return sb.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.3fs", d.Seconds())
}
