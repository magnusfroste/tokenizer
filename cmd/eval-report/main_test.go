package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/magnusfroste/tokenizer/internal/evals"
)

const datasetPath = "../../evals/dataset-v1.yaml"

func TestGenerateRunsDataset(t *testing.T) {
	report, err := generate(datasetPath)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if report.Total < 50 {
		t.Errorf("expected at least 50 cases, got %d", report.Total)
	}
	if len(report.Results) != report.Total {
		t.Errorf("results = %d, want %d", len(report.Results), report.Total)
	}
}

func TestWriteReportProducesArtifacts(t *testing.T) {
	report, err := generate(datasetPath)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	dir := t.TempDir()
	if err := writeReport(report, filepath.Join(dir, "eval-report")); err != nil {
		t.Fatalf("writeReport: %v", err)
	}

	jsonPath := filepath.Join(dir, "eval-report", "report.json")
	txtPath := filepath.Join(dir, "eval-report", "report.txt")

	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	var parsed evals.Report
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("report.json is not valid JSON: %v", err)
	}
	if parsed.Total != report.Total {
		t.Errorf("round-tripped total = %d, want %d", parsed.Total, report.Total)
	}

	if info, err := os.Stat(txtPath); err != nil || info.Size() == 0 {
		t.Errorf("report.txt missing or empty: err=%v", err)
	}
}

func TestGenerateMissingDatasetErrors(t *testing.T) {
	if _, err := generate("does-not-exist.yaml"); err == nil {
		t.Fatal("expected error for missing dataset")
	}
}
