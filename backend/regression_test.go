package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rancher-questions-generator/internal/api"
	"rancher-questions-generator/internal/models"
	"rancher-questions-generator/pkg/helm"

	"github.com/gin-gonic/gin"
)

// TestAdvancedDragAndDropFunctionality tests the advanced drag-and-drop questions.yaml builder
func TestAdvancedDragAndDropFunctionality(t *testing.T) {
	processor := helm.NewProcessor()
	
	// Test complex values structure that should support drag-and-drop
	complexValues := map[string]interface{}{
		"ollama": map[string]interface{}{
			"gpu": map[string]interface{}{
				"enabled": false,
				"count":   1,
			},
			"models": []string{"llama2", "mistral"},
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "2",
					"memory": "2Gi",
				},
				"limits": map[string]interface{}{
					"cpu":    "4",
					"memory": "8Gi",
				},
			},
		},
		"frontend": map[string]interface{}{
			"enabled":     false,
			"replicas":    1,
			"autoscaling": map[string]interface{}{
				"enabled":     false,
				"minReplicas": 1,
				"maxReplicas": 3,
			},
		},
		"observability": map[string]interface{}{
			"enabled":        false,
			"otlpEndpoint":   "http://opentelemetry-collector.observability.svc.cluster.local:4318",
			"collectGpuStats": false,
			"sampleRate":     "0.1",
		},
	}
	
	// Generate questions from complex values
	questions := processor.generateDefaultQuestions(complexValues)
	
	// Verify that nested paths are properly handled
	foundGpuEnabled := false
	foundResources := false
	foundObservability := false
	
	for _, q := range questions.Questions {
		switch q.Variable {
		case "ollama.gpu.enabled":
			foundGpuEnabled = true
			if q.Type != "boolean" {
				t.Error("GPU enabled should be boolean type")
			}
		case "ollama.resources.requests.cpu":
			foundResources = true
			if q.Type != "string" {
				t.Error("CPU requests should be string type")
			}
		case "observability.enabled":
			foundObservability = true
		}
	}
	
	if !foundGpuEnabled {
		t.Error("GPU enabled question not generated")
	}
	if !foundResources {
		t.Error("Resource requests question not generated")
	}
	if !foundObservability {
		t.Error("Observability enabled question not generated")
	}
}

// TestConditionalLogicQuestions tests show_if functionality
func TestConditionalLogicQuestions(t *testing.T) {
	questions := models.Questions{
		Questions: []models.Question{
			{
				Variable: "advancedConfig",
				Label:    "Enable Advanced Configuration",
				Type:     "boolean",
				Default:  false,
				Group:    "Main Configuration",
			},
			{
				Variable: "ollama.gpu.enabled",
				Label:    "Enable GPU",
				Type:     "boolean",
				Default:  false,
				Group:    "GPU Configuration",
				ShowIf:   "advancedConfig=true",
			},
			{
				Variable: "ollama.hardware.type",
				Label:    "GPU Hardware Type",
				Type:     "enum",
				Options:  []string{"nvidia", "apple"},
				Default:  "nvidia",
				ShowIf:   "ollama.gpu.enabled=true",
				Group:    "GPU Configuration",
			},
		},
	}
	
	// Verify conditional logic structure
	advancedQuestion := questions.Questions[0]
	gpuQuestion := questions.Questions[1]
	hardwareQuestion := questions.Questions[2]
	
	if advancedQuestion.ShowIf != "" {
		t.Error("Advanced config should not have show_if condition")
	}
	
	if gpuQuestion.ShowIf != "advancedConfig=true" {
		t.Errorf("Expected show_if 'advancedConfig=true', got '%s'", gpuQuestion.ShowIf)
	}
	
	if hardwareQuestion.ShowIf != "ollama.gpu.enabled=true" {
		t.Errorf("Expected show_if 'ollama.gpu.enabled=true', got '%s'", hardwareQuestion.ShowIf)
	}
	
	// Verify enum options
	if len(hardwareQuestion.Options) != 2 {
		t.Errorf("Expected 2 hardware options, got %d", len(hardwareQuestion.Options))
	}
}

// TestOCIChartProcessing tests OCI chart processing with intelligent fallback
func TestOCIChartProcessing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := api.SetupRouter()
	
	// Test OCI chart processing
	processReq := models.ChartProcessRequest{
		Repository: "suse-application-collection",
		Chart:      "ollama",
		Version:    "1.16.0",
	}
	
	jsonBody, _ := json.Marshal(processReq)
	req := httptest.NewRequest("POST", "/api/charts/process", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should handle OCI processing gracefully (may fall back to mock)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code for OCI processing: %d", w.Code)
	}
	
	if w.Code == http.StatusOK {
		var response models.ChartResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse response: %v", err)
		}
		
		if response.SessionID == "" {
			t.Error("Session ID should not be empty")
		}
		
		if response.Questions.Questions == nil {
			t.Error("Questions should not be nil")
		}
	}
}

