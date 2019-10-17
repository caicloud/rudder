package main

import "fmt"

func printTable(table [][]string) {
	var widths []int
	for _, row := range table {
		for i, col := range row {
			if i >= len(widths) {
				widths = append(widths, len(col))
			} else if widths[i] < len(col) {
				widths[i] = len(col)
			}
		}
	}
	for _, row := range table {
		for i, col := range row {
			width := widths[i] + 3
			if i < len(widths)-1 {
				fmt.Printf(fmt.Sprintf("%%-%ds", width), col)
			} else {
				fmt.Printf("%s", col)
			}
		}
		fmt.Println()
	}
}
