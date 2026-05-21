package gap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type GapAnalyzer struct {
	projectRoot string
	coverage    *CoverageReport
	gitChanges  *GitChangeReport
	riskMatrix  RiskMatrix
}

type CoverageReport struct {
	Files map[string]FileCoverage `json:"files"`
}

type FileCoverage struct {
	Path               string   `json:"path"`
	CoveragePercent    float64  `json:"coverage_percent"`
	UncoveredLines     []int    `json:"uncovered_lines"`
	TotalLines         int      `json:"total_lines"`
	CoveredLines       int      `json:"covered_lines"`
	FunctionsTotal     int      `json:"functions_total"`
	FunctionsCovered   int      `json:"functions_covered"`
	HasTestFile        bool     `json:"has_test_file"`
	HighRiskPatterns   []string `json:"high_risk_patterns"`
}

type GitChangeReport struct {
	RecentChanges []FileChange `json:"recent_changes"`
	ModifiedNoTest []FileChange `json:"modified_no_test"`
}

type FileChange struct {
	Path       string `json:"path"`
	Status     string `json:"status"`
	LastChange string `json:"last_change"`
	HasTest    bool   `json:"has_test"`
	Complexity int    `json:"complexity"`
}

type RiskMatrix struct {
	Entries []RiskEntry `json:"entries"`
}

type RiskEntry struct {
	FilePath         string  `json:"file_path"`
	RiskLevel        string  `json:"risk_level"`
	RiskScore        float64 `json:"risk_score"`
	RiskFactors      []string `json:"risk_factors"`
	RecommendedTests []string `json:"recommended_tests"`
	Reason           string  `json:"reason"`
}

type GapAnalysisResult struct {
	Timestamp          time.Time        `json:"timestamp"`
	TotalFilesAnalyzed int              `json:"total_files_analyzed"`
	HighRiskGaps       []RiskEntry      `json:"high_risk_gaps"`
	MediumRiskGaps     []RiskEntry      `json:"medium_risk_gaps"`
	LowRiskGaps        []RiskEntry      `json:"low_risk_gaps"`
	CoverageGaps       []CoverageGap    `json:"coverage_gaps"`
	ChangeGaps         []ChangeGap      `json:"change_gaps"`
	Summary            AnalysisSummary   `json:"summary"`
}

type CoverageGap struct {
	FilePath         string   `json:"file_path"`
	UncoveredLines   []int    `json:"uncovered_lines"`
	UncoveredRanges  []LineRange `json:"uncovered_ranges"`
	FunctionName     string   `json:"function_name,omitempty"`
	IsCriticalPath  bool     `json:"is_critical_path"`
}

type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type ChangeGap struct {
	FilePath    string   `json:"file_path"`
	Changes     []Change `json:"changes"`
	RiskLevel   string   `json:"risk_level"`
	Reason      string   `json:"reason"`
}

type Change struct {
	Type     string `json:"type"` // added, modified, deleted
	Location string `json:"location"`
	Content  string `json:"content,omitempty"`
}

type AnalysisSummary struct {
	TotalGaps          int     `json:"total_gaps"`
	HighPriorityCount  int     `json:"high_priority_count"`
	MediumPriorityCount int    `json:"medium_priority_count"`
	LowPriorityCount   int     `json:"low_priority_count"`
	AvgCoverage        float64 `json:"avg_coverage"`
	CriticalModules    []string `json:"critical_modules"`
}

var (
	criticalPatterns = []string{
		`func\s+\(.*\)?\s+(Deploy|Build|Rollback|Heal)`,
		`func\s+\(.*\)?\s+(Auth|Login|Logout|Validate)`,
		`func\s+\(.*\)?\s+(Create|Update|Delete).*(Credential|Key|Secret)`,
		`func\s+\(.*\)?\s+Parse.*\(.*string`,
		`func\s+\(.*\)?\s+Validate.*\(.*\)`,
	}

	highRiskKeywords = []string{
		"goroutine", "channel", "mutex", "RWMutex", "sync.",
		"exec.Command", "os/exec",
		"crypto.", "decrypt", "encrypt",
		"sql.", "gorm.",
		"json.Unmarshal", "json.Marshal",
		"regexp.", "template.",
		"http.", "ServeHTTP", "HandleFunc",
	}
)

