package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"sctool-go/internal/data"
	"sctool-go/internal/model"
	"sctool-go/internal/parser"
	"sctool-go/internal/score"
)

// App struct holds all application state shared between Wails bindings.
type App struct {
	ctx context.Context

	mu sync.Mutex

	// Loaded static data (initialized once at startup)
	questions []data.QuestionInfo
	labels    []data.LabelRow
	benchmarks []data.BenchmarkRow
	riskCoefs  []data.RiskCoef

	// Wizard session state (two independent sessions: 今年度 / 昨年度)
	sessions map[string]*WizardSession
}

// WizardSession holds the in-progress state for one data year.
type WizardSession struct {
	// Raw CSV data
	Headers []string
	Rows    [][]string

	// Processed records after CalculateScores
	Processed []model.ProcessedRecord

	// Source file name (for default save name)
	SourceFileName string
}

// --- Wire types (JSON-serializable structs sent to/from frontend) ---

// CSVLoadResult is returned after loading a CSV file.
type CSVLoadResult struct {
	OK          bool       `json:"ok"`
	Error       string     `json:"error,omitempty"`
	FileName    string     `json:"fileName"`
	RowCount    int        `json:"rowCount"`
	ColCount    int        `json:"colCount"`
	Headers     []string   `json:"headers"`
	Preview     [][]string `json:"preview"` // first 5 rows
	UniqueVals  []string   `json:"uniqueVals"` // all unique values across q columns
}

// BasicAttributesMap holds the column mapping for basic demographic fields.
type BasicAttributesMap struct {
	EmpID  string `json:"empid"`
	Age    string `json:"age"`
	Gender string `json:"gender"`
	Dept1  string `json:"dept1"`
	Dept2  string `json:"dept2"`
}

// ColumnMapConfig matches the JSON schema from wizard_module.R.
type ColumnMapConfig struct {
	BasicAttributes BasicAttributesMap `json:"basic_attributes"`
	NBJSQQuestions  map[string]string  `json:"nbjsq_questions"` // "q1".."q80" → column name
}

// GenderValMap holds the raw CSV values that correspond to male/female.
type GenderValMap struct {
	Male   string `json:"male"`
	Female string `json:"female"`
}

// NBJSQBulkValMap holds bulk (section-level) value mappings.
type NBJSQBulkValMap struct {
	GroupAEFGH []string `json:"group_aefgh"`
	GroupB     []string `json:"group_b"`
	GroupC     []string `json:"group_c"`
	GroupD     []string `json:"group_d"`
}

// ValueMapConfig matches the JSON schema from wizard_module.R.
type ValueMapConfig struct {
	Gender          GenderValMap        `json:"gender"`
	NBJSQBulk       NBJSQBulkValMap     `json:"nbjsq_bulk"`
	NBJSQIndividual map[string][]string `json:"nbjsq_individual"` // "q1".."q80" → [val1,val2,val3,val4]
}

// AutoDetectResult holds suggested column mappings based on leading digits.
type AutoDetectResult struct {
	NBJSQQuestions map[string]string `json:"nbjsq_questions"` // "q1".."q80" → best match column name
}

// ApplyResult is returned after applying mappings and running score calculation.
type ApplyResult struct {
	OK           bool   `json:"ok"`
	Error        string `json:"error,omitempty"`
	RecordCount  int    `json:"recordCount"`
	HighStressN  int    `json:"highStressN"`
	IncompleteN  int    `json:"incompleteN"`
}

