package main

import (
	"fmt"
	"log"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./cmd/inspect <file.xlsx> [sheetname]")
	}

	f, err := excelize.OpenFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	fmt.Printf("Sheets (%d): %v\n\n", len(sheets), sheets)

	sheetName := sheets[0]
	if len(os.Args) > 2 {
		sheetName = os.Args[2]
	}
	fmt.Printf("Inspecting sheet: %q\n\n", sheetName)

	// Show merged cells
	merged, _ := f.GetMergeCells(sheetName)
	fmt.Printf("Merged cells (%d total):\n", len(merged))
	for i, mc := range merged {
		if i < 15 {
			fmt.Printf("  %s -> %s: %q\n", mc.GetStartAxis(), mc.GetEndAxis(), mc.GetCellValue())
		}
	}
	if len(merged) > 15 {
		fmt.Printf("  ... and %d more\n", len(merged)-15)
	}

	// Show first 10 rows with ALL columns
	rows, _ := f.GetRows(sheetName)
	fmt.Printf("\nFirst 10 rows (all columns):\n")
	for i, row := range rows {
		if i >= 10 {
			break
		}
		fmt.Printf("Row %2d (%d cols): ", i+1, len(row))
		for j, cell := range row {
			if cell != "" {
				fmt.Printf("[%s]=%q ", colName(j), truncate(cell, 20))
			}
		}
		fmt.Println()
	}

	// Check outline levels
	fmt.Printf("\nOutline levels (rows 1-30):\n")
	for i := 1; i <= 30; i++ {
		level, _ := f.GetRowOutlineLevel(sheetName, i)
		if level > 0 {
			fmt.Printf("  Row %d: level %d\n", i, level)
		}
	}
}

func colName(idx int) string {
	name := ""
	for idx >= 0 {
		name = string(rune('A'+idx%26)) + name
		idx = idx/26 - 1
	}
	return name
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
