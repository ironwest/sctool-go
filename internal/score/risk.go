package score

import (
	"math"

	"sctool-go/internal/data"
	"sctool-go/internal/model"
)

// riskIndustryCoefs holds pre-parsed coefficients for one industry.
type riskIndustryCoefs struct {
	// long coefficients
	longDemand        float64
	longControl       float64
	longBossSupport   float64
	longFellowSupport float64
	// cross coefficients
	crossDemand        float64
	crossControl       float64
	crossBossSupport   float64
	crossFellowSupport float64
	// averages (used for centering)
	avgDemand        float64
	avgControl       float64
	avgBossSupport   float64
	avgFellowSupport float64
}

// buildRiskCoefs extracts coefficients for a specific industry from the loaded CSV.
func buildRiskCoefs(coefs []data.RiskCoef, gyousyu string) riskIndustryCoefs {
	c := riskIndustryCoefs{}
	for _, rc := range coefs {
		if rc.Gyousyu != gyousyu {
			continue
		}
		switch rc.Type {
		case "long":
			switch rc.CoefName {
			case "demand":
				c.longDemand = rc.Coef
				if !math.IsNaN(rc.Avg) {
					c.avgDemand = rc.Avg
				}
			case "control":
				c.longControl = rc.Coef
				if !math.IsNaN(rc.Avg) {
					c.avgControl = rc.Avg
				}
			case "boss_support":
				c.longBossSupport = rc.Coef
				if !math.IsNaN(rc.Avg) {
					c.avgBossSupport = rc.Avg
				}
			case "fellow_support":
				c.longFellowSupport = rc.Coef
				if !math.IsNaN(rc.Avg) {
					c.avgFellowSupport = rc.Avg
				}
			}
		case "cross":
			switch rc.CoefName {
			case "demand":
				c.crossDemand = rc.Coef
			case "control":
				c.crossControl = rc.Coef
			case "boss_support":
				c.crossBossSupport = rc.Coef
			case "fellow_support":
				c.crossFellowSupport = rc.Coef
			}
		}
	}
	return c
}

// GroupStats holds aggregated demand/control/support means for one group.
type GroupStats struct {
	GroupValues   map[string]string
	Demand        float64
	Control       float64
	BossSupport   float64
	FellowSupport float64
}

// AggregateGroupStats computes per-group means of demand/control/support.
func AggregateGroupStats(records []model.ProcessedRecord, groupVars []string) []GroupStats {
	type acc struct {
		dSum, cSum, bSum, fSum     float64
		dCnt, cCnt, bCnt, fCnt    int
	}

	type key = string // JSON-encoded group key, simpler to use a string concat
	byGroup := make(map[string]*acc)
	groupValsMap := make(map[string]map[string]string)
	order := []string{}

	makeKey := func(r model.ProcessedRecord) (string, map[string]string) {
		k := ""
		m := make(map[string]string, len(groupVars))
		for _, gv := range groupVars {
			var val string
			switch gv {
			case "dept1":
				val = r.Dept1
			case "dept2":
				val = r.Dept2
			case "gender":
				val = r.Gender
			case "age_kubun":
				val = r.AgeKubun
			}
			m[gv] = val
			k += gv + "=" + val + ";"
		}
		return k, m
	}

	for _, rec := range records {
		k, m := makeKey(rec)
		if _, ok := byGroup[k]; !ok {
			byGroup[k] = &acc{}
			groupValsMap[k] = m
			order = append(order, k)
		}
		a := byGroup[k]
		if !math.IsNaN(rec.Demand) {
			a.dSum += rec.Demand; a.dCnt++
		}
		if !math.IsNaN(rec.Control) {
			a.cSum += rec.Control; a.cCnt++
		}
		if !math.IsNaN(rec.BossSupport) {
			a.bSum += rec.BossSupport; a.bCnt++
		}
		if !math.IsNaN(rec.FellowSupport) {
			a.fSum += rec.FellowSupport; a.fCnt++
		}
	}

	result := make([]GroupStats, 0, len(byGroup))
	for _, k := range order {
		a := byGroup[k]
		gs := GroupStats{GroupValues: groupValsMap[k]}
		gs.Demand = nanMean(a.dSum, a.dCnt)
		gs.Control = nanMean(a.cSum, a.cCnt)
		gs.BossSupport = nanMean(a.bSum, a.bCnt)
		gs.FellowSupport = nanMean(a.fSum, a.fCnt)
		result = append(result, gs)
	}
	return result
}

func nanMean(sum float64, cnt int) float64 {
	if cnt == 0 {
		return math.NaN()
	}
	return sum / float64(cnt)
}

// CalculateRiskScore computes comprehensive health risk scores per group.
// Port of calculate_sougoukrisk.R.
//
// groupStats should come from AggregateGroupStats.
// coefs should be loaded via data.LoadRiskCoefs().
// gyousyu is the industry name (e.g. "全産業").
func CalculateRiskScore(groupStats []GroupStats, coefs []data.RiskCoef, gyousyu string) []model.RiskResult {
	c := buildRiskCoefs(coefs, gyousyu)

	// Old (fixed) coefficients from the R source.
	const (
		oldAvgDemand  = 8.7
		oldCoefDemand = 0.076
		oldAvgControl = 8.0
		oldCoefControl = -0.089
		oldAvgBoss    = 7.6
		oldCoefBoss   = -0.097
		oldAvgFellow  = 8.1
		oldCoefFellow = -0.097
	)

	result := make([]model.RiskResult, 0, len(groupStats))

	for _, gs := range groupStats {
		d := gs.Demand
		ctrl := gs.Control
		boss := gs.BossSupport
		fellow := gs.FellowSupport

		riskALong := floorMin350(
			math.Exp(((d-c.avgDemand)*c.longDemand)+((ctrl-c.avgControl)*c.longControl)) * 100,
		)
		riskBLong := floorMin350(
			math.Exp(((boss-c.avgBossSupport)*c.longBossSupport)+((fellow-c.avgFellowSupport)*c.longFellowSupport)) * 100,
		)
		totalLong := math.Floor(riskALong * riskBLong / 100)

		riskACross := floorMin350(
			math.Exp(((d-c.avgDemand)*c.crossDemand)+((ctrl-c.avgControl)*c.crossControl)) * 100,
		)
		riskBCross := floorMin350(
			math.Exp(((boss-c.avgBossSupport)*c.crossBossSupport)+((fellow-c.avgFellowSupport)*c.crossFellowSupport)) * 100,
		)
		totalCross := math.Floor(riskACross * riskBCross / 100)

		riskAOld := floorMin350(
			math.Exp(((d-oldAvgDemand)*oldCoefDemand)+((ctrl-oldAvgControl)*oldCoefControl)) * 100,
		)
		riskBOld := floorMin350(
			math.Exp(((boss-oldAvgBoss)*oldCoefBoss)+((fellow-oldAvgFellow)*oldCoefFellow)) * 100,
		)
		totalOld := math.Floor(riskAOld * riskBOld / 100)

		result = append(result, model.RiskResult{
			GroupValues:    gs.GroupValues,
			RiskALong:      riskALong,
			RiskBLong:      riskBLong,
			TotalRiskLong:  totalLong,
			RiskACross:     riskACross,
			RiskBCross:     riskBCross,
			TotalRiskCross: totalCross,
			RiskAOld:       riskAOld,
			RiskBOld:       riskBOld,
			TotalRiskOld:   totalOld,
		})
	}

	return result
}

// floorMin350 applies floor(pmin(x, 350)) as in the R code.
func floorMin350(x float64) float64 {
	if x > 350 {
		x = 350
	}
	return math.Floor(x)
}