func NewGapAnalyzer(projectRoot string) *GapAnalyzer {
	return &GapAnalyzer{
		projectRoot: projectRoot,
		coverage:    &CoverageReport{Files: make(map[string]FileCoverage)},
		gitChanges:  &GitChangeReport{},
		riskMatrix:  RiskMatrix{Entries: make([]RiskEntry, 0)},
	}
}

func (g *GapAnalyzer) Analyze() (*GapAnalysisResult, error) {
	result := &GapAnalysisResult{
		Timestamp: time.Now(),
	}

	if err := g.collectCoverageData(); err != nil {
		return nil, fmt.Errorf("failed to collect coverage data: %w", err)
	}

	if err := g.analyzeGitChanges(); err != nil {
		return nil, fmt.Errorf("failed to analyze git changes: %w", err)
	}

	g.identifyCoverageGaps()
	g.identifyChangeGaps()
	g.assessRiskLevels()

	result.TotalFilesAnalyzed = len(g.coverage.Files)
	result.HighRiskGaps = g.filterByRisk("high")
	result.MediumRiskGaps = g.filterByRisk("medium")
	result.LowRiskGaps = g.filterByRisk("low")
	result.Summary = g.generateSummary()

	return result, nil
}

func (g *GapAnalyzer) collectCoverageData() error {
	profileFile := filepath.Join(g.projectRoot, "coverage.out")
	if _, err := os.Stat(profileFile); os.IsNotExist(err) {
		if err := g.runCoverage(); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(profileFile)
	if err != nil {
		return fmt.Errorf("failed to read coverage file: %w", err)
	}

	return g.parseCoverageReport(string(data))
}

func (g *GapAnalyzer) runCoverage() error {
	cmd := exec.Command("go", "test", "-coverprofile=coverage.out", "-count=1", "./...")
	cmd.Dir = g.projectRoot
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CI=true")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Coverage run warning: %v\nOutput: %s\n", err, string(output))
	}
	return nil
}

func (g *GapAnalyzer) parseCoverageReport(content string) error {
	lines := strings.Split(content, "\n")
	var currentFile string
	fileStmts := make(map[string]map[int]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		if strings.HasPrefix(line, "github.com/Yogdunana/deploypilot/") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			currentFile = g.cleanFilePath(parts[0])
			lineData := strings.TrimSpace(parts[1])
			g.parseCoverageEntry(currentFile, lineData, fileStmts)
		}
	}

	for filePath, stmts := range fileStmts {
		fc := g.coverage.Files[filePath]
		if fc.TotalLines > 0 {
			fc.CoveredLines = 0
			for _, covered := range stmts {
				if covered {
					fc.CoveredLines++
				}
			}
			fc.CoveragePercent = float64(fc.CoveredLines) / float64(fc.TotalLines) * 100
			g.coverage.Files[filePath] = fc
		}
	}

	return nil
}

func (g *GapAnalyzer) parseCoverageEntry(filePath, lineData string, fileStmts map[string]map[int]bool) {
	if _, exists := g.coverage.Files[filePath]; !exists {
		g.coverage.Files[filePath] = FileCoverage{
			Path:             filePath,
			CoveragePercent:  0,
			UncoveredLines:   make([]int, 0),
			TotalLines:       0,
			CoveredLines:     0,
			FunctionsTotal:   0,
			FunctionsCovered: 0,
			HighRiskPatterns: make([]string, 0),
		}
	}

	parts := strings.Fields(lineData)
	if len(parts) < 2 {
		return
	}

	rangePart := parts[0]
	hits := 0
	fmt.Sscanf(parts[len(parts)-1], "%d", &hits)

	startParts := strings.Split(rangePart, ",")
	if len(startParts) < 2 {
		return
	}

	startLoc := startParts[0]
	endLoc := startParts[1]

	startLine := 0
	startCol := 0
	fmt.Sscanf(startLoc, "%d.%d", &startLine, &startCol)

	endLine := 0
	endCol := 0
	fmt.Sscanf(endLoc, "%d.%d", &endLine, &endCol)

	if startLine == 0 {
		return
	}

	fc := g.coverage.Files[filePath]

	if fileStmts[filePath] == nil {
		fileStmts[filePath] = make(map[int]bool)
	}

	for line := startLine; line <= endLine; line++ {
		fc.TotalLines++
		fileStmts[filePath][line] = hits > 0
		if hits == 0 {
			fc.UncoveredLines = append(fc.UncoveredLines, line)
		}
	}

	g.detectHighRiskPatterns(filePath, &fc)
	g.checkForTestFile(filePath, &fc)

	g.coverage.Files[filePath] = fc
}

