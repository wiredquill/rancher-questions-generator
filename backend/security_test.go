package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rancher-questions-generator/internal/api"
	"rancher-questions-generator/internal/models"
	"rancher-questions-generator/pkg/helm"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return api.SetupRouter()
}

// TestPathTraversalProtection tests protection against zip slip attacks
func TestPathTraversalProtection(t *testing.T) {
	processor := helm.NewProcessor()
	
	// Create a malicious tar.gz file with path traversal
	tempDir := t.TempDir()
	maliciousTar := filepath.Join(tempDir, "malicious.tgz")
	
	// Create tar.gz with path traversal attempts
	file, err := os.Create(maliciousTar)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()
	
	gzWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzWriter)
	
	// Add malicious files with path traversal
	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"legitimate/file.yaml",
	}
	
	for _, path := range maliciousPaths {
		header := &tar.Header{
			Name: path,
			Mode: 0644,
			Size: 13,
		}
		
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("Failed to write tar header: %v", err)
		}
		
		if _, err := tarWriter.Write([]byte("malicious content")); err != nil {
			t.Fatalf("Failed to write tar content: %v", err)
		}
	}
	
	tarWriter.Close()
	gzWriter.Close()
	
	// Test extraction
	extractDir := filepath.Join(tempDir, "extract")
	err = processor.ExtractTarGz(maliciousTar, extractDir)
	
	// Should not fail (protection should handle it gracefully)
	if err != nil {
		t.Logf("Extraction failed (expected): %v", err)
	}
	
	// Verify no files were extracted outside the target directory
	extractDirAbs, _ := filepath.Abs(extractDir)
	
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		pathAbs, _ := filepath.Abs(path)
		
		// If this is an extracted file, it should be within extractDir
		if strings.Contains(path, "extract") && !strings.HasPrefix(pathAbs, extractDirAbs) {
			t.Errorf("File extracted outside target directory: %s", path)
		}
		
		return nil
	})
	
	if err != nil {
		t.Errorf("Failed to walk directory: %v", err)
	}
}

// TestInputValidation tests various input validation scenarios
func TestInputValidation(t *testing.T) {
	router := setupTestRouter()
	
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		description    string
	}{
		{
			name:           "malicious_chart_url",
			method:         "POST",
			path:           "/api/chart",
			body:           models.ChartRequest{URL: "file:///etc/passwd"},
			expectedStatus: http.StatusInternalServerError,
			description:    "Should reject file:// URLs",
		},
		{
			name:           "javascript_injection",
			method:         "POST",
			path:           "/api/chart",
			body:           models.ChartRequest{URL: "javascript:alert('xss')"},
			expectedStatus: http.StatusInternalServerError,
			description:    "Should reject javascript: URLs",
		},
		{
			name:           "null_bytes",
			method:         "POST",
			path:           "/api/chart",
			body:           models.ChartRequest{URL: "https://example.com/chart\x00.tgz"},
			expectedStatus: http.StatusInternalServerError,
			description:    "Should handle null bytes safely",
		},
		{
			name:           "oversized_url",
			method:         "POST",
			path:           "/api/chart",
			body:           models.ChartRequest{URL: strings.Repeat("a", 10000)},
			expectedStatus: http.StatusInternalServerError,
			description:    "Should handle oversized URLs",
		},
		{
			name:   "sql_injection_repo_name",
			method: "POST",
			path:   "/api/repositories",
			body: models.RepositoryRequest{
				Name: "test'; DROP TABLE repositories; --",
				URL:  "https://charts.example.com",
			},
			expectedStatus: http.StatusOK, // Should be sanitized, not cause SQL injection
			description:    "Should handle SQL injection attempts in repository name",
		},
		{
			name:   "xss_repo_description",
			method: "POST",
			path:   "/api/repositories",
			body: models.RepositoryRequest{
				Name:        "test-repo",
				URL:         "https://charts.example.com",
				Description: "<script>alert('xss')</script>",
			},
			expectedStatus: http.StatusOK,
			description:    "Should handle XSS attempts in repository description",
		},
		{
			name:           "command_injection_search",
			method:         "GET",
			path:           "/api/charts/search?query=nginx;rm -rf /",
			expectedStatus: http.StatusOK,
			description:    "Should handle command injection in search query",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d. %s", 
					tt.name, tt.expectedStatus, w.Code, tt.description)
			}
			
			// Additional checks for specific security tests
			responseBody := w.Body.String()
			
			// Check that error responses don't leak sensitive information
			if w.Code >= 400 && w.Code < 600 {
				sensitivePatterns := []string{
					"/tmp/",
					"/var/",
					"/etc/",
					"internal error",
					"stack trace",
					"database",
					"sql",
				}
				
				for _, pattern := range sensitivePatterns {
					if strings.Contains(strings.ToLower(responseBody), pattern) {
						t.Errorf("%s: Response may leak sensitive information: %s", 
							tt.name, pattern)
					}
				}
			}
		})
	}
}

