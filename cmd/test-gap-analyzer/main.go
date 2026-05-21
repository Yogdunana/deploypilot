package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	gap "github.com/Yogdunana/deploypilot/internal/testgap"
)

var (
	flagProjectRoot  = flag.String("project", ".", "Project root directory")
	flagOutput       = flag.String("output", "test-gap-report.json", "Output file path")
	flagReport       = flag.String("report", "", "Generate markdown report to file")
	flagGenTests     = flag.Bool("generate-tests", false, "Generate test files for identified gaps")
	flagMinRisk      = flag.String("min-risk", "medium", "Minimum risk level to report (high, medium, low)")
	flagCriticalOnly = flag.Bool("critical-only", false, "Only analyze critical paths")
)

func main() {
	flag.Parse()

	projectRoot, err := filepath.Abs(*flagProjectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid project path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Test Gap Analyzer\n")
	fmt.Printf("================\n\n")
	fmt.Printf("Project: %s\n", projectRoot)
	fmt.Printf("Analyzing...\n\n")

	analyzer := gap.NewGapAnalyzer(projectRoot)
	result, err := analyzer.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during analysis: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Analysis Complete!\n\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total Files Analyzed: %d\n", result.TotalFilesAnalyzed)
	fmt.Printf("  Average Coverage: %.1f%%\n", result.Summary.AvgCoverage)
	fmt.Printf("  Total Gaps Found: %d\n\n", result.Summary.TotalGaps)
	fmt.Printf("Risk Distribution:\n")
	fmt.Printf("  High Risk:   %d\n", result.Summary.HighPriorityCount)
	fmt.Printf("  Medium Risk: %d\n", result.Summary.MediumPriorityCount)
	fmt.Printf("  Low Risk:    %d\n\n", result.Summary.LowPriorityCount)

	if len(result.Summary.CriticalModules) > 0 {
		fmt.Printf("Critical Modules:\n")
		for _, module := range result.Summary.CriticalModules {
			fmt.Printf("  - %s\n", module)
		}
		fmt.Printf("\n")
	}

	if len(result.HighRiskGaps) > 0 {
		fmt.Printf("High Risk Gaps (%d):\n", len(result.HighRiskGaps))
		for i, gap := range result.HighRiskGaps {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(result.HighRiskGaps)-10)
				break
			}
			relPath, _ := filepath.Rel(projectRoot, gap.FilePath)
			fmt.Printf("  [%d] %s\n", i+1, relPath)
			fmt.Printf("      Score: %.1f | Reason: %s\n", gap.RiskScore, gap.Reason)
			if len(gap.RecommendedTests) > 0 {
				fmt.Printf("      Suggested tests: %s\n", gap.RecommendedTests[0])
			}
		}
		fmt.Printf("\n")
	}

	if *flagOutput != "" {
		outputPath, _ := filepath.Abs(*flagOutput)
		if err := analyzer.ExportJSON(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to export JSON: %v\n", err)
		} else {
			fmt.Printf("JSON report saved to: %s\n\n", outputPath)
		}
	}

	if *flagReport != "" {
		generator := gap.NewTestGenerator(projectRoot)
		report := generator.GenerateReport(result)
		reportPath, _ := filepath.Abs(*flagReport)
		if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write report: %v\n", err)
		} else {
			fmt.Printf("Markdown report saved to: %s\n\n", reportPath)
		}
	}

	if *flagGenTests {
		fmt.Printf("Generating tests for high-risk gaps...\n")
		generator := gap.NewTestGenerator(projectRoot)
		generated := 0
		for _, riskEntry := range result.HighRiskGaps {
			files, err := generator.GenerateForGap(&gap.CoverageGap{
				FilePath: riskEntry.FilePath,
			})
			if err != nil {
				continue
			}
			generated += len(files)
		}
		fmt.Printf("Generated %d test files.\n\n", generated)
	}

	if *flagCriticalOnly {
		criticalFuncs := analyzer.GetCriticalFunctions()
		if len(criticalFuncs) > 0 {
			fmt.Printf("Critical Functions Without Tests:\n")
			for _, fn := range criticalFuncs {
				fmt.Printf("  - %s\n", fn)
			}
		}

		untestedPaths := analyzer.GetUntestedCriticalPaths()
		if len(untestedPaths) > 0 {
			fmt.Printf("\nUntested Critical Paths:\n")
			for _, path := range untestedPaths {
				relPath, _ := filepath.Rel(projectRoot, path.FilePath)
				fmt.Printf("  - %s (%d uncovered lines)\n", relPath, len(path.UncoveredLines))
			}
		}
	}

	minRisk := *flagMinRisk
	gapCount := len(result.HighRiskGaps)
	if minRisk == "low" || minRisk == "medium" {
		gapCount += len(result.MediumRiskGaps)
	}
	if minRisk == "low" {
		gapCount += len(result.LowRiskGaps)
	}

	fmt.Printf("Total gaps meeting criteria: %d\n", gapCount)

	if result.Summary.HighPriorityCount > 0 {
		os.Exit(2)
	}
}
