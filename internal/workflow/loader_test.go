// Package workflow_test 测试工作流加载器
package workflow_test

import (
	"testing"

	"github.com/dministrator/symphony/internal/workflow"
)

func TestParseEmptyContent(t *testing.T) {
	content := []byte("")
	def, err := workflow.Parse(content)
	if err != nil {
		t.Fatalf("failed to parse empty content: %v", err)
	}

	if len(def.Config) != 0 {
		t.Errorf("expected empty config, got %v", def.Config)
	}

	if def.PromptTemplate != "" {
		t.Errorf("expected empty prompt template, got %s", def.PromptTemplate)
	}
}

func TestParseNoFrontMatter(t *testing.T) {
	content := []byte("# Hello World\n\nThis is a test prompt.")
	def, err := workflow.Parse(content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(def.Config) != 0 {
		t.Errorf("expected empty config, got %v", def.Config)
	}

	expected := "# Hello World\n\nThis is a test prompt."
	if def.PromptTemplate != expected {
		t.Errorf("expected prompt '%s', got '%s'", expected, def.PromptTemplate)
	}
}

func TestParseWithFrontMatter(t *testing.T) {
	content := []byte(`---
tracker:
  kind: linear
  project_slug: TEST
polling:
  interval_ms: 60000
---

# Task Prompt

Please work on this issue.
`)

	def, err := workflow.Parse(content)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if def.Config == nil {
		t.Fatal("expected non-nil config")
	}

	tracker, ok := def.Config["tracker"].(map[string]any)
	if !ok {
		t.Fatal("expected tracker config")
	}

	if tracker["kind"] != "linear" {
		t.Errorf("expected tracker kind 'linear', got %v", tracker["kind"])
	}

	if tracker["project_slug"] != "TEST" {
		t.Errorf("expected project slug 'TEST', got %v", tracker["project_slug"])
	}

	expectedPrompt := "# Task Prompt\n\nPlease work on this issue."
	if def.PromptTemplate != expectedPrompt {
		t.Errorf("expected prompt '%s', got '%s'", expectedPrompt, def.PromptTemplate)
	}
}

func TestParseInvalidFrontMatter(t *testing.T) {
	content := []byte(`---
invalid yaml content [
---

# Prompt
`)

	_, err := workflow.Parse(content)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseFrontMatterNotMap(t *testing.T) {
	content := []byte(`---
- item1
- item2
---

# Prompt
`)

	_, err := workflow.Parse(content)
	if err == nil {
		t.Error("expected error for non-map front matter")
	}
}

func TestParseMissingClosing(t *testing.T) {
	content := []byte(`---
tracker:
  kind: linear

# Prompt
`)

	_, err := workflow.Parse(content)
	if err == nil {
		t.Error("expected error for missing closing ---")
	}
}