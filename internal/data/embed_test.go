package data_test

import (
	"math"
	"testing"

	"sctool-go/internal/data"
)

func TestLoadQuestions(t *testing.T) {
	questions, err := data.LoadQuestions()
	if err != nil {
		t.Fatalf("LoadQuestions: %v", err)
	}
	if len(questions) != 80 {
		t.Errorf("want 80 questions, got %d", len(questions))
	}

	// q8 should be reverse-scored (仕事のコントロール)
	q8 := questions[7] // 0-indexed
	if q8.QNum != 8 {
		t.Errorf("questions[7].QNum = %d, want 8", q8.QNum)
	}
	if !q8.IsReverse {
		t.Error("q8 should be reverse-scored")
	}
	if q8.SyakudoMinorEng != "s_control" {
		t.Errorf("q8 minor scale = %q, want \"s_control\"", q8.SyakudoMinorEng)
	}

	// q1 should NOT be reverse-scored
	q1 := questions[0]
	if q1.IsReverse {
		t.Error("q1 should not be reverse-scored")
	}
	if q1.SyakudoMinorEng != "w_vol" {
		t.Errorf("q1 minor scale = %q, want \"w_vol\"", q1.SyakudoMinorEng)
	}
}

func TestLoadBenchmarks(t *testing.T) {
	rows, err := data.LoadBenchmarks()
	if err != nil {
		t.Fatalf("LoadBenchmarks: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no benchmark rows loaded")
	}

	// Check that 全体 sheet exists
	found := false
	for _, r := range rows {
		if r.Sheet == "全体" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no rows for sheet 全体")
	}

	// Spot-check: 全体 / 心理的な仕事の負担（量） should have reasonable mean/SD
	for _, r := range rows {
		if r.Sheet == "全体" && r.SyakudoName == "心理的な仕事の負担（量）" {
			if r.MeanVal <= 0 || r.SdVal <= 0 {
				t.Errorf("unexpected mean/sd: %v", r)
			}
			t.Logf("全体 / 仕事の負担（量）: mean=%.4f sd=%.4f", r.MeanVal, r.SdVal)
		}
	}
}

func TestLoadRiskCoefs(t *testing.T) {
	coefs, err := data.LoadRiskCoefs()
	if err != nil {
		t.Fatalf("LoadRiskCoefs: %v", err)
	}
	if len(coefs) == 0 {
		t.Fatal("no risk coef rows loaded")
	}

	// 全産業 should have demand/control/boss_support/fellow_support for both long and cross
	requiredCoefs := map[string]bool{
		"全産業:long:demand":         false,
		"全産業:long:control":        false,
		"全産業:long:boss_support":   false,
		"全産業:long:fellow_support": false,
		"全産業:cross:demand":        false,
	}
	for _, rc := range coefs {
		key := rc.Gyousyu + ":" + rc.Type + ":" + rc.CoefName
		if _, ok := requiredCoefs[key]; ok {
			requiredCoefs[key] = true
		}
		// avg should be NaN or a valid positive number
		if !math.IsNaN(rc.Avg) && rc.Avg < 0 {
			t.Errorf("negative avg for %v", rc)
		}
	}
	for k, found := range requiredCoefs {
		if !found {
			t.Errorf("required coef not found: %s", k)
		}
	}
}

func TestLoadLabels(t *testing.T) {
	labels, err := data.LoadLabels()
	if err != nil {
		t.Fatalf("LoadLabels: %v", err)
	}
	if len(labels) == 0 {
		t.Fatal("no label rows loaded")
	}

	// w_vol should map to 心理的な仕事の負担（量）
	for _, l := range labels {
		if l.SyakudoEnglish == "w_vol" {
			if l.SyakudoHensati != "心理的な仕事の負担（量）" {
				t.Errorf("w_vol hensati name = %q", l.SyakudoHensati)
			}
			t.Logf("w_vol: hensati=%q japanese=%q", l.SyakudoHensati, l.SyakudoJapanese)
			return
		}
	}
	t.Error("w_vol not found in labels")
}
