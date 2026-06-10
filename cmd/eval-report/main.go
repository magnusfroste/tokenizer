// Command eval-report runs the routing eval dataset and writes a report
// artifact (report.json + report.txt) for CI to upload (ISSUE-050). It exits
// non-zero if the pass rate falls below -min-pass, so it can double as a gate.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/magnusfroste/tokenizer/internal/evals"
)

func main() {
	var (
		dataset = flag.String("dataset", "evals/dataset-v1.yaml", "path to the eval dataset YAML")
		outDir  = flag.String("out", "eval-report", "output directory for the report artifact")
		minPass = flag.Float64("min-pass", 0.0, "fail if pass rate is below this fraction (0 disables)")
	)
	flag.Parse()

	report, err := generate(*dataset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := writeReport(report, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(evals.FormatReport(report))
	fmt.Printf("Report written to %s/\n", *outDir)

	if *minPass > 0 && report.PassRate() < *minPass {
		fmt.Fprintf(os.Stderr, "pass rate %.3f below threshold %.3f\n", report.PassRate(), *minPass)
		os.Exit(1)
	}
}

// generate loads the dataset and runs it through the routing engine.
func generate(datasetPath string) (evals.Report, error) {
	rn, err := evals.NewRunner()
	if err != nil {
		return evals.Report{}, fmt.Errorf("new runner: %w", err)
	}
	ds, err := evals.LoadDataset(datasetPath)
	if err != nil {
		return evals.Report{}, fmt.Errorf("load dataset: %w", err)
	}
	return rn.Run(ds)
}

// writeReport writes report.json and report.txt into dir, creating it if needed.
func writeReport(report evals.Report, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), jsonBytes, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.txt"), []byte(evals.FormatReport(report)), 0o644)
}
