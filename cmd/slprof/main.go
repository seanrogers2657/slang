package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

type Sample struct {
	Time time.Duration
	RSS  float64 // Memory in MB
	CPU  float64 // CPU percentage
}

func main() {
	app := &cli.App{
		Name:  "slprof",
		Usage: "Profile memory and CPU usage of a command",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "interval",
				Aliases: []string{"i"},
				Value:   50,
				Usage:   "Sampling interval in milliseconds",
			},
			&cli.IntFlag{
				Name:    "height",
				Aliases: []string{"H"},
				Value:   12,
				Usage:   "Chart height in lines",
			},
			&cli.BoolFlag{
				Name:  "csv",
				Usage: "Output raw CSV data instead of charts",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("usage: slprof [options] <command> [args...]")
			}

			args := c.Args().Slice()
			interval := time.Duration(c.Int("interval")) * time.Millisecond
			height := c.Int("height")
			csvOutput := c.Bool("csv")

			samples, err := profileCommand(args, interval)
			if err != nil {
				return err
			}

			if csvOutput {
				printCSV(samples)
			} else {
				printCharts(samples, height)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func profileCommand(args []string, interval time.Duration) ([]Sample, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	pid := cmd.Process.Pid
	var samples []Sample
	var mu sync.Mutex
	done := make(chan struct{})
	start := time.Now()

	// Sampling goroutine
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rss, cpu, err := sampleProcess(pid)
				if err != nil {
					continue // Process may have exited
				}
				mu.Lock()
				samples = append(samples, Sample{
					Time: time.Since(start),
					RSS:  rss,
					CPU:  cpu,
				})
				mu.Unlock()
			}
		}
	}()

	// Wait for command to finish
	err := cmd.Wait()
	close(done)

	if err != nil {
		// Don't treat non-zero exit as error for profiling purposes
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, fmt.Errorf("command failed: %w", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	return samples, nil
}

// sampleProcess gets RSS (in MB) and CPU% for a process on macOS
func sampleProcess(pid int) (rss float64, cpu float64, err error) {
	cmd := exec.Command("ps", "-o", "rss=,pcpu=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			rssKB, _ := strconv.ParseFloat(fields[0], 64)
			rss = rssKB / 1024.0
			cpu, _ = strconv.ParseFloat(fields[1], 64)
		}
	}

	return rss, cpu, nil
}

func printCSV(samples []Sample) {
	fmt.Println("time_ms,rss_mb,cpu_percent")
	for _, s := range samples {
		fmt.Printf("%.0f,%.2f,%.1f\n", float64(s.Time.Milliseconds()), s.RSS, s.CPU)
	}
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 40 {
		return 80 // default
	}
	return width
}

func printCharts(samples []Sample, height int) {
	if len(samples) == 0 {
		fmt.Println("No samples collected (process too short?)")
		return
	}

	termWidth := getTerminalWidth()

	// Extract data series
	rssData := make([]float64, len(samples))
	cpuData := make([]float64, len(samples))
	var maxRSS, maxCPU float64
	var totalRSS, totalCPU float64

	for i, s := range samples {
		rssData[i] = s.RSS
		cpuData[i] = s.CPU
		if s.RSS > maxRSS {
			maxRSS = s.RSS
		}
		if s.CPU > maxCPU {
			maxCPU = s.CPU
		}
		totalRSS += s.RSS
		totalCPU += s.CPU
	}

	duration := samples[len(samples)-1].Time

	// Print header
	fmt.Println()
	headerLine := strings.Repeat("═", termWidth-2)
	fmt.Println(headerLine)
	fmt.Printf("  Duration: %v | Samples: %d\n", duration.Round(time.Millisecond), len(samples))
	fmt.Println(headerLine)

	// Memory chart
	fmt.Println()
	fmt.Printf("  Memory (RSS) - Max: %.1f MB, Avg: %.1f MB\n", maxRSS, totalRSS/float64(len(samples)))
	fmt.Println("  " + strings.Repeat("─", termWidth-4))
	drawChart(rssData, height, termWidth, "MB")

	// CPU chart
	fmt.Println()
	fmt.Printf("  CPU Usage - Max: %.1f%%, Avg: %.1f%%\n", maxCPU, totalCPU/float64(len(samples)))
	fmt.Println("  " + strings.Repeat("─", termWidth-4))
	drawChart(cpuData, height, termWidth, "%")

	fmt.Println()
}