func (g *GapAnalyzer) cleanFilePath(path string) string {
	path = strings.TrimPrefix(path, "github.com/Yogdunana/deploypilot/")
	path = strings.TrimPrefix(path, "./")
	return filepath.Join(g.projectRoot, path)
}

func (g *GapAnalyzer) detectHighRiskPatterns(filePath string, fc *FileCoverage) {
	for _, pattern := range highRiskKeywords {
		if strings.Contains(filePath, pattern) || g.fileContains(filePath, pattern) {
			fc.HighRiskPatterns = append(fc.HighRiskPatterns, pattern)
		}
	}
}

func (g *GapAnalyzer) fileContains(filePath, pattern string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), pattern)
}

func (g *GapAnalyzer) checkForTestFile(filePath string, fc *FileCoverage) {
	testPath := filePath + "_test.go"
	if _, err := os.Stat(testPath); err == nil {
		fc.HasTestFile = true
	}

	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	baseNameWithoutExt := strings.TrimSuffix(baseName, ext)
	testPath2 := filepath.Join(filepath.Dir(filePath), baseNameWithoutExt+"_test.go")
	if _, err := os.Stat(testPath2); err == nil {
		fc.HasTestFile = true
	}
}

func (g *GapAnalyzer) analyzeGitChanges() error {
	g.gitChanges.RecentChanges = make([]FileChange, 0)
	g.gitChanges.ModifiedNoTest = make([]FileChange, 0)

	cmd := exec.Command("git", "diff", "--name-only", "HEAD~20", "HEAD")
	cmd.Dir = g.projectRoot
	output, err := cmd.Output()
	if err != nil {
		g.gitChanges.RecentChanges = []FileChange{}
		return nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, file := range files {
		if file == "" || strings.HasSuffix(file, "_test.go") {
			continue
		}
		if strings.HasPrefix(file, "vendor/") || strings.HasPrefix(file, "web/") {
			continue
		}

		change := FileChange{
			Path:       file,
			Status:     "modified",
			LastChange: time.Now().Format(time.RFC3339),
			HasTest:    g.hasTestFor(file),
			Complexity: g.estimateComplexity(file),
		}

		g.gitChanges.RecentChanges = append(g.gitChanges.RecentChanges, change)

		if !change.HasTest {
			g.gitChanges.ModifiedNoTest = append(g.gitChanges.ModifiedNoTest, change)
		}
	}

	return nil
}

func (g *GapAnalyzer) hasTestFor(filePath string) bool {
	ext := filepath.Ext(filePath)
	baseName := strings.TrimSuffix(filePath, ext)

	patterns := []string{
		filepath.Join(filepath.Dir(filePath), filepath.Base(baseName)+"_test.go"),
		filepath.Join(g.projectRoot, "internal", filepath.Base(filepath.Dir(filePath))+"_test.go"),
	}

	for _, pattern := range patterns {
		if _, err := os.Stat(pattern); err == nil {
			return true
		}
	}
	return false
}

func (g *GapAnalyzer) estimateComplexity(filePath string) int {
	content, err := os.ReadFile(filepath.Join(g.projectRoot, filePath))
	if err != nil {
		return 1
	}

	complexity := 0
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if strings.Contains(line, "if") || strings.Contains(line, "for") ||
			strings.Contains(line, "switch") || strings.Contains(line, "case") {
			complexity++
		}
	}

	return complexity
}

func (g *GapAnalyzer) identifyCoverageGaps() {
	for path, fc := range g.coverage.Files {
		if fc.CoveragePercent < 70 && len(fc.UncoveredLines) > 5 {
			entry := RiskEntry{
				FilePath:    path,
				RiskLevel:   g.calculateRiskLevel(fc),
				RiskScore:   g.calculateRiskScore(fc),
				RiskFactors: fc.HighRiskPatterns,
				Reason:      fmt.Sprintf("Coverage: %.1f%%, Uncovered lines: %d", fc.CoveragePercent, len(fc.UncoveredLines)),
			}
			entry.RecommendedTests = g.suggestTests(path, fc)

			g.riskMatrix.Entries = append(g.riskMatrix.Entries, entry)
		}
	}
}

