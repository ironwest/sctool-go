// Package model defines the core data structures shared across the application.
package model

import "math"

// NA sentinel: use math.NaN() for missing float64 values.
// For integer/string fields, zero value or empty string indicates missing.

// RawRecord represents one row of a CSV after column mapping is applied.
// Q values use 0 to indicate a missing (NA) answer.
type RawRecord struct {
	EmpID  string
	Age    int
	Gender string
	Dept1  string
	Dept2  string
	Q      [80]int  // Q[0] = q1 … Q[79] = q80; 0 = missing
	QValid [80]bool // true when the corresponding Q value was present
}

// ProcessedRecord represents one employee after all score calculations.
// Float64 fields use math.NaN() for missing values.
type ProcessedRecord struct {
	TempID   int
	EmpID    string
	Age      int
	AgeKubun string // "10代","20代",…,"60代以上"
	Gender   string
	Dept1    string
	Dept2    string

	// Raw question answers (same as RawRecord.Q)
	Q      [80]int
	QValid [80]bool

	// Individual question scores after reversal (q1_score … q80_score)
	QScore [80]float64

	// Minor and major scale scores (e.g. "w_vol", "w_total")
	// Key: English scale name
	ScaleScores map[string]float64

	// High-stress calculation intermediate values
	AreaA float64 // sum of hsscore for questions 1-17
	AreaB float64 // sum of hsscore for questions 18-46
	AreaC float64 // sum of hsscore for questions 47-55
	IsHS  *bool   // nil = NA (incomplete answers)

	// Demand/Control/Support raw sums (inverted scoring, used for risk calc)
	Demand        float64
	Control       float64
	BossSupport   float64
	FellowSupport float64
}

// IsNaN is a convenience wrapper.
func IsNaN(f float64) bool { return math.IsNaN(f) }

// GroupKey identifies a grouping cell in analysis outputs.
type GroupKey struct {
	Dept1    string
	Dept2    string
	Gender   string
	AgeKubun string
}

// HensatiRow holds one group's hensati (偏差値) for one scale.
type HensatiRow struct {
	GroupVarValue string  // the value of the grouping variable
	HensatiGrp    string  // benchmark sheet used
	ScaleEng      string  // English scale name
	ScaleJapanese string  // Japanese scale name
	Value         float64 // group mean score
	MeanVal       float64 // benchmark mean
	SdVal         float64 // benchmark SD
	Hensati       float64 // 50 + 10*(Value-MeanVal)/SdVal
}

// RiskResult holds one group's comprehensive health risk scores.
type RiskResult struct {
	GroupValues   map[string]string
	RiskALong     float64
	RiskBLong     float64
	TotalRiskLong float64
	RiskACross    float64
	RiskBCross    float64
	TotalRiskCross float64
	RiskAOld      float64
	RiskBOld      float64
	TotalRiskOld  float64
}

// GroupSummary holds aggregated statistics for one group in the analysis table.
type GroupSummary struct {
	GroupValues      map[string]string
	N                int
	IncompleteN      int
	HighStressN      int
	HighStressRatio  float64
	ScaleScores      map[string]float64 // group mean scores
	HensatiScores    map[string]float64 // hensati values
	TotalRiskLong    float64
	TotalRiskCross   float64
	TotalRiskOld     float64
}
