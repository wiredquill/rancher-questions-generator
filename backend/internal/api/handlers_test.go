package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rancher-questions-generator/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return SetupRouter()
}

func TestHealthCheck(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestProcessChart(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid chart request",
			requestBody: models.ChartRequest{
				URL: "https://charts.bitnami.com/bitnami/nginx-15.4.4.tgz",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request body",
			requestBody:    map[string]interface{}{"invalid": "data"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty request",
			requestBody:    models.ChartRequest{URL: ""},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/chart", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response models.ChartResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.SessionID)
				assert.NotNil(t, response.Values)
				assert.NotNil(t, response.Questions)
			}
		})
	}
}

func TestAddRepository(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid http repository",
			requestBody: models.RepositoryRequest{
				Name: "test-repo",
				URL:  "https://charts.example.com",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid oci repository",
			requestBody: models.RepositoryRequest{
				Name: "oci-repo",
				URL:  "oci://registry.example.com/charts",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "repository with auth",
			requestBody: models.RepositoryRequest{
				Name: "auth-repo",
				URL:  "oci://private.registry.com/charts",
				Auth: &models.Authentication{
					Username: "testuser",
					Password: "testpass",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing name",
			requestBody:    models.RepositoryRequest{URL: "https://charts.example.com"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url",
			requestBody:    models.RepositoryRequest{Name: "test"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/repositories", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Repository added successfully", response["message"])
			}
		})
	}
}

func TestListRepositories(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/repositories", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	repositories, exists := response["repositories"]
	assert.True(t, exists)
	assert.NotNil(t, repositories)
	
	// Should have default repositories
	repoList, ok := repositories.([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(repoList), 0)
}

func TestSearchCharts(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "GET search all charts",
			method:         "GET",
			path:           "/api/charts/search",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET search with query",
			method:         "GET",
			path:           "/api/charts/search?query=nginx",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET search with repository filter",
			method:         "GET",
			path:           "/api/charts/search?repository=bitnami",
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST search",
			method: "POST",
			path:   "/api/charts/search",
			body: models.ChartSearchRequest{
				Query:      "nginx",
				Repository: "bitnami",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" && tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				charts, exists := response["charts"]
				assert.True(t, exists)
				assert.NotNil(t, charts)
			}
		})
	}
}

func TestProcessChartFromRepository(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid chart process request",
			requestBody: models.ChartProcessRequest{
				Repository: "bitnami",
				Chart:      "nginx",
				Version:    "15.4.4",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing repository",
			requestBody:    models.ChartProcessRequest{Chart: "nginx"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing chart",
			requestBody:    models.ChartProcessRequest{Repository: "bitnami"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent repository",
			requestBody: models.ChartProcessRequest{
				Repository: "non-existent",
				Chart:      "nginx",
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/charts/process", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response models.ChartResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.SessionID)
			}
		})
	}
}

func TestGetStorageClasses(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/storage-classes", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	storageClasses, exists := response["storage_classes"]
	assert.True(t, exists)
	assert.NotNil(t, storageClasses)
	
	// Should have at least one storage class
	scList, ok := storageClasses.([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(scList), 0)
}

func TestRemoveRepository(t *testing.T) {
	router := setupRouter()
	
	// First add a repository
	addReq := models.RepositoryRequest{
		Name: "test-remove",
		URL:  "https://charts.example.com",
	}
	jsonBody, _ := json.Marshal(addReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/repositories", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	tests := []struct {
		name           string
		repoName       string
		expectedStatus int
	}{
		{
			name:           "remove existing repository",
			repoName:       "test-remove",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "remove non-existent repository",
			repoName:       "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/repositories/"+tt.repoName, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Repository removed successfully", response["message"])
			}
		})
	}
}

func TestGetRepositoryCharts(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		repository     string
		expectedStatus int
	}{
		{
			name:           "get charts from existing repository",
			repository:     "bitnami",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get charts from non-existent repository",
			repository:     "non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/repositories/"+tt.repository+"/charts", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				
				charts, exists := response["charts"]
				assert.True(t, exists)
				assert.NotNil(t, charts)
			}
		})
	}
}

func TestCORSHeaders(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
}

func TestSessionManagement(t *testing.T) {
	router := setupRouter()

	// Create a chart session
	chartReq := models.ChartRequest{
		URL: "https://charts.bitnami.com/bitnami/nginx-15.4.4.tgz",
	}
	jsonBody, _ := json.Marshal(chartReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/chart", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var createResponse models.ChartResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.NoError(t, err)
	sessionID := createResponse.SessionID

	// Test GET session
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/chart/"+sessionID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var getResponse models.ChartResponse
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	assert.NoError(t, err)
	assert.Equal(t, sessionID, getResponse.SessionID)

	// Test UPDATE session
	updateQuestions := models.Questions{
		Questions: []models.Question{
			{
				Variable: "test.variable",
				Label:    "Test Variable",
				Type:     "string",
			},
		},
	}
	jsonBody, _ = json.Marshal(updateQuestions)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/chart/"+sessionID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test GET YAML
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/chart/"+sessionID+"/q", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-yaml", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "questions.yaml")

	// Test non-existent session
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/chart/non-existent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Integration test for full workflow
func TestFullWorkflow(t *testing.T) {
	router := setupRouter()

	// 1. Add a custom repository
	addRepoReq := models.RepositoryRequest{
		Name: "workflow-test",
		URL:  "https://charts.workflow.com",
	}
	jsonBody, _ := json.Marshal(addRepoReq)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/repositories", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. List repositories (should include our new one)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/repositories", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 3. Search for charts
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/charts/search?query=nginx", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Process a chart from repository
	processReq := models.ChartProcessRequest{
		Repository: "bitnami",
		Chart:      "nginx",
		Version:    "15.4.4",
	}
	jsonBody, _ = json.Marshal(processReq)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/charts/process", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var processResponse models.ChartResponse
	err := json.Unmarshal(w.Body.Bytes(), &processResponse)
	assert.NoError(t, err)
	sessionID := processResponse.SessionID

	// 5. Download YAML
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/chart/"+sessionID+"/q", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. Remove the repository
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/repositories/workflow-test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}