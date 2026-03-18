package score_test

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"sctool-go/internal/data"
	"sctool-go/internal/model"
	"sctool-go/internal/score"
)

// loadProcessedCSV reads the processed CSV produced by R and returns the expected values
// alongside reconstructed RawRecords (from the q1..q80 columns in that file).
func loadProcessedCSV(t *testing.T, path string) ([]model.RawRecord, []model.ProcessedRecord) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read csv %s: %v", path, err)
	}
	if len(rows) < 2 {
		t.Fatalf("csv too short")
	}

	header := rows[0]
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	get := func(row []string, name string) string {
		i, ok := colIdx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	getInt := func(row []string, name string) int {
		v, _ := strconv.Atoi(get(row, name))
		return v
	}
	getFloat := func(row []string, name string) float64 {
		s := get(row, name)
		if s == "" || s == "NA" {
			return math.NaN()
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	raws := make([]model.RawRecord, 0, len(rows)-1)
	expected := make([]model.ProcessedRecord, 0, len(rows)-1)

	for i, row := range rows[1:] {
		raw := model.RawRecord{
			EmpID:  get(row, "empid"),
			Age:    getInt(row, "age"),
			Gender: get(row, "gender"),
			Dept1:  get(row, "dept1"),
			Dept2:  get(row, "dept2"),
		}
		for q := 1; q <= 80; q++ {
			s := get(row, fmt.Sprintf("q%d", q))
			if s != "" && s != "NA" {
				if v, err := strconv.Atoi(s); err == nil && v >= 1 && v <= 4 {
					raw.Q[q-1] = v
					raw.QValid[q-1] = true
				}
			}
		}

		exp := model.ProcessedRecord{
			TempID:      i + 1,
			AgeKubun:    get(row, "age_kubun"),
			ScaleScores: make(map[string]float64),
		}

		// Question scores
		for q := 1; q <= 80; q++ {
			exp.QScore[q-1] = getFloat(row, fmt.Sprintf("q%d_score", q))
		}

		// Scale scores
		for _, name := range []string{
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
		} {
			exp.ScaleScores[name] = getFloat(row, name)
		}

		exp.AreaA = getFloat(row, "A")
		exp.AreaB = getFloat(row, "B")
		exp.AreaC = getFloat(row, "C")
		isHSStr := get(row, "is_hs")
		if isHSStr == "TRUE" {
			b := true
			exp.IsHS = &b
		} else if isHSStr == "FALSE" {
			b := false
			exp.IsHS = &b
		}

		exp.Demand = getFloat(row, "demand")
		exp.Control = getFloat(row, "control")
		exp.BossSupport = getFloat(row, "boss_support")
		exp.FellowSupport = getFloat(row, "fellow_support")

		raws = append(raws, raw)
		expected = append(expected, exp)
	}

	return raws, expected
}

func TestCalculateScores_AgainstROutput(t *testing.T) {
	questions, err := data.LoadQuestions()
	if err != nil {
		t.Fatalf("LoadQuestions: %v", err)
	}

	raws, expected := loadProcessedCSV(t, "../../testdata/processed_nbjsq_dummy_data1_alpha.csv")

	got := score.CalculateScores(raws, questions)

	if len(got) != len(expected) {
		t.Fatalf("record count: got %d, want %d", len(got), len(expected))
	}

	const tol = 1e-9 // floating-point comparison tolerance

	errCount := 0
	maxErr := 5 // stop reporting after 5 errors per category

	// Check question scores
	for i, rec := range got {
		for q := 0; q < 80; q++ {
			exp := expected[i].QScore[q]
			got_ := rec.QScore[q]
			if math.IsNaN(exp) && math.IsNaN(got_) {
				continue
			}
			if math.IsNaN(exp) != math.IsNaN(got_) || math.Abs(exp-got_) > tol {
				if errCount < maxErr {
					t.Errorf("row %d q%d_score: got %.6f, want %.6f", i+1, q+1, got_, exp)
				}
				errCount++
			}
		}
	}

	// Check scale scores
	scaleNames := []string{
		"w_total", "w_vol", "w_qua", "w_hutan", "w_tai", "w_env",
		"s_total", "s_control", "s_ginou", "s_tek", "s_igi",
		"b_total", "b_bosssupp", "b_collsupp",
		"j_total",
		"p_total", "p_kakki", "p_iraira", "p_hirou", "p_huan", "p_utu",
	}
	for _, name := range scaleNames {
		scalErrCount := 0
		for i, rec := range got {
			exp := expected[i].ScaleScores[name]
			got_ := rec.ScaleScores[name]
			if math.IsNaN(exp) && math.IsNaN(got_) {
				continue
			}
			if math.IsNaN(exp) != math.IsNaN(got_) || math.Abs(exp-got_) > tol {
				if scalErrCount < 3 {
					t.Errorf("row %d scale %s: got %.6f, want %.6f", i+1, name, got_, exp)
				}
				scalErrCount++
			}
		}
		if scalErrCount > 0 {
			t.Errorf("scale %s: %d mismatches total", name, scalErrCount)
		}
	}

	// Check high-stress areas A, B, C and IsHS
	hsErrCount := 0
	for i, rec := range got {
		exp := expected[i]

		if !nanEq(rec.AreaA, exp.AreaA, tol) && hsErrCount < maxErr {
			t.Errorf("row %d AreaA: got %.2f, want %.2f", i+1, rec.AreaA, exp.AreaA)
			hsErrCount++
		}
		if !nanEq(rec.AreaB, exp.AreaB, tol) && hsErrCount < maxErr {
			t.Errorf("row %d AreaB: got %.2f, want %.2f", i+1, rec.AreaB, exp.AreaB)
			hsErrCount++
		}
		if !nanEq(rec.AreaC, exp.AreaC, tol) && hsErrCount < maxErr {
			t.Errorf("row %d AreaC: got %.2f, want %.2f", i+1, rec.AreaC, exp.AreaC)
			hsErrCount++
		}

		expIsHS := exp.IsHS
		gotIsHS := rec.IsHS
		if (expIsHS == nil) != (gotIsHS == nil) {
			if hsErrCount < maxErr {
				t.Errorf("row %d is_hs nil mismatch: got %v, want %v", i+1, gotIsHS, expIsHS)
			}
			hsErrCount++
		} else if expIsHS != nil && gotIsHS != nil && *expIsHS != *gotIsHS {
			if hsErrCount < maxErr {
				t.Errorf("row %d is_hs: got %v, want %v (A=%.0f B=%.0f C=%.0f)",
					i+1, *gotIsHS, *expIsHS, rec.AreaA, rec.AreaB, rec.AreaC)
			}
			hsErrCount++
		}
	}

	// Check demand/control/support
	dcsErrCount := 0
	for i, rec := range got {
		exp := expected[i]
		fields := [][2]float64{
			{rec.Demand, exp.Demand},
			{rec.Control, exp.Control},
			{rec.BossSupport, exp.BossSupport},
			{rec.FellowSupport, exp.FellowSupport},
		}
		names := []string{"demand", "control", "boss_support", "fellow_support"}
		for j, f := range fields {
			if !nanEq(f[0], f[1], tol) && dcsErrCount < maxErr {
				t.Errorf("row %d %s: got %.2f, want %.2f", i+1, names[j], f[0], f[1])
				dcsErrCount++
			}
		}
	}

	// Check age_kubun
	for i, rec := range got {
		if rec.AgeKubun != expected[i].AgeKubun && expected[i].AgeKubun != "" {
			t.Errorf("row %d age_kubun: got %q, want %q", i+1, rec.AgeKubun, expected[i].AgeKubun)
		}
	}

	if errCount > 0 {
		t.Errorf("total question score mismatches: %d", errCount)
	}
}

func TestCalculateScores_Counts(t *testing.T) {
	questions, err := data.LoadQuestions()
	if err != nil {
		t.Fatalf("LoadQuestions: %v", err)
	}

	raws, expected := loadProcessedCSV(t, "../../testdata/processed_nbjsq_dummy_data1_alpha.csv")
	got := score.CalculateScores(raws, questions)

	// Count high-stress in expected vs got
	expHS, gotHS := 0, 0
	for i := range got {
		if expected[i].IsHS != nil && *expected[i].IsHS {
			expHS++
		}
		if got[i].IsHS != nil && *got[i].IsHS {
			gotHS++
		}
	}
	if expHS != gotHS {
		t.Errorf("high-stress count: got %d, want %d", gotHS, expHS)
	} else {
		t.Logf("high-stress count matches: %d/%d", gotHS, len(got))
	}
}

func nanEq(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}
	return math.Abs(a-b) <= tol
}
