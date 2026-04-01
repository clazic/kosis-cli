package output

import (
	"fmt"
	"os"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// XLSXFormatter formats data as XLSX (Excel) with styling.
type XLSXFormatter struct{}

// Format formats the data as XLSX and writes it to a file.
// opts.Writer must be an *os.File for this formatter.
func (xf *XLSXFormatter) Format(data []map[string]interface{}, opts FormatOptions) error {
	// Validate that Writer is an *os.File
	file, ok := opts.Writer.(*os.File)
	if !ok {
		return fmt.Errorf("XLSX 포맷터는 파일 기반 출력만 지원합니다")
	}

	// Create a new workbook
	wb := excelize.NewFile()
	defer wb.Close()

	// Remove default sheet
	wb.DeleteSheet("Sheet1")

	// Create new sheet with default name
	sheetName := "KOSIS Data"
	if _, err := wb.NewSheet(sheetName); err != nil {
		return fmt.Errorf("시트 생성 실패: %w", err)
	}

	columns := getColumns(data, opts)
	if len(columns) == 0 {
		// Empty data, just save the file
		return wb.SaveAs(file.Name())
	}

	// Write header row
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = convertColumnName(col, true)
	}

	// Write headers with styling
	headerStyle, err := wb.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"ADD8E6"}, // Light blue
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("헤더 스타일 생성 실패: %w", err)
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		if err := wb.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("헤더 셀 설정 실패: %w", err)
		}
		if err := wb.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("헤더 셀 스타일 적용 실패: %w", err)
		}
	}

	// Number style for DT column (data)
	customNumFmt := "0.00"
	numStyle, err := wb.NewStyle(&excelize.Style{
		CustomNumFmt: &customNumFmt,
	})
	if err != nil {
		return fmt.Errorf("숫자 스타일 생성 실패: %w", err)
	}

	// Write data rows
	rowsToWrite := len(data)
	if opts.MaxRows > 0 && rowsToWrite > opts.MaxRows {
		rowsToWrite = opts.MaxRows
	}

	for i := 0; i < rowsToWrite; i++ {
		for j, col := range columns {
			cellRef := fmt.Sprintf("%s%d", string(rune('A'+j)), i+2)
			value := data[i][col]

			// Try to parse as number for DT column
			if col == "DT" && value != nil {
				if str, ok := value.(string); ok {
					if num, err := strconv.ParseFloat(str, 64); err == nil {
						if err := wb.SetCellValue(sheetName, cellRef, num); err != nil {
							return fmt.Errorf("데이터 셀 설정 실패: %w", err)
						}
						if err := wb.SetCellStyle(sheetName, cellRef, cellRef, numStyle); err != nil {
							return fmt.Errorf("숫자 스타일 적용 실패: %w", err)
						}
						continue
					}
				} else if f, ok := value.(float64); ok {
					if err := wb.SetCellValue(sheetName, cellRef, f); err != nil {
						return fmt.Errorf("데이터 셀 설정 실패: %w", err)
					}
					if err := wb.SetCellStyle(sheetName, cellRef, cellRef, numStyle); err != nil {
						return fmt.Errorf("숫자 스타일 적용 실패: %w", err)
					}
					continue
				}
			}

			// Default: set as string
			if err := wb.SetCellValue(sheetName, cellRef, formatValue(value)); err != nil {
				return fmt.Errorf("데이터 셀 설정 실패: %w", err)
			}
		}
	}

	// Auto-adjust column widths
	for i := range columns {
		col := string(rune('A' + i))
		if err := wb.SetColWidth(sheetName, col, col, 15); err != nil {
			return fmt.Errorf("컬럼 폭 조정 실패: %w", err)
		}
	}

	// Save the file
	if err := wb.SaveAs(file.Name()); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	return nil
}