func drawChart(data []float64, height, termWidth int, unit string) {
	if len(data) == 0 {
		return
	}

	// Find min/max
	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Handle flat line case
	if maxVal == minVal {
		maxVal = minVal + 1
	}

	// Calculate label width for y-axis
	maxLabel := formatValue(maxVal)
	labelWidth := len(maxLabel) + 1

	// Chart area dimensions
	chartWidth := termWidth - labelWidth - 4 // 4 for margins and axis
	if chartWidth < 20 {
		chartWidth = 20
	}

	// Resample data to fit chart width
	resampled := resampleData(data, chartWidth)

	// Build the chart grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, chartWidth)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Plot points
	for x, val := range resampled {
		// Map value to row (0 = top = max, height-1 = bottom = min)
		normalized := (val - minVal) / (maxVal - minVal)
		row := height - 1 - int(normalized*float64(height-1))
		if row < 0 {
			row = 0
		}
		if row >= height {
			row = height - 1
		}

		// Draw vertical line from bottom to this point
		for r := height - 1; r >= row; r-- {
			if r == row {
				grid[r][x] = '█'
			} else {
				grid[r][x] = '│'
			}
		}
	}

	// Print chart with y-axis labels
	for row := 0; row < height; row++ {
		// Calculate value for this row
		rowVal := maxVal - (float64(row)/float64(height-1))*(maxVal-minVal)

		// Print label on first, middle, and last rows
		var label string
		if row == 0 {
			label = formatValue(maxVal)
		} else if row == height-1 {
			label = formatValue(minVal)
		} else if row == height/2 {
			label = formatValue((maxVal + minVal) / 2)
		} else {
			label = strings.Repeat(" ", len(maxLabel))
		}
		_ = rowVal // suppress unused warning

		fmt.Printf("  %*s │%s\n", labelWidth-1, label, string(grid[row]))
	}

	// X-axis
	fmt.Printf("  %s └%s\n", strings.Repeat(" ", labelWidth-1), strings.Repeat("─", chartWidth))

	// Time labels
	if len(data) > 0 {
		fmt.Printf("  %s  0%s%s\n",
			strings.Repeat(" ", labelWidth-1),
			strings.Repeat(" ", chartWidth-4),
			unit)
	}
}

func resampleData(data []float64, targetLen int) []float64 {
	if len(data) <= targetLen {
		// Stretch data to fill width
		result := make([]float64, targetLen)
		for i := 0; i < targetLen; i++ {
			srcIdx := float64(i) * float64(len(data)-1) / float64(targetLen-1)
			idx := int(srcIdx)
			if idx >= len(data)-1 {
				result[i] = data[len(data)-1]
			} else {
				// Linear interpolation
				frac := srcIdx - float64(idx)
				result[i] = data[idx]*(1-frac) + data[idx+1]*frac
			}
		}
		return result
	}

	// Compress data by averaging
	result := make([]float64, targetLen)
	bucketSize := float64(len(data)) / float64(targetLen)

	for i := 0; i < targetLen; i++ {
		start := int(float64(i) * bucketSize)
		end := int(float64(i+1) * bucketSize)
		if end > len(data) {
			end = len(data)
		}
		if start >= end {
			start = end - 1
		}

		sum := 0.0
		for j := start; j < end; j++ {
			sum += data[j]
		}
		result[i] = sum / float64(end-start)
	}

	return result
}

func formatValue(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.0f", v)
	} else if v >= 100 {
		return fmt.Sprintf("%.0f", v)
	} else if v >= 10 {
		return fmt.Sprintf("%.1f", v)
	} else if v >= 1 {
		return fmt.Sprintf("%.1f", v)
	} else {
		return fmt.Sprintf("%.2f", math.Max(0, v))
	}
}