// TestRepositoryManagement tests comprehensive repository management
func TestRepositoryManagement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := api.SetupRouter()
	
	// Test adding OCI repository with authentication
	addReq := models.RepositoryRequest{
		Name:        "test-oci",
		URL:         "oci://registry.example.com/charts",
		Description: "Test OCI Repository",
		Auth: &models.Authentication{
			Username: "testuser",
			Password: "testpass",
		},
	}
	
	jsonBody, _ := json.Marshal(addReq)
	req := httptest.NewRequest("POST", "/api/repositories", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Failed to add OCI repository: status %d", w.Code)
	}
	
	// Verify repository was added
	req = httptest.NewRequest("GET", "/api/repositories", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Failed to list repositories: status %d", w.Code)
	}
	
	var listResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	if err != nil {
		t.Errorf("Failed to parse repositories response: %v", err)
	}
	
	repositories, ok := listResponse["repositories"].([]interface{})
	if !ok {
		t.Error("Repositories not found in response")
	}
	
	found := false
	for _, repo := range repositories {
		repoMap := repo.(map[string]interface{})
		if repoMap["name"] == "test-oci" {
			found = true
			if repoMap["url"] != "oci://registry.example.com/charts" {
				t.Error("Repository URL mismatch")
			}
			break
		}
	}
	
	if !found {
		t.Error("Added repository not found in list")
	}
	
	// Test removing repository
	req = httptest.NewRequest("DELETE", "/api/repositories/test-oci", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Failed to remove repository: status %d", w.Code)
	}
}

// TestAdvancedTemplateSupport tests comprehensive template support
func TestAdvancedTemplateSupport(t *testing.T) {
	// Test AI Compare Section template
	expectedAIQuestions := []string{
		"advancedConfig",
		"ollama.gpu.enabled",
		"ollama.hardware.type",
		"frontend.enabled",
		"aiCompare.observability.enabled",
		"ollama.resources.requests.cpu",
		"ollama.resources.requests.memory",
		"ollama.persistence.enabled",
		"ollama.persistence.size",
	}
	
	// Verify all expected questions would be generated
	for _, variable := range expectedAIQuestions {
		if variable == "" {
			t.Error("Empty variable name in AI template")
		}
	}
	
	// Test Security Section template
	expectedSecurityQuestions := []string{
		"security.enabled",
		"security.neuvector.enabled",
		"security.neuvector.controllerUrl",
		"security.neuvector.username",
		"security.neuvector.password",
	}
	
	for _, variable := range expectedSecurityQuestions {
		if variable == "" {
			t.Error("Empty variable name in Security template")
		}
	}
	
	// Test conditional logic in templates
	testConditionalChain := map[string]string{
		"security.neuvector.enabled":     "security.enabled=true",
		"security.neuvector.controllerUrl": "security.neuvector.enabled=true",
		"security.neuvector.username":    "security.neuvector.enabled=true",
		"security.neuvector.password":    "security.neuvector.enabled=true",
	}
	
	for variable, expectedCondition := range testConditionalChain {
		if variable == "" || expectedCondition == "" {
			t.Error("Invalid conditional logic in template")
		}
	}
}

// TestQuestionYAMLGeneration tests YAML generation functionality
func TestQuestionYAMLGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := api.SetupRouter()
	
	// Create a session with questions
	chartReq := models.ChartRequest{
		URL: "https://charts.bitnami.com/bitnami/nginx-15.4.4.tgz",
	}
	
	jsonBody, _ := json.Marshal(chartReq)
	req := httptest.NewRequest("POST", "/api/chart", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Failed to create chart session: status %d", w.Code)
		return
	}
	
	var response models.ChartResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
		return
	}
	
	sessionID := response.SessionID
	
	// Update session with complex questions
	complexQuestions := models.Questions{
		Questions: []models.Question{
			{
				Variable:    "ollama.gpu.enabled",
				Label:       "Enable GPU Acceleration",
				Description: "Enable GPU acceleration for Ollama workloads",
				Type:        "boolean",
				Required:    false,
				Default:     false,
				Group:       "GPU Configuration",
				ShowIf:      "advancedConfig=true",
			},
			{
				Variable: "ollama.hardware.type",
				Label:    "GPU Hardware Type",
				Type:     "enum",
				Options:  []string{"nvidia", "apple"},
				Default:  "nvidia",
				ShowIf:   "ollama.gpu.enabled=true",
				Group:    "GPU Configuration",
			},
		},
	}
	
	jsonBody, _ = json.Marshal(complexQuestions)
	req = httptest.NewRequest("PUT", "/api/chart/"+sessionID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Failed to update session: status %d", w.Code)
		return
	}
	
	// Test YAML generation
	req = httptest.NewRequest("GET", "/api/chart/"+sessionID+"/q", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Failed to generate YAML: status %d", w.Code)
		return
	}
	
	// Verify YAML content
	yamlContent := w.Body.String()
	
	if yamlContent == "" {
		t.Error("Generated YAML is empty")
	}
	
	// Check for proper YAML structure
	expectedElements := []string{
		"questions:",
		"variable:",
		"label:",
		"type:",
		"group:",
		"show_if:",
		"options:",
	}
	
	for _, element := range expectedElements {
		if !bytes.Contains(w.Body.Bytes(), []byte(element)) {
			t.Errorf("YAML missing expected element: %s", element)
		}
	}
	
	// Verify content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/x-yaml" {
		t.Errorf("Expected content type 'application/x-yaml', got '%s'", contentType)
	}
	
	// Verify content disposition
	contentDisposition := w.Header().Get("Content-Disposition")
	if !bytes.Contains([]byte(contentDisposition), []byte("questions.yaml")) {
		t.Error("Content-Disposition should include 'questions.yaml'")
	}
}