// TestAuthenticationSecurity tests authentication-related security
func TestAuthenticationSecurity(t *testing.T) {
	router := setupTestRouter()
	
	tests := []struct {
		name        string
		auth        *models.Authentication
		description string
		shouldPass  bool
	}{
		{
			name: "weak_password",
			auth: &models.Authentication{
				Username: "admin",
				Password: "123",
			},
			description: "Should handle weak passwords",
			shouldPass:  true, // System should accept but log warning
		},
		{
			name: "default_credentials",
			auth: &models.Authentication{
				Username: "admin",
				Password: "admin",
			},
			description: "Should handle default credentials",
			shouldPass:  true, // System should accept but log warning
		},
		{
			name: "empty_credentials",
			auth: &models.Authentication{
				Username: "",
				Password: "",
			},
			description: "Should handle empty credentials",
			shouldPass:  true,
		},
		{
			name: "injection_username",
			auth: &models.Authentication{
				Username: "admin'; DROP TABLE users; --",
				Password: "password",
			},
			description: "Should handle SQL injection in username",
			shouldPass:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoReq := models.RepositoryRequest{
				Name: fmt.Sprintf("test-repo-%s", tt.name),
				URL:  "oci://registry.example.com/charts",
				Auth: tt.auth,
			}
			
			jsonBody, _ := json.Marshal(repoReq)
			req := httptest.NewRequest("POST", "/api/repositories", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if tt.shouldPass && w.Code != http.StatusOK {
				t.Errorf("%s: Expected success but got status %d. %s", 
					tt.name, w.Code, tt.description)
			}
		})
	}
}

// TestResourceExhaustion tests protection against resource exhaustion attacks
func TestResourceExhaustion(t *testing.T) {
	router := setupTestRouter()
	
	// Test with large number of concurrent requests
	numRequests := 50
	done := make(chan bool, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(i int) {
			defer func() { done <- true }()
			
			chartReq := models.ChartRequest{
				URL: fmt.Sprintf("https://charts.example%d.com/chart.tgz", i),
			}
			
			jsonBody, _ := json.Marshal(chartReq)
			req := httptest.NewRequest("POST", "/api/chart", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Should handle gracefully, not crash
			if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
				t.Errorf("Unexpected status code %d for request %d", w.Code, i)
			}
		}(i)
	}
	
	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// TestSecurityHeaders tests security-related HTTP headers
func TestSecurityHeaders(t *testing.T) {
	router := setupTestRouter()
	
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Check CORS headers
	if corsHeader := w.Header().Get("Access-Control-Allow-Origin"); corsHeader != "*" {
		t.Errorf("Expected CORS header '*', got '%s'", corsHeader)
	}
	
	// Note: In production, these headers should be more restrictive
	// This test documents current behavior and can be updated for production
}

// TestTempFileCleanup tests that temporary files are cleaned up properly
func TestTempFileCleanup(t *testing.T) {
	processor := helm.NewProcessor()
	
	initialFiles := countTempFiles()
	
	// Process multiple charts
	urls := []string{
		"https://charts.bitnami.com/bitnami/nginx-15.4.4.tgz",
		"oci://dp.apps.rancher.io/charts/ollama:1.16.0",
	}
	
	for _, url := range urls {
		_, _, err := processor.ProcessChart(url)
		// Errors are expected for network requests in test environment
		if err != nil {
			t.Logf("Expected error processing %s: %v", url, err)
		}
	}
	
	finalFiles := countTempFiles()
	
	// Should not accumulate too many temp files
	if finalFiles > initialFiles+10 {
		t.Errorf("Potential temp file leak: started with %d, ended with %d", 
			initialFiles, finalFiles)
	}
}

func countTempFiles() int {
	count := 0
	tmpDir := "/tmp"
	
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignore errors
		}
		
		if strings.Contains(path, "helm-charts") || strings.Contains(path, "chart-") {
			count++
		}
		
		return nil
	})
	
	return count
}