func (g *GapAnalyzer) linesToRanges(lines []int) []LineRange {
	if len(lines) == 0 {
		return nil
	}

	sort.Ints(lines)
	ranges := make([]LineRange, 0)
	start := lines[0]
	end := lines[0]

	for i := 1; i < len(lines); i++ {
		if lines[i] == end+1 {
			end = lines[i]
		} else {
			ranges = append(ranges, LineRange{Start: start, End: end})
			start = lines[i]
			end = lines[i]
		}
	}
	ranges = append(ranges, LineRange{Start: start, End: end})

	return ranges
}

func (g *GapAnalyzer) isCriticalPath(filePath string) bool {
	criticalModules := []string{
		"auth", "credential", "key", "secret",
		"deploy", "build", "rollback",
		"health", "heal", "monitor",
		"database", "sql",
	}

	lower := strings.ToLower(filePath)
	for _, module := range criticalModules {
		if strings.Contains(lower, module) {
			return true
		}
	}
	return false
}

func (g *GapAnalyzer) calculateRiskLevel(fc FileCoverage) string {
	score := g.calculateRiskScore(fc)
	if score >= 70 {
		return "high"
	}
	if score >= 40 {
		return "medium"
	}
	return "low"
}

func (g *GapAnalyzer) calculateRiskScore(fc FileCoverage) float64 {
	score := 0.0

	if fc.CoveragePercent < 50 {
		score += 30
	} else if fc.CoveragePercent < 70 {
		score += 20
	} else if fc.CoveragePercent < 80 {
		score += 10
	}

	if !fc.HasTestFile {
		score += 20
	}

	if len(fc.HighRiskPatterns) > 3 {
		score += 25
	} else if len(fc.HighRiskPatterns) > 0 {
		score += 15
	}

	if fc.FunctionsTotal > 0 {
		functionCoverage := float64(fc.FunctionsCovered) / float64(fc.FunctionsTotal) * 100
		if functionCoverage < 50 {
			score += 15
		}
	}

	return score
}

func (g *GapAnalyzer) suggestTests(filePath string, fc FileCoverage) []string {
	suggestions := make([]string, 0)
	baseName := filepath.Base(filePath)
	funcName := strings.TrimSuffix(baseName, ".go")

	if strings.Contains(filePath, "handler") {
		suggestions = append(suggestions,
			fmt.Sprintf("TestHandle%sRequest", strings.Title(funcName)),
			fmt.Sprintf("TestHandle%sValidation", strings.Title(funcName)),
			fmt.Sprintf("TestHandle%sError", strings.Title(funcName)),
		)
	} else if strings.Contains(filePath, "service") {
		suggestions = append(suggestions,
			fmt.Sprintf("Test%sCreate", strings.Title(funcName)),
			fmt.Sprintf("Test%sUpdate", strings.Title(funcName)),
			fmt.Sprintf("Test%sDelete", strings.Title(funcName)),
			fmt.Sprintf("Test%sGet", strings.Title(funcName)),
		)
	} else if strings.Contains(filePath, "engine") {
		suggestions = append(suggestions,
			fmt.Sprintf("Test%sExecute", strings.Title(funcName)),
			fmt.Sprintf("Test%sRollback", strings.Title(funcName)),
		)
	} else {
		suggestions = append(suggestions,
			fmt.Sprintf("Test%sBasic", strings.Title(funcName)),
			fmt.Sprintf("Test%sEdgeCases", strings.Title(funcName)),
		)
	}

	for _, pattern := range fc.HighRiskPatterns {
		switch {
		case strings.Contains(pattern, "goroutine") || strings.Contains(pattern, "sync."):
			suggestions = append(suggestions, fmt.Sprintf("Test%sConcurrency", strings.Title(funcName)))
		case strings.Contains(pattern, "crypto."):
			suggestions = append(suggestions, fmt.Sprintf("Test%sCrypto", strings.Title(funcName)))
		case strings.Contains(pattern, "json."):
			suggestions = append(suggestions, fmt.Sprintf("Test%sParse", strings.Title(funcName)))
		case strings.Contains(pattern, "http."):
			suggestions = append(suggestions, fmt.Sprintf("Test%sHTTP", strings.Title(funcName)))
		}
	}

	return suggestions
}

