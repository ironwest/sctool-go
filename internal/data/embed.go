package data

import (
	"encoding/csv"
	"embed"
	"io"
	"math"
	"strconv"
)

//go:embed static/*.csv
var staticFS embed.FS

// QuestionInfo holds metadata for one NBJSQ question.
type QuestionInfo struct {
	QNum            int
	IsReverse       bool
	SyakudoMinorEng string // e.g. "w_vol"
	SyakudoMajorEng string // e.g. "w_total", empty when not applicable
}

// BenchmarkRow holds one row from table11.csv (NBJSQ rows only).
type BenchmarkRow struct {
	Sheet       string  // "全体", "男性", "女性", "10代" …
	SyakudoName string  // Japanese scale name, matches nbjsq_label_hensati.syakudo_hensati
	MeanVal     float64
	SdVal       float64
}

// RiskCoef holds one row from risk_coefficients.csv.
type RiskCoef struct {
	Gyousyu  string
	Type     string // "long" or "cross"
	CoefName string
	Coef     float64
	Avg      float64 // math.NaN() when column is empty
}

// LabelRow holds one row from nbjsq_label_hensati.csv.
type LabelRow struct {
	SyakudoEnglish  string
	SyakudoHensati  string // join key matching BenchmarkRow.SyakudoName
	SyakudoJapanese string
}

// LoadQuestions reads nbjsq_question_text.csv from the embedded FS.
// Returns a slice indexed 0..79 corresponding to q1..q80.
func LoadQuestions() ([]QuestionInfo, error) {
	f, err := staticFS.Open("static/nbjsq_question_text.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Read() // header: qnum,qtext,is_reverse,syakudo_minor,syakudo_major,syakudo_minor_eng,syakudo_major_eng

	var result []QuestionInfo
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		qnum, _ := strconv.Atoi(rec[0])
		majorEng := rec[6]
		if majorEng == "outcome" {
			majorEng = ""
		}
		result = append(result, QuestionInfo{
			QNum:            qnum,
			IsReverse:       rec[2] == "1",
			SyakudoMinorEng: rec[5],
			SyakudoMajorEng: majorEng,
		})
	}
	return result, nil
}

// LoadBenchmarks reads table11.csv and returns only NBJSQ rows.
func LoadBenchmarks() ([]BenchmarkRow, error) {
	f, err := staticFS.Open("static/table11.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// table11.csv has long annotation text with commas; LazyQuotes handles edge cases.
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.Read() // header: type,sheet,qtype,尺度分類,尺度名,得点範囲,平均値,標準偏差,注釈

	var result []BenchmarkRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(rec) < 8 || rec[2] != "NBJSQ" {
			continue
		}
		mean, err1 := strconv.ParseFloat(rec[6], 64)
		sd, err2 := strconv.ParseFloat(rec[7], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		result = append(result, BenchmarkRow{
			Sheet:       rec[1],
			SyakudoName: rec[4],
			MeanVal:     mean,
			SdVal:       sd,
		})
	}
	return result, nil
}

// LoadRiskCoefs reads risk_coefficients.csv.
func LoadRiskCoefs() ([]RiskCoef, error) {
	f, err := staticFS.Open("static/risk_coefficients.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Read() // header: gyousyu,type,coefname,coef,avg

	var result []RiskCoef
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		coef, _ := strconv.ParseFloat(rec[3], 64)
		avg := math.NaN()
		if rec[4] != "" {
			avg, _ = strconv.ParseFloat(rec[4], 64)
		}
		result = append(result, RiskCoef{
			Gyousyu:  rec[0],
			Type:     rec[1],
			CoefName: rec[2],
			Coef:     coef,
			Avg:      avg,
		})
	}
	return result, nil
}

// LoadLabels reads nbjsq_label_hensati.csv.
func LoadLabels() ([]LabelRow, error) {
	f, err := staticFS.Open("static/nbjsq_label_hensati.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Read() // header: syakudo_english,syakudo_hensati,syakudo_japanese

	var result []LabelRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, LabelRow{
			SyakudoEnglish:  rec[0],
			SyakudoHensati:  rec[1],
			SyakudoJapanese: rec[2],
		})
	}
	return result, nil
}