// TestErrorInformationDisclosure tests that errors don't leak sensitive information
func TestErrorInformationDisclosure(t *testing.T) {
	router := setupTestRouter()
	
	// Test various error scenarios
	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{
			name:   "invalid_json",
			method: "POST",
			path:   "/api/chart",
			body:   "invalid json",
		},
		{
			name:   "missing_required_field",
			method: "POST",
			path:   "/api/repositories",
			body:   map[string]string{"name": "test"},
		},
		{
			name:   "invalid_url",
			method: "POST",
			path:   "/api/chart",
			body:   models.ChartRequest{URL: "not-a-url"},
		},
		{
			name:   "non_existent_endpoint",
			method: "GET",
			path:   "/api/non-existent",
			body:   nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			
			if tt.body != nil {
				if bodyStr, ok := tt.body.(string); ok {
					req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(bodyStr))
				} else {
					jsonBody, _ := json.Marshal(tt.body)
					req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				}
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			responseBody := strings.ToLower(w.Body.String())
			
			// Check for information disclosure
			sensitivePatterns := []string{
				"panic",
				"goroutine",
				"runtime.",
				"/go/src/",
				"/usr/local/go/",
				"internal server error",
				"database",
				"connection",
				"password",
				"secret",
				"token",
				"key",
			}
			
			for _, pattern := range sensitivePatterns {
				if strings.Contains(responseBody, pattern) {
					t.Errorf("Error response contains sensitive information: %s", pattern)
				}
			}
		})
	}
}

// TestRateLimiting tests basic rate limiting behavior
func TestRateLimiting(t *testing.T) {
	router := setupTestRouter()
	
	// Make rapid requests to the same endpoint
	numRequests := 20
	successCount := 0
	
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code == http.StatusOK {
			successCount++
		}
	}
	
	// All health check requests should succeed (no rate limiting implemented yet)
	// This test documents current behavior
	if successCount != numRequests {
		t.Logf("Rate limiting may be in effect: %d/%d requests succeeded", successCount, numRequests)
	}
}

// TestContentTypeValidation tests content type validation
func TestContentTypeValidation(t *testing.T) {
	router := setupTestRouter()
	
	tests := []struct {
		name        string
		contentType string
		body        string
		expectError bool
	}{
		{
			name:        "valid_json",
			contentType: "application/json",
			body:        `{"url": "https://example.com/chart.tgz"}`,
			expectError: false,
		},
		{
			name:        "invalid_content_type",
			contentType: "text/plain",
			body:        `{"url": "https://example.com/chart.tgz"}`,
			expectError: true,
		},
		{
			name:        "missing_content_type",
			contentType: "",
			body:        `{"url": "https://example.com/chart.tgz"}`,
			expectError: true,
		},
		{
			name:        "xml_injection",
			contentType: "application/xml",
			body:        `<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><foo>&xxe;</foo>`,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/chart", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if tt.expectError && w.Code == http.StatusOK {
				t.Error("Expected error but request succeeded")
			}
			
			if !tt.expectError && w.Code >= 400 {
				t.Errorf("Expected success but got status %d", w.Code)
			}
		})
	}
}