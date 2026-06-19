package ingest

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// readXLSX reads every worksheet of an uploaded workbook into title → rows.
// RawCellValue keeps numbers/dates as raw strings (matching the Google Sheets
// UNFORMATTED_VALUE path), so the classifier sees the same data either way.
func readXLSX(r io.Reader) (map[string][][]string, error) {
	f, err := excelize.OpenReader(r, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("gagal membuka workbook: %w", err)
	}
	defer f.Close()

	out := map[string][][]string{}
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			continue // skip unreadable sheet, keep the rest
		}
		out[name] = rows
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("workbook kosong / tidak ada sheet terbaca")
	}
	return out, nil
}
