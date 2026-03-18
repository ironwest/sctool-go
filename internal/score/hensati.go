package score

import (
	"math"

	"sctool-go/internal/data"
	"sctool-go/internal/model"
)

// HensatiInput describes one group's aggregated mean scores for hensati calculation.
type HensatiInput struct {
	GroupVar  string            // value of the grouping variable (e.g. "営業部", "男性")
	MeanScores map[string]float64 // scale_eng → group mean
}

// CalculateHensati computes 偏差値 for each group/scale combination.
// Port of calculate_hensati.R.
//
// tgtSheet is one of: "全体", "age_kubun", "gender"
// benchRows should be loaded from table11.csv (LoadBenchmarks).
// labels should be loaded from nbjsq_label_hensati.csv (LoadLabels).
// groups is a slice of group inputs (one per dept/age/gender bucket).
func CalculateHensati(
	groups []HensatiInput,
	tgtSheet string,
	benchRows []data.BenchmarkRow,
	labels []data.LabelRow,
) []model.HensatiRow {

	// Build label lookup: syakudo_hensati → LabelRow
	labelMap := make(map[string]data.LabelRow, len(labels))
	for _, l := range labels {
		labelMap[l.SyakudoHensati] = l
	}

	// Build label lookup: syakudo_english → LabelRow
	engLabelMap := make(map[string]data.LabelRow, len(labels))
	for _, l := range labels {
		engLabelMap[l.SyakudoEnglish] = l
	}

	// Build benchmark lookup: sheet → syakudo_hensati_name → BenchmarkRow
	// When tgtSheet == "全体", all groups use the "全体" sheet.
	// When tgtSheet == "age_kubun", each group's GroupVar is the sheet name.
	// When tgtSheet == "gender", same.
	benchBySheet := make(map[string]map[string]data.BenchmarkRow)
	for _, b := range benchRows {
		if _, ok := benchBySheet[b.Sheet]; !ok {
			benchBySheet[b.Sheet] = make(map[string]data.BenchmarkRow)
		}
		benchBySheet[b.Sheet][b.SyakudoName] = b
	}

	var result []model.HensatiRow

	for _, grp := range groups {
		// Determine which benchmark sheet to use.
		var sheet string
		switch tgtSheet {
		case "age_kubun", "gender":
			sheet = grp.GroupVar
		default:
			sheet = "全体"
		}

		benchSheet, ok := benchBySheet[sheet]
		if !ok {
			// No benchmark for this group value; skip.
			continue
		}

		for scaleEng, meanScore := range grp.MeanScores {
			if math.IsNaN(meanScore) {
				continue
			}

			// Find the label row for this scale.
			lbl, ok := engLabelMap[scaleEng]
			if !ok {
				continue
			}

			// Find the benchmark row.
			bench, ok := benchSheet[lbl.SyakudoHensati]
			if !ok {
				continue
			}

			var hensati float64
			if bench.SdVal == 0 {
				hensati = 50
			} else {
				hensati = 50 + 10*(meanScore-bench.MeanVal)/bench.SdVal
			}

			hensatiGrp := "全体"
			if tgtSheet != "全体" {
				hensatiGrp = grp.GroupVar
			}

			result = append(result, model.HensatiRow{
				GroupVarValue: grp.GroupVar,
				HensatiGrp:    hensatiGrp,
				ScaleEng:      scaleEng,
				ScaleJapanese: lbl.SyakudoJapanese,
				Value:         meanScore,
				MeanVal:       bench.MeanVal,
				SdVal:         bench.SdVal,
				Hensati:       hensati,
			})
		}
	}

	return result
}

// GroupByVar aggregates ProcessedRecords by a single grouping variable,
// returning a slice of HensatiInput (one per unique group value).
// scaleNames lists which ScaleScores keys to aggregate.
func GroupByVar(records []model.ProcessedRecord, groupVar string, scaleNames []string) []HensatiInput {
	type accumulator struct {
		sums   map[string]float64
		counts map[string]int
		nas    map[string]bool
	}
	byGroup := make(map[string]*accumulator)
	order := []string{} // preserve insertion order

	getGroupValue := func(r model.ProcessedRecord) string {
		switch groupVar {
		case "dept1":
			return r.Dept1
		case "dept2":
			return r.Dept2
		case "gender":
			return r.Gender
		case "age_kubun":
			return r.AgeKubun
		default:
			return ""
		}
	}

	for _, rec := range records {
		gv := getGroupValue(rec)
		if gv == "" {
			continue
		}
		acc, ok := byGroup[gv]
		if !ok {
			acc = &accumulator{
				sums:   make(map[string]float64),
				counts: make(map[string]int),
				nas:    make(map[string]bool),
			}
			byGroup[gv] = acc
			order = append(order, gv)
		}
		for _, name := range scaleNames {
			v, exists := rec.ScaleScores[name]
			if !exists || math.IsNaN(v) {
				// R uses na.rm=TRUE for group means in hensati
				continue
			}
			acc.sums[name] += v
			acc.counts[name]++
		}
	}

	result := make([]HensatiInput, 0, len(byGroup))
	for _, gv := range order {
		acc := byGroup[gv]
		means := make(map[string]float64, len(scaleNames))
		for _, name := range scaleNames {
			cnt := acc.counts[name]
			if cnt == 0 {
				means[name] = math.NaN()
			} else {
				means[name] = acc.sums[name] / float64(cnt)
			}
		}
		result = append(result, HensatiInput{
			GroupVar:   gv,
			MeanScores: means,
		})
	}
	return result
}

// AllScaleNames returns all minor + major scale names used in hensati calculation.
// Matches the R logic: unique(c(nbjsq$syakudo_minor_eng, nbjsq$syakudo_major_eng))
// excluding NA and "outcome".
func AllScaleNames(questions []data.QuestionInfo) []string {
	seen := make(map[string]bool)
	var result []string
	addIfNew := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, q := range questions {
		addIfNew(q.SyakudoMinorEng)
		if q.SyakudoMajorEng != "" && q.SyakudoMajorEng != "outcome" {
			addIfNew(q.SyakudoMajorEng)
		}
	}
	return result
}
