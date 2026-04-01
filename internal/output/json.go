package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONFormatter formats data as JSON.
type JSONFormatter struct{}

// Format formats the data as JSON and writes it to the specified writer.
func (jf *JSONFormatter) Format(data []map[string]interface{}, opts FormatOptions) error {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	columns := getColumns(data, opts)

	// Filter and convert columns if necessary
	output := make([]map[string]interface{}, len(data))
	for i, row := range data {
		filtered := make(map[string]interface{})
		for _, col := range columns {
			displayCol := convertColumnName(col, opts.Korean)
			filtered[displayCol] = row[col]
		}
		output[i] = filtered
	}

	// Apply MaxRows limit if specified
	if opts.MaxRows > 0 && len(output) > opts.MaxRows {
		output = output[:opts.MaxRows]
	}

	// Determine if we should pretty print
	isTTY := IsTTY(opts.Writer)

	var jsonBytes []byte
	var err error

	if isTTY {
		// Pretty print with indent for TTY
		jsonBytes, err = json.MarshalIndent(output, "", "  ")
	} else {
		// Compact for pipes
		jsonBytes, err = json.Marshal(output)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	_, err = opts.Writer.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	// Add newline for readability
	fmt.Fprint(opts.Writer, "\n")

	return nil
}
