// Package parser handles CSV file parsing with encoding detection and column mapping.
package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"sctool-go/internal/model"
)

// ColumnMap defines how CSV columns map to the standard fields.
// Each field holds the zero-based index in the CSV, or -1 if not mapped.
type ColumnMap struct {
	EmpID  int
	Age    int
	Gender int
	Dept1  int
	Dept2  int
	Q      [80]int // Q[i] = column index for q(i+1); -1 = unmapped
}

// NewColumnMap returns a ColumnMap with all fields set to -1 (unmapped).
func NewColumnMap() ColumnMap {
	cm := ColumnMap{EmpID: -1, Age: -1, Gender: -1, Dept1: -1, Dept2: -1}
	for i := range cm.Q {
		cm.Q[i] = -1
	}
	return cm
}

// ValueMap defines how raw string values map to numeric scores 1-4.
// Key: column name from CSV; value: map[rawString]int (1-4, 0=missing).
type ValueMap map[string]map[string]int

// ParseResult holds the result of parsing and mapping a CSV file.
type ParseResult struct {
	Headers []string
	Records []model.RawRecord
}

// ParseCSVFile opens a file, auto-detects encoding (UTF-8 or Shift-JIS),
// and returns the raw CSV headers and rows.
func ParseCSVFile(path string) ([]string, [][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}
	return parseCSVBytes(raw)
}

// parseCSVBytes detects encoding and parses CSV from a byte slice.
func parseCSVBytes(raw []byte) ([]string, [][]string, error) {
	// Remove UTF-8 BOM if present.
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}

	// Try UTF-8 first.
	if utf8.Valid(raw) {
		return readCSV(strings.NewReader(string(raw)))
	}

	// Fall back to Shift-JIS.
	decoded, _, err := transform.Bytes(japanese.ShiftJIS.NewDecoder(), raw)
	if err != nil {
		return nil, nil, fmt.Errorf("shift-jis decode: %w", err)
	}
	return readCSV(strings.NewReader(string(decoded)))
}

func readCSV(r io.Reader) ([]string, [][]string, error) {
	cr := csv.NewReader(r)
	cr.LazyQuotes = true
	all, err := cr.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("empty CSV file")
	}
	return all[0], all[1:], nil
}

// ApplyMapping converts raw CSV rows into RawRecords using the provided maps.
// Columns not present in colMap are skipped.
// Values not found in valMap are treated as missing (QValid=false).
func ApplyMapping(headers []string, rows [][]string, colMap ColumnMap, valMap ValueMap) ([]model.RawRecord, error) {
	records := make([]model.RawRecord, 0, len(rows))

	for rowIdx, row := range rows {
		if len(row) == 0 {
			continue
		}
		padRow := func(idx int) string {
			if idx < 0 || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		rec := model.RawRecord{}

		// Attribute columns
		rec.EmpID = padRow(colMap.EmpID)
		if colMap.Age >= 0 {
			if v, err := strconv.Atoi(padRow(colMap.Age)); err == nil {
				rec.Age = v
			}
		}
		rec.Gender = padRow(colMap.Gender)
		rec.Dept1 = padRow(colMap.Dept1)
		rec.Dept2 = padRow(colMap.Dept2)

		// Question columns
		for i, colIdx := range colMap.Q {
			if colIdx < 0 {
				continue // not mapped
			}
			raw := padRow(colIdx)
			if raw == "" {
				continue // treat as missing
			}

			colName := ""
			if colIdx < len(headers) {
				colName = headers[colIdx]
			}

			// Try value map first.
			if vm, ok := valMap[colName]; ok {
				if num, ok2 := vm[raw]; ok2 && num >= 1 && num <= 4 {
					rec.Q[i] = num
					rec.QValid[i] = true
				}
				// If not in valMap, treat as missing.
			} else {
				// No value map for this column: try direct integer parse.
				if num, err := strconv.Atoi(raw); err == nil && num >= 1 && num <= 4 {
					rec.Q[i] = num
					rec.QValid[i] = true
				}
			}
		}

		_ = rowIdx
		records = append(records, rec)
	}

	return records, nil
}

// ParseProcessedCSV reads a previously-processed CSV (output of the R wizard or this app).
// It expects the canonical column set produced by CalculateScores.
func ParseProcessedCSV(path string) ([]model.ProcessedRecord, error) {
	headers, rows, err := ParseCSVFile(path)
	if err != nil {
		return nil, err
	}

	// Build a name→index map for fast lookup.
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[strings.TrimSpace(h)] = i
	}

	get := func(row []string, name string) string {
		i, ok := colIdx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	getFloat := func(row []string, name string) float64 {
		s := get(row, name)
		if s == "" || s == "NA" {
			return float64(0) // caller should check separately
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	getInt := func(row []string, name string) int {
		s := get(row, name)
		v, _ := strconv.Atoi(s)
		return v
	}

	records := make([]model.ProcessedRecord, 0, len(rows))
	for i, row := range rows {
		rec := model.ProcessedRecord{
			TempID:   i + 1,
			EmpID:    get(row, "empid"),
			Age:      getInt(row, "age"),
			AgeKubun: get(row, "age_kubun"),
			Gender:   get(row, "gender"),
			Dept1:    get(row, "dept1"),
			Dept2:    get(row, "dept2"),
		}

		// Raw question answers
		for q := 1; q <= 80; q++ {
			name := fmt.Sprintf("q%d", q)
			s := get(row, name)
			if s != "" && s != "NA" {
				if v, err := strconv.Atoi(s); err == nil {
					rec.Q[q-1] = v
					rec.QValid[q-1] = true
				}
			}
		}

		// Question scores
		for q := 1; q <= 80; q++ {
			name := fmt.Sprintf("q%d_score", q)
			rec.QScore[q-1] = getFloat(row, name)
		}

		// Scale scores
		scaleNames := []string{
			"w_total", "w_vol", "w_qua", "w_hutan", "w_tai", "w_env",
			"w_jyoutyo", "w_yakuwarikat", "w_wsbneg",
			"s_total", "s_control", "s_ginou", "s_tek", "s_igi",
			"s_yakuwarimei", "s_growth",
			"b_total", "b_bosssupp", "b_collsupp", "b_ecoreward",
			"b_sonreward", "b_antreward", "b_bossleader", "b_bossfair",
			"b_homeru", "b_sippai",
			"j_total", "j_keiei", "j_change", "j_kojin", "j_jinji",
			"j_dei", "j_carrier", "j_wsbpos",
			"p_total", "p_kakki", "p_iraira", "p_hirou", "p_huan", "p_utu",
			"na_cc", "na_famsupp", "na_workmanzoku", "na_kateimanzoku",
			"o_harass", "o_sc", "o_we",
		}
		rec.ScaleScores = make(map[string]float64, len(scaleNames))
		for _, name := range scaleNames {
			rec.ScaleScores[name] = getFloat(row, name)
		}

		// High-stress fields
		rec.AreaA = getFloat(row, "A")
		rec.AreaB = getFloat(row, "B")
		rec.AreaC = getFloat(row, "C")
		isHSStr := get(row, "is_hs")
		if isHSStr == "TRUE" {
			b := true
			rec.IsHS = &b
		} else if isHSStr == "FALSE" {
			b := false
			rec.IsHS = &b
		}

		// Demand/control/support
		rec.Demand = getFloat(row, "demand")
		rec.Control = getFloat(row, "control")
		rec.BossSupport = getFloat(row, "boss_support")
		rec.FellowSupport = getFloat(row, "fellow_support")

		records = append(records, rec)
	}
	return records, nil
}
