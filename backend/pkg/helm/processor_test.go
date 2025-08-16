package helm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rancher-questions-generator/internal/models"
)

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()
	if processor == nil {
		t.Fatal("NewProcessor() returned nil")
	}
	if processor.tempDir != "/tmp/helm-charts" {
		t.Errorf("Expected tempDir to be '/tmp/helm-charts', got %s", processor.tempDir)
	}
}

func TestGenerateMockValues(t *testing.T) {
	processor := NewProcessor()
	
	tests := []struct {
		chartName    string
		expectedKeys []string
	}{
		{
			chartName:    "ollama",
			expectedKeys: []string{"replicaCount", "image", "service", "resources", "persistence", "ollama"},
		},
		{
			chartName:    "prometheus",
			expectedKeys: []string{"replicaCount", "image", "service", "persistence", "resources", "retention"},
		},
		{
			chartName:    "unknown-chart",
			expectedKeys: []string{"replicaCount", "image", "service", "resources", "persistence", "autoscaling"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.chartName, func(t *testing.T) {
			values := processor.generateMockValues(tt.chartName)
			if values == "" {
				t.Error("generateMockValues returned empty string")
			}
			
			// Check if expected keys are present in the YAML
			for _, key := range tt.expectedKeys {
				if !strings.Contains(values, key) {
					t.Errorf("Expected key '%s' not found in generated values", key)
				}
			}
		})
	}
}

func TestExtractTarGz(t *testing.T) {
	processor := NewProcessor()
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Test with non-existent file
	err := processor.extractTarGz("non-existent.tgz", tempDir)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGenerateDefaultQuestions(t *testing.T) {
	processor := NewProcessor()
	
	tests := []struct {
		name     string
		values   map[string]interface{}
		expected int // expected number of questions
	}{
		{
			name: "basic values",
			values: map[string]interface{}{
				"service": map[string]interface{}{
					"type": "LoadBalancer",
				},
			},
			expected: 3, // name, namespace, service.type
		},
		{
			name: "values with persistence",
			values: map[string]interface{}{
				"persistence": map[string]interface{}{
					"storageClass": "fast",
				},
			},
			expected: 3, // name, namespace, persistence.storageClass
		},
		{
			name:     "empty values",
			values:   map[string]interface{}{},
			expected: 2, // name, namespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questions := processor.generateDefaultQuestions(tt.values)
			if len(questions.Questions) != tt.expected {
				t.Errorf("Expected %d questions, got %d", tt.expected, len(questions.Questions))
			}
			
			// Verify basic questions are always present
			foundName := false
			foundNamespace := false
			for _, q := range questions.Questions {
				if q.Variable == "name" {
					foundName = true
				}
				if q.Variable == "namespace" {
					foundNamespace = true
				}
			}
			
			if !foundName {
				t.Error("Expected 'name' question not found")
			}
			if !foundNamespace {
				t.Error("Expected 'namespace' question not found")
			}
		})
	}
}

func TestHasNestedKey(t *testing.T) {
	processor := NewProcessor()
	
	data := map[string]interface{}{
		"service": map[string]interface{}{
			"type": "LoadBalancer",
			"port": 8080,
		},
		"simple": "value",
	}
	
	tests := []struct {
		name     string
		keys     []string
		expected bool
	}{
		{
			name:     "existing nested key",
			keys:     []string{"service", "type"},
			expected: true,
		},
		{
			name:     "non-existing nested key",
			keys:     []string{"service", "missing"},
			expected: false,
		},
		{
			name:     "simple key",
			keys:     []string{"simple"},
			expected: true,
		},
		{
			name:     "non-existing simple key",
			keys:     []string{"missing"},
			expected: false,
		},
		{
			name:     "deep non-existing path",
			keys:     []string{"service", "type", "deep"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.hasNestedKey(data, tt.keys...)
			if result != tt.expected {
				t.Errorf("hasNestedKey(%v) = %v, expected %v", tt.keys, result, tt.expected)
			}
		})
	}
}

func TestMergeQuestions(t *testing.T) {
	processor := NewProcessor()
	
	existing := models.Questions{
		Questions: []models.Question{
			{
				Variable: "existing.var",
				Label:    "Existing Variable",
				Type:     "string",
			},
		},
	}
	
	defaults := models.Questions{
		Questions: []models.Question{
			{
				Variable: "existing.var",
				Label:    "Should not override",
				Type:     "boolean",
			},
			{
				Variable: "new.var",
				Label:    "New Variable",
				Type:     "int",
			},
		},
	}
	
	merged := processor.mergeQuestions(existing, defaults)
	
	if len(merged.Questions) != 2 {
		t.Errorf("Expected 2 questions after merge, got %d", len(merged.Questions))
	}
	
	// Check that existing question was not overridden
	for _, q := range merged.Questions {
		if q.Variable == "existing.var" && q.Type != "string" {
			t.Error("Existing question was incorrectly overridden")
		}
		if q.Variable == "new.var" && q.Type != "int" {
			t.Error("New question was not added correctly")
		}
	}
}

func TestFindFile(t *testing.T) {
	processor := NewProcessor()
	
	// Create a temporary directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	os.MkdirAll(subDir, 0755)
	
	// Create test files
	testFile := filepath.Join(tempDir, "values.yaml")
	subFile := filepath.Join(subDir, "questions.yaml")
	
	os.WriteFile(testFile, []byte("test"), 0644)
	os.WriteFile(subFile, []byte("test"), 0644)
	
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "find values.yaml",
			filename: "values.yaml",
			expected: testFile,
		},
		{
			name:     "find questions.yaml in subdirectory",
			filename: "questions.yaml",
			expected: subFile,
		},
		{
			name:     "non-existent file",
			filename: "missing.yaml",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.findFile(tempDir, tt.filename)
			if result != tt.expected {
				t.Errorf("findFile(%s, %s) = %s, expected %s", tempDir, tt.filename, result, tt.expected)
			}
		})
	}
}

func TestCreateMockOCIChart(t *testing.T) {
	processor := NewProcessor()
	
	tests := []struct {
		name        string
		ociURL      string
		expectedDir string
	}{
		{
			name:   "ollama chart",
			ociURL: "oci://dp.apps.rancher.io/charts/ollama:1.16.0",
		},
		{
			name:   "prometheus chart",
			ociURL: "oci://registry.example.com/charts/prometheus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := processor.createMockOCIChart(tt.ociURL)
			if err != nil {
				t.Fatalf("createMockOCIChart failed: %v", err)
			}
			
			// Check if directory was created
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("Expected directory %s was not created", dir)
			}
			
			// Check if values.yaml exists
			valuesPath := filepath.Join(dir, "values.yaml")
			if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
				t.Errorf("Expected values.yaml was not created at %s", valuesPath)
			}
			
			// Cleanup
			os.RemoveAll(dir)
		})
	}
}

// Benchmark tests
func BenchmarkGenerateDefaultQuestions(b *testing.B) {
	processor := NewProcessor()
	values := map[string]interface{}{
		"service": map[string]interface{}{
			"type": "LoadBalancer",
		},
		"persistence": map[string]interface{}{
			"storageClass": "fast",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.generateDefaultQuestions(values)
	}
}

func BenchmarkHasNestedKey(b *testing.B) {
	processor := NewProcessor()
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "value",
			},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.hasNestedKey(data, "level1", "level2", "level3")
	}
}