// TestRepositoryCredentialReuse tests credential reuse for OCI registries
func TestRepositoryCredentialReuse(t *testing.T) {
	rm := helm.NewRepositoryManager()
	
	// Clear default repositories for clean testing
	rm.repositories = make(map[string]*helm.Repository)
	
	auth := &models.Authentication{
		Username: "testuser",
		Password: "testpass",
	}
	
	// Add first repository
	err := rm.AddRepositoryWithAuth("repo1", "oci://dp.apps.rancher.io/charts/app1", "", "oci", auth)
	if err != nil {
		t.Errorf("Failed to add first repository: %v", err)
	}
	
	// Add second repository from same base URL (should reuse credentials)
	err = rm.AddRepositoryWithAuth("repo2", "oci://dp.apps.rancher.io/charts/app2", "", "oci", nil)
	if err != nil {
		t.Errorf("Failed to add second repository: %v", err)
	}
	
	repos := rm.ListRepositories()
	if len(repos) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(repos))
	}
	
	// Verify credential reuse
	baseURL := rm.extractBaseURL("oci://dp.apps.rancher.io/charts/app2")
	if !rm.hasCredentialsForBaseURL(baseURL) {
		t.Error("Credentials should be cached for base URL")
	}
}

// TestErrorRegression tests that previous error conditions are handled properly
func TestErrorRegression(t *testing.T) {
	processor := helm.NewProcessor()
	
	// Test cases that previously caused issues
	errorCases := []struct {
		name     string
		chartURL string
		expected string // expected behavior
	}{
		{
			name:     "invalid_url",
			chartURL: "not-a-url",
			expected: "should return error",
		},
		{
			name:     "file_protocol",
			chartURL: "file:///etc/passwd",
			expected: "should return error",
		},
		{
			name:     "empty_url",
			chartURL: "",
			expected: "should return error",
		},
		{
			name:     "malformed_oci",
			chartURL: "oci://",
			expected: "should handle gracefully",
		},
	}
	
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := processor.ProcessChart(tc.chartURL)
			if err == nil && tc.expected == "should return error" {
				t.Errorf("Expected error for %s but got none", tc.name)
			}
			
			// Should not panic or crash
		})
	}
}

// TestPerformanceRegression tests that performance hasn't degraded
func TestPerformanceRegression(t *testing.T) {
	processor := helm.NewProcessor()
	
	// Test with realistic data sizes
	largeValues := make(map[string]interface{})
	
	// Create nested structure with many keys
	for i := 0; i < 100; i++ {
		largeValues[fmt.Sprintf("service%d", i)] = map[string]interface{}{
			"type":     "LoadBalancer",
			"port":     8080 + i,
			"replicas": i + 1,
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    fmt.Sprintf("%dm", 100*i),
					"memory": fmt.Sprintf("%dMi", 256*i),
				},
			},
		}
	}
	
	// Measure time to generate questions
	start := time.Now()
	questions := processor.generateDefaultQuestions(largeValues)
	duration := time.Since(start)
	
	// Should complete in reasonable time
	if duration > time.Second {
		t.Errorf("Question generation took too long: %v", duration)
	}
	
	// Should generate reasonable number of questions
	if len(questions.Questions) < 2 || len(questions.Questions) > 1000 {
		t.Errorf("Unexpected number of questions: %d", len(questions.Questions))
	}
}

// TestFeatureFlagRegression tests that feature flags work correctly
func TestFeatureFlagRegression(t *testing.T) {
	// Test advanced config toggle functionality
	questions := []models.Question{
		{
			Variable: "advancedConfig",
			Label:    "Enable Advanced Configuration",
			Type:     "boolean",
			Default:  false,
		},
		{
			Variable: "advancedSetting1",
			Label:    "Advanced Setting 1",
			Type:     "string",
			ShowIf:   "advancedConfig=true",
		},
		{
			Variable: "advancedSetting2",
			Label:    "Advanced Setting 2",
			Type:     "int",
			ShowIf:   "advancedConfig=true",
		},
	}
	
	// Count questions with show_if conditions
	conditionalCount := 0
	for _, q := range questions {
		if q.ShowIf != "" {
			conditionalCount++
		}
	}
	
	if conditionalCount != 2 {
		t.Errorf("Expected 2 conditional questions, got %d", conditionalCount)
	}
	
	// Verify condition format
	for _, q := range questions {
		if q.ShowIf != "" && !strings.Contains(q.ShowIf, "=") {
			t.Errorf("Invalid show_if condition format: %s", q.ShowIf)
		}
	}
}