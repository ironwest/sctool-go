package score

import (
	"math"
	"strings"

	"sctool-go/internal/data"
	"sctool-go/internal/model"
)

// AnalysisTableRow is one row in the 偏差値表 (analysis table).
type AnalysisTableRow struct {
	GroupLabel      string             `json:"groupLabel"`
	IsTotal         bool               `json:"isTotal"`
	N               int                `json:"n"`
	IncompleteN     int                `json:"incompleteN"`
	HighStressN     int                `json:"highStressN"`
	HighStressRatio float64            `json:"highStressRatio"`
	TotalRisk       float64            `json:"totalRisk"`
	Hensati         map[string]float64 `json:"hensati"` // Japanese scale name → hensati value
}

// groupAccum accumulates stats for one group.
type groupAccum struct {
	sums   map[string]float64
	counts map[string]int
	n      int
	incN   int
	hsN    int
	// demand/control/support for risk
	dSum, cSum, bSum, fSum     float64
	dCnt, cCnt, bCnt, fCnt int
}

func newGroupAccum(scaleNames []string) *groupAccum {
	acc := &groupAccum{
		sums:   make(map[string]float64, len(scaleNames)),
		counts: make(map[string]int, len(scaleNames)),
	}
	return acc
}

func (acc *groupAccum) addRecord(rec model.ProcessedRecord, scaleNames []string) {
	acc.n++
	for i := 0; i < 80; i++ {
		if !rec.QValid[i] {
			acc.incN++
			break
		}
	}
	if rec.IsHS != nil && *rec.IsHS {
		acc.hsN++
	}
	for _, name := range scaleNames {
		v, exists := rec.ScaleScores[name]
		if !exists || math.IsNaN(v) {
			continue
		}
		acc.sums[name] += v
		acc.counts[name]++
	}
	if !math.IsNaN(rec.Demand) {
		acc.dSum += rec.Demand
		acc.dCnt++
	}
	if !math.IsNaN(rec.Control) {
		acc.cSum += rec.Control
		acc.cCnt++
	}
	if !math.IsNaN(rec.BossSupport) {
		acc.bSum += rec.BossSupport
		acc.bCnt++
	}
	if !math.IsNaN(rec.FellowSupport) {
		acc.fSum += rec.FellowSupport
		acc.fCnt++
	}
}

func (acc *groupAccum) toHensatiInput(key string, scaleNames []string) HensatiInput {
	means := make(map[string]float64, len(scaleNames))
	for _, name := range scaleNames {
		cnt := acc.counts[name]
		if cnt == 0 {
			means[name] = math.NaN()
		} else {
			means[name] = acc.sums[name] / float64(cnt)
		}
	}
	return HensatiInput{GroupVar: key, MeanScores: means}
}

func (acc *groupAccum) toGroupStats(key string) GroupStats {
	return GroupStats{
		GroupValues:   map[string]string{"__key": key},
		Demand:        nanMean(acc.dSum, acc.dCnt),
		Control:       nanMean(acc.cSum, acc.cCnt),
		BossSupport:   nanMean(acc.bSum, acc.bCnt),
		FellowSupport: nanMean(acc.fSum, acc.fCnt),
	}
}

