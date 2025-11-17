package timing

import (
	"strings"
	"testing"
	"time"
)

func TestTimer_StartEnd(t *testing.T) {
	timer := NewTimer()

	timer.Start("Stage 1")
	time.Sleep(10 * time.Millisecond)
	timer.End()

	if len(timer.stages) != 1 {
		t.Errorf("expected 1 stage, got %d", len(timer.stages))
	}

	if timer.stages[0].Name != "Stage 1" {
		t.Errorf("expected stage name 'Stage 1', got '%s'", timer.stages[0].Name)
	}

	if timer.stages[0].Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", timer.stages[0].Duration)
	}
}

func TestTimer_MultipleStages(t *testing.T) {
	timer := NewTimer()

	stages := []string{"Lexer", "Parser", "Codegen"}
	for _, stage := range stages {
		timer.Start(stage)
		time.Sleep(5 * time.Millisecond)
		timer.End()
	}

	if len(timer.stages) != len(stages) {
		t.Errorf("expected %d stages, got %d", len(stages), len(timer.stages))
	}

	for i, stage := range stages {
		if timer.stages[i].Name != stage {
			t.Errorf("stage %d: expected '%s', got '%s'", i, stage, timer.stages[i].Name)
		}
	}
}

func TestTimer_AutoEnd(t *testing.T) {
	timer := NewTimer()

	// Starting a new stage should automatically end the previous one
	timer.Start("Stage 1")
	time.Sleep(5 * time.Millisecond)

	timer.Start("Stage 2")
	time.Sleep(5 * time.Millisecond)
	timer.End()

	if len(timer.stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(timer.stages))
	}

	if timer.stages[0].Name != "Stage 1" {
		t.Errorf("expected first stage 'Stage 1', got '%s'", timer.stages[0].Name)
	}

	if timer.stages[1].Name != "Stage 2" {
		t.Errorf("expected second stage 'Stage 2', got '%s'", timer.stages[1].Name)
	}
}

func TestTimer_Total(t *testing.T) {
	timer := NewTimer()

	timer.Start("Stage 1")
	time.Sleep(10 * time.Millisecond)
	timer.End()

	timer.Start("Stage 2")
	time.Sleep(10 * time.Millisecond)
	timer.End()

	total := timer.Total()
	if total < 20*time.Millisecond {
		t.Errorf("expected total >= 20ms, got %v", total)
	}
}

func TestTimer_Summary(t *testing.T) {
	timer := NewTimer()

	timer.Start("Lexer")
	time.Sleep(5 * time.Millisecond)
	timer.End()

	timer.Start("Parser")
	time.Sleep(10 * time.Millisecond)
	timer.End()

	summary := timer.Summary()

	// Check that summary contains expected elements
	if !strings.Contains(summary, "Compilation Summary:") {
		t.Error("summary should contain title")
	}

	if !strings.Contains(summary, "Lexer") {
		t.Error("summary should contain 'Lexer'")
	}

	if !strings.Contains(summary, "Parser") {
		t.Error("summary should contain 'Parser'")
	}

	if !strings.Contains(summary, "Total:") {
		t.Error("summary should contain 'Total:'")
	}

	// Check that percentages are present
	if !strings.Contains(summary, "%") {
		t.Error("summary should contain percentages")
	}
}

func TestTimer_EmptySummary(t *testing.T) {
	timer := NewTimer()
	summary := timer.Summary()

	if summary != "" {
		t.Errorf("expected empty summary for timer with no stages, got '%s'", summary)
	}
}

func TestTimer_EndWithoutStart(t *testing.T) {
	timer := NewTimer()
	timer.End() // Should not panic

	if len(timer.stages) != 0 {
		t.Errorf("expected 0 stages after calling End without Start, got %d", len(timer.stages))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"nanoseconds", 500 * time.Nanosecond, "ns"},
		{"microseconds", 500 * time.Microsecond, "µs"},
		{"milliseconds", 50 * time.Millisecond, "ms"},
		{"seconds", 2 * time.Second, "s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatDuration(%v) = %s, expected to contain '%s'",
					tt.duration, result, tt.contains)
			}
		})
	}
}

func TestTimer_SummaryAlignment(t *testing.T) {
	timer := NewTimer()

	// Add stages with different name lengths
	timer.Start("A")
	timer.End()

	timer.Start("Very Long Stage Name")
	timer.End()

	summary := timer.Summary()

	// Split into lines and check alignment
	lines := strings.Split(summary, "\n")
	var stageLine1, stageLine2 string
	for _, line := range lines {
		if strings.Contains(line, "A:") {
			stageLine1 = line
		}
		if strings.Contains(line, "Very Long Stage Name:") {
			stageLine2 = line
		}
	}

	if stageLine1 == "" || stageLine2 == "" {
		t.Error("could not find stage lines in summary")
	}

	// Both lines should have similar structure (name, duration, percentage)
	// We just check they both contain percentage signs for now
	if !strings.Contains(stageLine1, "%") || !strings.Contains(stageLine2, "%") {
		t.Error("stage lines should contain percentages")
	}
}
