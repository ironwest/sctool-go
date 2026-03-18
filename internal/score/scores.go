// Package score implements all calculation logic ported from the R modules.
package score

import (
	"math"

	"sctool-go/internal/data"
	"sctool-go/internal/model"
)

// CalculateScores computes all score fields for each record.
// This is a port of calculate_scores.R.
//
// Steps:
//  1. Apply reversal to each question answer → QScore
//  2. Aggregate QScore into minor and major scales → ScaleScores
//  3. Compute high-stress (IsHS) using areas A/B/C (q1-q55)
//  4. Compute demand/control/boss_support/fellow_support sums (inverted)
//  5. Add AgeKubun
func CalculateScores(records []model.RawRecord, questions []data.QuestionInfo) []model.ProcessedRecord {
	// Build lookup: qnum (1-based) → QuestionInfo
	qinfo := make(map[int]data.QuestionInfo, len(questions))
	for _, q := range questions {
		qinfo[q.QNum] = q
	}

	// Pre-build minor→major mapping (only for the 5 main major scales).
	majorScales := map[string]bool{
		"w_total": true, "s_total": true, "j_total": true,
		"b_total": true, "p_total": true,
	}
	minorToMajor := make(map[string]string)
	for _, q := range questions {
		if q.SyakudoMajorEng != "" && majorScales[q.SyakudoMajorEng] {
			minorToMajor[q.SyakudoMinorEng] = q.SyakudoMajorEng
		}
	}

	result := make([]model.ProcessedRecord, 0, len(records))

	for tempID, raw := range records {
		rec := model.ProcessedRecord{
			TempID:      tempID + 1,
			EmpID:       raw.EmpID,
			Age:         raw.Age,
			Gender:      raw.Gender,
			Dept1:       raw.Dept1,
			Dept2:       raw.Dept2,
			Q:           raw.Q,
			QValid:      raw.QValid,
			ScaleScores: make(map[string]float64),
			AgeKubun:    ageKubun(raw.Age),
		}

		// --- Step 1: question scores with reversal ---
		for i := 0; i < 80; i++ {
			qnum := i + 1
			qi, ok := qinfo[qnum]
			if !ok {
				rec.QScore[i] = math.NaN()
				continue
			}
			if !raw.QValid[i] {
				rec.QScore[i] = math.NaN()
				continue
			}
			v := float64(raw.Q[i])
			if qi.IsReverse {
				rec.QScore[i] = 5 - v
			} else {
				rec.QScore[i] = v
			}
		}

		// --- Step 2: minor scale scores (mean of question scores in the scale) ---
		minorSums := make(map[string]float64)
		minorCounts := make(map[string]int)
		for i := 0; i < 80; i++ {
			qi, ok := qinfo[i+1]
			if !ok || qi.SyakudoMinorEng == "" {
				continue
			}
			score := rec.QScore[i]
			if math.IsNaN(score) {
				// Any NA within a scale makes the whole scale NA (na.rm=FALSE in R)
				minorSums[qi.SyakudoMinorEng] = math.NaN()
				minorCounts[qi.SyakudoMinorEng] = -1 // sentinel for "has NA"
				continue
			}
			if minorCounts[qi.SyakudoMinorEng] == -1 {
				continue // already NA
			}
			minorSums[qi.SyakudoMinorEng] += score
			minorCounts[qi.SyakudoMinorEng]++
		}
		for name, sum := range minorSums {
			cnt := minorCounts[name]
			if cnt <= 0 {
				rec.ScaleScores[name] = math.NaN()
			} else {
				rec.ScaleScores[name] = sum / float64(cnt)
			}
		}

		// --- Step 3: major scale scores (mean of minor scale scores) ---
		majorSums := make(map[string]float64)
		majorCounts := make(map[string]int)
		majorNA := make(map[string]bool)
		for minor, major := range minorToMajor {
			if major == "" {
				continue
			}
			s, ok := rec.ScaleScores[minor]
			if !ok || math.IsNaN(s) {
				majorNA[major] = true
				continue
			}
			if majorNA[major] {
				continue
			}
			majorSums[major] += s
			majorCounts[major]++
		}
		for major := range majorScales {
			if majorNA[major] {
				rec.ScaleScores[major] = math.NaN()
			} else if cnt := majorCounts[major]; cnt > 0 {
				rec.ScaleScores[major] = majorSums[major] / float64(cnt)
			}
		}

		// --- Step 4: high-stress calculation ---
		// R uses 5-score (inverted), then sums by area, compares totals.
		// Area A: q1-17, Area B: q18-46, Area C: q47-55
		areaA, areaANA := sumHSScore(rec.QScore, qinfo, 1, 17)
		areaB, areaBNA := sumHSScore(rec.QScore, qinfo, 18, 46)
		areaC, areaCNA := sumHSScore(rec.QScore, qinfo, 47, 55)

		rec.AreaA = areaA
		rec.AreaB = areaB
		rec.AreaC = areaC

		if areaANA || areaBNA || areaCNA {
			rec.IsHS = nil // NA
		} else {
			var hs bool
			if areaB >= 77 {
				hs = true
			} else if (areaA+areaC) >= 76 && areaB >= 63 {
				hs = true
			}
			rec.IsHS = &hs
		}

		// --- Step 5: demand/control/boss_support/fellow_support (inverted raw values) ---
		// R: mutate(across(matches("q"), ~5-.)) then sum specific questions
		// demand = q1+q2+q3, control = q8+q9+q10
		// boss_support = q47+q50+q53, fellow_support = q48+q51+q54
		rec.Demand = invertedSum(raw, []int{1, 2, 3})
		rec.Control = invertedSum(raw, []int{8, 9, 10})
		rec.BossSupport = invertedSum(raw, []int{47, 50, 53})
		rec.FellowSupport = invertedSum(raw, []int{48, 51, 54})

		result = append(result, rec)
	}

	return result
}

// sumHSScore returns the sum of (5 - qscore) for questions in [from, to] (1-based).
// Returns (sum, hasNA). The QScore values here are already-reversed regular scores;
// the HS calculation applies another inversion: hsscore = 5 - score.
func sumHSScore(qscores [80]float64, qinfo map[int]data.QuestionInfo, from, to int) (float64, bool) {
	sum := 0.0
	for qnum := from; qnum <= to; qnum++ {
		score := qscores[qnum-1]
		if math.IsNaN(score) {
			return math.NaN(), true
		}
		sum += 5 - score
	}
	return sum, false
}

// invertedSum returns the sum of (5 - raw_value) for the given question numbers (1-based).
// Returns NaN if any value is missing.
func invertedSum(raw model.RawRecord, qnums []int) float64 {
	sum := 0.0
	for _, qnum := range qnums {
		if !raw.QValid[qnum-1] {
			return math.NaN()
		}
		sum += 5 - float64(raw.Q[qnum-1])
	}
	return sum
}

// ageKubun returns the age group label for a given age.
func ageKubun(age int) string {
	switch {
	case age >= 0 && age <= 19:
		return "10代"
	case age >= 20 && age <= 29:
		return "20代"
	case age >= 30 && age <= 39:
		return "30代"
	case age >= 40 && age <= 49:
		return "40代"
	case age >= 50 && age <= 59:
		return "50代"
	case age >= 60:
		return "60代以上"
	default:
		return ""
	}
}
