package chart

import (
	"fmt"
	"os"
	"strings"

	"github.com/guptarohit/asciigraph"
)

func renderTerminal(seriesList []Series, opts Options) error {
	width := opts.Width
	if width <= 0 {
		width = 80
	}
	height := opts.Height
	if height <= 0 {
		height = 15
	}

	for i, s := range seriesList {
		if len(s.Values) == 0 {
			continue
		}

		// Downsample if data is too large for terminal display
		if len(s.Values) > 200 {
			fmt.Fprintf(os.Stderr, "참고: 데이터가 %d건으로 많아 200건으로 다운샘플링하여 표시합니다.\n", len(s.Values))
			s = downsample(s, 200)
		}

		title := opts.Title
		if title == "" && s.Name != "" {
			title = s.Name
		}
		if title != "" && len(seriesList) > 1 {
			title = fmt.Sprintf("%s - %s", opts.Title, s.Name)
		}

		caption := buildCaption(s.Labels)

		plotOpts := []asciigraph.Option{
			asciigraph.Height(height),
			asciigraph.Width(width),
		}
		if title != "" {
			plotOpts = append(plotOpts, asciigraph.Caption(title))
		}

		graph := asciigraph.Plot(s.Values, plotOpts...)
		fmt.Println(graph)

		if caption != "" {
			fmt.Println(caption)
		}

		if i < len(seriesList)-1 {
			fmt.Println()
		}
	}

	return nil
}

// buildCaption creates an x-axis label line from labels.
func buildCaption(labels []string) string {
	if len(labels) == 0 {
		return ""
	}

	// Show up to 10 labels evenly spaced
	maxLabels := 10
	if len(labels) <= maxLabels {
		return "          " + strings.Join(labels, "  ")
	}

	step := len(labels) / (maxLabels - 1)
	var selected []string
	for i := 0; i < len(labels); i += step {
		selected = append(selected, labels[i])
	}
	// Always include the last label
	if selected[len(selected)-1] != labels[len(labels)-1] {
		selected = append(selected, labels[len(labels)-1])
	}
	return "          " + strings.Join(selected, "  ")
}

// downsample reduces a series to the target number of points by evenly sampling.
func downsample(s Series, target int) Series {
	n := len(s.Values)
	if n <= target {
		return s
	}
	result := Series{Name: s.Name}
	step := float64(n-1) / float64(target-1)
	for i := 0; i < target; i++ {
		idx := int(float64(i)*step + 0.5)
		if idx >= n {
			idx = n - 1
		}
		result.Values = append(result.Values, s.Values[idx])
		if idx < len(s.Labels) {
			result.Labels = append(result.Labels, s.Labels[idx])
		}
	}
	return result
}