// AnalysisTableResult is the response for GetAnalysisTable.
type AnalysisTableResult struct {
	OK       bool                    `json:"ok"`
	Error    string                  `json:"error,omitempty"`
	GroupVar string                  `json:"groupVar"`
	Rows     []score.AnalysisTableRow `json:"rows"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		sessions: make(map[string]*WizardSession),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load static data
	var err error
	a.questions, err = data.LoadQuestions()
	if err != nil {
		fmt.Println("ERROR loading questions:", err)
	}
	a.labels, err = data.LoadLabels()
	if err != nil {
		fmt.Println("ERROR loading labels:", err)
	}
	a.benchmarks, err = data.LoadBenchmarks()
	if err != nil {
		fmt.Println("ERROR loading benchmarks:", err)
	}
	a.riskCoefs, err = data.LoadRiskCoefs()
	if err != nil {
		fmt.Println("ERROR loading risk coefs:", err)
	}
}

func (a *App) getSession(yearLabel string) *WizardSession {
	if s, ok := a.sessions[yearLabel]; ok {
		return s
	}
	s := &WizardSession{}
	a.sessions[yearLabel] = s
	return s
}

// --- Wails bindings ---

// OpenCSVFileDialog opens a native file picker and returns the selected path.
func (a *App) OpenCSVFileDialog() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "CSVファイルを選択",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

// OpenJSONFileDialog opens a native file picker for JSON config files.
func (a *App) OpenJSONFileDialog() string {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "設定ファイルを選択",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

// SaveFileDialog opens a native save dialog and returns the chosen path.
func (a *App) SaveFileDialog(defaultName string, ext string) string {
	filter := "CSV Files (*.csv)"
	pattern := "*.csv"
	if ext == "json" {
		filter = "JSON Files (*.json)"
		pattern = "*.json"
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存先を選択",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: filter, Pattern: pattern},
		},
	})
	if err != nil {
		return ""
	}
	return path
}

// LoadCSVFile reads the CSV file at the given path, stores it in the session,
// and returns a preview plus all unique values found in question columns.
func (a *App) LoadCSVFile(yearLabel string, path string) CSVLoadResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	headers, rows, err := parseCSVFileInternal(path)
	if err != nil {
		return CSVLoadResult{OK: false, Error: err.Error()}
	}

	session := a.getSession(yearLabel)
	session.Headers = headers
	session.Rows = rows
	session.SourceFileName = filepath.Base(path)
	session.Processed = nil

	// Build preview (first 5 rows)
	preview := rows
	if len(preview) > 5 {
		preview = rows[:5]
	}

	// Collect all unique values across all columns (for value mapping UI)
	allVals := make(map[string]bool)
	for _, row := range rows {
		for _, cell := range row {
			v := strings.TrimSpace(cell)
			if v != "" {
				allVals[v] = true
			}
		}
	}
	uniqueVals := make([]string, 0, len(allVals))
	for v := range allVals {
		uniqueVals = append(uniqueVals, v)
	}
	sort.Strings(uniqueVals)

	return CSVLoadResult{
		OK:         true,
		FileName:   session.SourceFileName,
		RowCount:   len(rows),
		ColCount:   len(headers),
		Headers:    headers,
		Preview:    preview,
		UniqueVals: uniqueVals,
	}
}

// AutoDetectColumns suggests column mappings by matching leading digits in column names.
// Port of the R logic: as.numeric(str_extract(csv_headers, "\\d+"))
func (a *App) AutoDetectColumns(yearLabel string) AutoDetectResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.getSession(yearLabel)
	result := AutoDetectResult{
		NBJSQQuestions: make(map[string]string, 80),
	}

	re := regexp.MustCompile(`\d+`)
	// For each header, extract the first number
	headerNums := make(map[int]string) // number → first column name with that number
	for _, h := range session.Headers {
		m := re.FindString(h)
		if m == "" {
			continue
		}
		num, err := strconv.Atoi(m)
		if err != nil {
			continue
		}
		if _, exists := headerNums[num]; !exists {
			headerNums[num] = h
		}
	}

	for i := 1; i <= 80; i++ {
		key := fmt.Sprintf("q%d", i)
		if col, ok := headerNums[i]; ok {
			result.NBJSQQuestions[key] = col
		} else {
			result.NBJSQQuestions[key] = ""
		}
	}
	return result
}

// LoadColumnMapConfig reads a JSON column mapping config file.
func (a *App) LoadColumnMapConfig(path string) (ColumnMapConfig, error) {
	var cfg ColumnMapConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(b, &cfg)
	return cfg, err
}

// SaveColumnMapConfig saves a ColumnMapConfig to a JSON file.
func (a *App) SaveColumnMapConfig(cfg ColumnMapConfig, path string) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// LoadValueMapConfig reads a JSON value mapping config file.
func (a *App) LoadValueMapConfig(path string) (ValueMapConfig, error) {
	var cfg ValueMapConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(b, &cfg)
	return cfg, err
}

// SaveValueMapConfig saves a ValueMapConfig to a JSON file.
func (a *App) SaveValueMapConfig(cfg ValueMapConfig, path string) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// ApplyMappingAndCalculate applies column+value mappings, runs CalculateScores,
// and stores the result in the session.
func (a *App) ApplyMappingAndCalculate(yearLabel string, colCfg ColumnMapConfig, valCfg ValueMapConfig) ApplyResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.getSession(yearLabel)
	if len(session.Headers) == 0 {
		return ApplyResult{OK: false, Error: "CSVファイルが読み込まれていません"}
	}

	// Build index map: header name → column index
	colNameToIdx := make(map[string]int, len(session.Headers))
	for i, h := range session.Headers {
		colNameToIdx[h] = i
	}

	// Build parser.ColumnMap from ColumnMapConfig
	colMap := parser.NewColumnMap()
	colMap.EmpID = colNameToIdx[colCfg.BasicAttributes.EmpID] - 1
	if _, ok := colNameToIdx[colCfg.BasicAttributes.EmpID]; !ok {
		colMap.EmpID = -1
	} else {
		colMap.EmpID = colNameToIdx[colCfg.BasicAttributes.EmpID]
	}
	colMap.Age = getColIdx(colNameToIdx, colCfg.BasicAttributes.Age)
	colMap.Gender = getColIdx(colNameToIdx, colCfg.BasicAttributes.Gender)
	colMap.Dept1 = getColIdx(colNameToIdx, colCfg.BasicAttributes.Dept1)
	colMap.Dept2 = getColIdx(colNameToIdx, colCfg.BasicAttributes.Dept2)

	for i := 1; i <= 80; i++ {
		key := fmt.Sprintf("q%d", i)
		colName := colCfg.NBJSQQuestions[key]
		colMap.Q[i-1] = getColIdx(colNameToIdx, colName)
	}

	// Build parser.ValueMap from ValueMapConfig
	// For each question column, map raw values → 1,2,3,4
	valMap := make(parser.ValueMap)

	buildQValMap := func(colName string, vals []string) {
		if colName == "" || len(vals) < 4 {
			return
		}
		m := make(map[string]int, 4)
		for i, v := range vals[:4] {
			if v != "" {
				m[v] = i + 1
			}
		}
		valMap[colName] = m
	}

	for i := 1; i <= 80; i++ {
		key := fmt.Sprintf("q%d", i)
		colName := colCfg.NBJSQQuestions[key]
		if colName == "" {
			continue
		}
		// Use individual mapping if available and non-empty
		if individual, ok := valCfg.NBJSQIndividual[key]; ok && len(individual) == 4 {
			buildQValMap(colName, individual)
		}
	}

	// Gender value map: map raw gender values → "男性"/"女性" by substitution
	// We store this separately and apply it during RawRecord construction
	genderMap := map[string]string{
		valCfg.Gender.Male:   "男性",
		valCfg.Gender.Female: "女性",
	}

	// Apply mapping to get RawRecords
	raws, err := parser.ApplyMapping(session.Headers, session.Rows, colMap, valMap)
	if err != nil {
		return ApplyResult{OK: false, Error: err.Error()}
	}

	// Apply gender mapping
	genderColIdx := getColIdx(colNameToIdx, colCfg.BasicAttributes.Gender)
	for i, raw := range raws {
		if genderColIdx >= 0 && genderColIdx < len(session.Rows[i]) {
			rawGender := strings.TrimSpace(session.Rows[i][genderColIdx])
			if mapped, ok := genderMap[rawGender]; ok {
				raws[i].Gender = mapped
			} else {
				raws[i].Gender = raw.Gender
			}
		}
	}

	// Run score calculation
	processed := score.CalculateScores(raws, a.questions)
	session.Processed = processed

	// Count stats
	highStressN, incompleteN := 0, 0
	for _, rec := range processed {
		if rec.IsHS == nil {
			incompleteN++
		} else if *rec.IsHS {
			highStressN++
		}
	}

	return ApplyResult{
		OK:          true,
		RecordCount: len(processed),
		HighStressN: highStressN,
		IncompleteN: incompleteN,
	}
}

// SaveProcessedCSV exports the processed records to a CSV file.
func (a *App) SaveProcessedCSV(yearLabel string, path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.getSession(yearLabel)
	if len(session.Processed) == 0 {
		return fmt.Errorf("処理済みデータがありません")
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header (matching R output schema)
	header := buildProcessedCSVHeader()
	w.Write(header)

	for _, rec := range session.Processed {
		w.Write(processedRecordToCSVRow(rec))
	}
	return nil
}

// LoadProcessedCSV loads an already-processed CSV into the session (Step 1' path).
func (a *App) LoadProcessedCSV(yearLabel string, path string) ApplyResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	records, err := parser.ParseProcessedCSV(path)
	if err != nil {
		return ApplyResult{OK: false, Error: err.Error()}
	}

	session := a.getSession(yearLabel)
	session.Processed = records
	session.SourceFileName = filepath.Base(path)

	highStressN, incompleteN := 0, 0
	for _, rec := range records {
		if rec.IsHS == nil {
			incompleteN++
		} else if *rec.IsHS {
			highStressN++
		}
	}

	return ApplyResult{
		OK:          true,
		RecordCount: len(records),
		HighStressN: highStressN,
		IncompleteN: incompleteN,
	}
}

// DefaultSaveFileName returns the default filename for saving a processed CSV.
func (a *App) DefaultSaveFileName(yearLabel string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	session := a.getSession(yearLabel)
	name := session.SourceFileName
	if name == "" {
		name = fmt.Sprintf("processed_%s_%s.csv", yearLabel, time.Now().Format("20060102"))
	} else if !strings.HasPrefix(name, "processed_") {
		name = "processed_" + name
	}
	return name
}

// DefaultConfigSaveFileName returns a timestamped default filename for config JSON.
func (a *App) DefaultConfigSaveFileName(kind string, yearLabel string) string {
	return fmt.Sprintf("%s_config_%s_%s.json", kind, yearLabel, time.Now().Format("20060102"))
}

// GetAnalysisTable computes the 偏差値表 for the given year and grouping options.
//
// groupVar: "dept1" | "dept2" | "dept1_dept2" | "age_kubun" | "gender"
// longOrCross: "long" | "cross"
// gyousyu: industry name, e.g. "全産業"
func (a *App) GetAnalysisTable(yearLabel string, groupVar string, longOrCross string, gyousyu string) AnalysisTableResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.getSession(yearLabel)
	if len(session.Processed) == 0 {
		return AnalysisTableResult{OK: false, Error: "処理済みデータがありません。先にデータを読み込んでください。"}
	}

	rows := score.GetAnalysisHyou(
		session.Processed,
		groupVar,
		longOrCross,
		gyousyu,
		a.questions,
		a.benchmarks,
		a.labels,
		a.riskCoefs,
	)

	return AnalysisTableResult{
		OK:       true,
		GroupVar: groupVar,
		Rows:     rows,
	}
}

// --- helpers ---

func getColIdx(m map[string]int, name string) int {
	if name == "" {
		return -1
	}
	if idx, ok := m[name]; ok {
		return idx
	}
	return -1
}

func parseCSVFileInternal(path string) ([]string, [][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("ファイル読み込みエラー: %w", err)
	}
	// Remove UTF-8 BOM
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	if !utf8.Valid(raw) {
		decoded, _, err := transform.Bytes(japanese.ShiftJIS.NewDecoder(), raw)
		if err != nil {
			return nil, nil, fmt.Errorf("文字コード変換エラー: %w", err)
		}
		raw = decoded
	}
	r := csv.NewReader(strings.NewReader(string(raw)))
	r.LazyQuotes = true
	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("CSV解析エラー: %w", err)
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("CSVファイルが空です")
	}
	return all[0], all[1:], nil
}

// buildProcessedCSVHeader returns the canonical column order for the processed CSV output.
// This matches the R output schema exactly for compatibility.
func buildProcessedCSVHeader() []string {
	cols := []string{"tempid", "empid", "age", "age_kubun", "gender", "dept1", "dept2"}
	for i := 1; i <= 80; i++ {
		cols = append(cols, fmt.Sprintf("q%d", i))
	}
	cols = append(cols, "A", "B", "C", "is_hs")
	for i := 1; i <= 80; i++ {
		cols = append(cols, fmt.Sprintf("q%d_score", i))
	}
	for _, s := range []string{
		"b_antreward", "b_bossfair", "b_bossleader", "b_bosssupp",
		"b_collsupp", "b_ecoreward", "b_homeru", "b_sippai", "b_sonreward",
		"j_carrier", "j_change", "j_dei", "j_jinji", "j_keiei", "j_kojin", "j_wsbpos",
		"na_cc", "na_famsupp", "na_kateimanzoku", "na_workmanzoku",
		"o_harass", "o_sc", "o_we",
		"p_hirou", "p_huan", "p_iraira", "p_kakki", "p_utu",
		"s_control", "s_ginou", "s_growth", "s_igi", "s_tek", "s_yakuwarimei",
		"w_env", "w_hutan", "w_jyoutyo", "w_qua", "w_tai", "w_vol", "w_wsbneg", "w_yakuwarikat",
		"b_total", "j_total", "p_total", "s_total", "w_total",
		"demand", "control", "boss_support", "fellow_support",
	} {
		cols = append(cols, s)
	}
	return cols
}

func processedRecordToCSVRow(rec model.ProcessedRecord) []string {
	row := []string{
		strconv.Itoa(rec.TempID),
		rec.EmpID,
		strconv.Itoa(rec.Age),
		rec.AgeKubun,
		rec.Gender,
		rec.Dept1,
		rec.Dept2,
	}
	for i := 0; i < 80; i++ {
		if rec.QValid[i] {
			row = append(row, strconv.Itoa(rec.Q[i]))
		} else {
			row = append(row, "")
		}
	}
	row = append(row, fmtFloat(rec.AreaA), fmtFloat(rec.AreaB), fmtFloat(rec.AreaC))
	if rec.IsHS == nil {
		row = append(row, "NA")
	} else if *rec.IsHS {
		row = append(row, "TRUE")
	} else {
		row = append(row, "FALSE")
	}
	for i := 0; i < 80; i++ {
		row = append(row, fmtFloat(rec.QScore[i]))
	}
	for _, s := range []string{
		"b_antreward", "b_bossfair", "b_bossleader", "b_bosssupp",
		"b_collsupp", "b_ecoreward", "b_homeru", "b_sippai", "b_sonreward",
		"j_carrier", "j_change", "j_dei", "j_jinji", "j_keiei", "j_kojin", "j_wsbpos",
		"na_cc", "na_famsupp", "na_kateimanzoku", "na_workmanzoku",
		"o_harass", "o_sc", "o_we",
		"p_hirou", "p_huan", "p_iraira", "p_kakki", "p_utu",
		"s_control", "s_ginou", "s_growth", "s_igi", "s_tek", "s_yakuwarimei",
		"w_env", "w_hutan", "w_jyoutyo", "w_qua", "w_tai", "w_vol", "w_wsbneg", "w_yakuwarikat",
		"b_total", "j_total", "p_total", "s_total", "w_total",
	} {
		v, ok := rec.ScaleScores[s]
		if !ok {
			row = append(row, "")
		} else {
			row = append(row, fmtFloat(v))
		}
	}
	row = append(row,
		fmtFloat(rec.Demand),
		fmtFloat(rec.Control),
		fmtFloat(rec.BossSupport),
		fmtFloat(rec.FellowSupport),
	)
	return row
}

func fmtFloat(f float64) string {
	if math.IsNaN(f) {
		return "NA"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