// GetAnalysisHyou builds the full 偏差値表 grouped by groupVar.
//
// groupVar: one of "dept1", "dept2", "dept1_dept2", "age_kubun", "gender"
// longOrCross: "long" or "cross" — which risk score to display
// gyousyu: industry name for the risk coefficient lookup (e.g. "全産業")
func GetAnalysisHyou(
	records []model.ProcessedRecord,
	groupVar string,
	longOrCross string,
	gyousyu string,
	questions []data.QuestionInfo,
	benchmarks []data.BenchmarkRow,
	labels []data.LabelRow,
	riskCoefs []data.RiskCoef,
) []AnalysisTableRow {
	if len(records) == 0 {
		return nil
	}

	scaleNames := AllScaleNames(questions)

	// Determine benchmark sheet
	tgtSheet := "全体"
	if groupVar == "age_kubun" || groupVar == "gender" {
		tgtSheet = groupVar
	}

	keyFn := makeKeyFn(groupVar)

	// Single pass: accumulate stats for each group and 全体
	totalAcc := newGroupAccum(scaleNames)
	byGroup := make(map[string]*groupAccum)
	order := []string{}

	for _, rec := range records {
		totalAcc.addRecord(rec, scaleNames)

		k := keyFn(rec)
		if k == "" {
			continue
		}
		acc, ok := byGroup[k]
		if !ok {
			acc = newGroupAccum(scaleNames)
			byGroup[k] = acc
			order = append(order, k)
		}
		acc.addRecord(rec, scaleNames)
	}

	// Build hensati for all groups
	groupInputs := make([]HensatiInput, 0, len(order))
	for _, k := range order {
		groupInputs = append(groupInputs, byGroup[k].toHensatiInput(k, scaleNames))
	}
	hensatiRows := CalculateHensati(groupInputs, tgtSheet, benchmarks, labels)

	// Build lookup: groupKey → (jpnName → hensati)
	hensatiByGroup := make(map[string]map[string]float64)
	for _, hr := range hensatiRows {
		if hensatiByGroup[hr.GroupVarValue] == nil {
			hensatiByGroup[hr.GroupVarValue] = make(map[string]float64)
		}
		jpn := strings.TrimSpace(hr.ScaleJapanese)
		hensatiByGroup[hr.GroupVarValue][jpn] = hr.Hensati
	}

	// Build hensati for 全体 row
	totHensatiRows := CalculateHensati(
		[]HensatiInput{totalAcc.toHensatiInput("全体", scaleNames)},
		"全体",
		benchmarks,
		labels,
	)
	totHensati := make(map[string]float64)
	for _, hr := range totHensatiRows {
		jpn := strings.TrimSpace(hr.ScaleJapanese)
		totHensati[jpn] = hr.Hensati
	}

	// Build risk scores: 全体 first, then each group
	groupStats := make([]GroupStats, 0, len(order)+1)
	groupStats = append(groupStats, totalAcc.toGroupStats("全体"))
	for _, k := range order {
		groupStats = append(groupStats, byGroup[k].toGroupStats(k))
	}
	riskResults := CalculateRiskScore(groupStats, riskCoefs, gyousyu)

	riskByKey := make(map[string]float64, len(riskResults))
	for _, rr := range riskResults {
		k := rr.GroupValues["__key"]
		var totalRisk float64
		if longOrCross == "long" {
			totalRisk = rr.TotalRiskLong
		} else {
			totalRisk = rr.TotalRiskCross
		}
		riskByKey[k] = totalRisk
	}

	// Assemble final rows: 全体 first, then groups
	result := make([]AnalysisTableRow, 0, len(order)+1)

	totN := totalAcc.n
	var totHSRatio float64
	if totN > 0 {
		totHSRatio = float64(totalAcc.hsN) / float64(totN)
	}
	result = append(result, AnalysisTableRow{
		GroupLabel:      "全体",
		IsTotal:         true,
		N:               totN,
		IncompleteN:     totalAcc.incN,
		HighStressN:     totalAcc.hsN,
		HighStressRatio: totHSRatio,
		TotalRisk:       riskByKey["全体"],
		Hensati:         totHensati,
	})

	for _, k := range order {
		acc := byGroup[k]
		var hsRatio float64
		if acc.n > 0 {
			hsRatio = float64(acc.hsN) / float64(acc.n)
		}
		result = append(result, AnalysisTableRow{
			GroupLabel:      groupLabel(groupVar, k),
			IsTotal:         false,
			N:               acc.n,
			IncompleteN:     acc.incN,
			HighStressN:     acc.hsN,
			HighStressRatio: hsRatio,
			TotalRisk:       riskByKey[k],
			Hensati:         hensatiByGroup[k],
		})
	}

	return result
}

func makeKeyFn(groupVar string) func(model.ProcessedRecord) string {
	switch groupVar {
	case "dept1":
		return func(r model.ProcessedRecord) string { return r.Dept1 }
	case "dept2":
		return func(r model.ProcessedRecord) string { return r.Dept2 }
	case "age_kubun":
		return func(r model.ProcessedRecord) string { return r.AgeKubun }
	case "gender":
		return func(r model.ProcessedRecord) string { return r.Gender }
	case "dept1_dept2":
		return func(r model.ProcessedRecord) string { return r.Dept1 + "|||" + r.Dept2 }
	default:
		return func(r model.ProcessedRecord) string { return "" }
	}
}

func groupLabel(groupVar, key string) string {
	if groupVar == "dept1_dept2" {
		parts := strings.SplitN(key, "|||", 2)
		if len(parts) == 2 {
			return parts[0] + " / " + parts[1]
		}
	}
	return key
}