func (g *GapAnalyzer) identifyChangeGaps() {
	for _, change := range g.gitChanges.ModifiedNoTest {
		entry := RiskEntry{
			FilePath:    change.Path,
			RiskLevel:   g.assessChangeRisk(change),
			RiskScore:   float64(change.Complexity * 10),
			RiskFactors: []string{"modified_no_test", "recent_change"},
			Reason:      "File modified but no corresponding test found",
		}
		entry.RecommendedTests = g.suggestTestsForChange(change)

		g.riskMatrix.Entries = append(g.riskMatrix.Entries, entry)
	}
}

func (g *GapAnalyzer) assessChangeRisk(change FileChange) string {
	if change.Complexity > 10 {
		return "high"
	}
	if change.Complexity > 5 {
		return "medium"
	}
	return "low"
}

func (g *GapAnalyzer) suggestTestsForChange(change FileChange) []string {
	ext := filepath.Ext(change.Path)
	baseName := strings.TrimSuffix(filepath.Base(change.Path), ext)

	return []string{
		fmt.Sprintf("Test%s_%s_Behavior", strings.Title(baseName), change.Status),
		fmt.Sprintf("Test%s_Regression", strings.Title(baseName)),
	}
}

func (g *GapAnalyzer) assessRiskLevels() {
	for i := range g.riskMatrix.Entries {
		entry := &g.riskMatrix.Entries[i]
		entry.RiskLevel = g.calculateRiskLevelForEntry(entry)
	}
}

func (g *GapAnalyzer) calculateRiskLevelForEntry(entry *RiskEntry) string {
	if entry.RiskScore >= 70 || len(entry.RiskFactors) >= 3 {
		return "high"
	}
	if entry.RiskScore >= 40 || len(entry.RiskFactors) >= 2 {
		return "medium"
	}
	return "low"
}

func (g *GapAnalyzer) filterByRisk(level string) []RiskEntry {
	var filtered []RiskEntry
	for _, entry := range g.riskMatrix.Entries {
		if entry.RiskLevel == level {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].RiskScore > filtered[j].RiskScore
	})
	return filtered
}

func (g *GapAnalyzer) generateSummary() AnalysisSummary {
	summary := AnalysisSummary{
		TotalGaps: len(g.riskMatrix.Entries),
		CriticalModules: make([]string, 0),
	}

	var totalCoverage float64
	var fileCount int

	for _, fc := range g.coverage.Files {
		totalCoverage += fc.CoveragePercent
		fileCount++
	}

	if fileCount > 0 {
		summary.AvgCoverage = totalCoverage / float64(fileCount)
	}

	for _, entry := range g.riskMatrix.Entries {
		switch entry.RiskLevel {
		case "high":
			summary.HighPriorityCount++
		case "medium":
			summary.MediumPriorityCount++
		case "low":
			summary.LowPriorityCount++
		}
	}

	critical := make(map[string]bool)
	for _, entry := range g.riskMatrix.Entries {
		if entry.RiskLevel == "high" && g.isCriticalPath(entry.FilePath) {
			critical[filepath.Base(filepath.Dir(entry.FilePath))] = true
		}
	}
	for module := range critical {
		summary.CriticalModules = append(summary.CriticalModules, module)
	}

	return summary
}

func (g *GapAnalyzer) ExportJSON(path string) error {
	result, err := g.Analyze()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (g *GapAnalyzer) GetCriticalFunctions() []string {
	criticalFuncs := make([]string, 0)

	for path := range g.coverage.Files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for _, pattern := range criticalPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllString(string(content), -1)
			for _, match := range matches {
				criticalFuncs = append(criticalFuncs, fmt.Sprintf("%s: %s", path, match))
			}
		}
	}

	return criticalFuncs
}

func (g *GapAnalyzer) GetUntestedCriticalPaths() []CoverageGap {
	gaps := make([]CoverageGap, 0)

	for path, fc := range g.coverage.Files {
		if !g.isCriticalPath(path) {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for _, pattern := range criticalPatterns {
			re := regexp.MustCompile(pattern)
			if re.MatchString(string(content)) {
				gap := CoverageGap{
					FilePath:        path,
					UncoveredLines:  fc.UncoveredLines,
					UncoveredRanges: g.linesToRanges(fc.UncoveredLines),
					IsCriticalPath:  true,
				}
				gaps = append(gaps, gap)
				break
			}
		}
	}

	return gaps
}
