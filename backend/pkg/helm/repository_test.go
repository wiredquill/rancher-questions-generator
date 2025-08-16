package helm

import (
	"strings"
	"testing"
	"time"

	"rancher-questions-generator/internal/models"
)

func TestNewRepositoryManager(t *testing.T) {
	rm := NewRepositoryManager()
	if rm == nil {
		t.Fatal("NewRepositoryManager() returned nil")
	}
	
	if rm.repositories == nil {
		t.Error("repositories map not initialized")
	}
	
	if rm.authCache == nil {
		t.Error("authCache map not initialized")
	}
	
	if rm.helmHome == "" {
		t.Error("helmHome not set")
	}
	
	// Check that default repositories were added
	repos := rm.ListRepositories()
	if len(repos) == 0 {
		t.Error("Expected default repositories to be added, got 0")
	}
	
	// Verify common default repositories
	expectedRepos := []string{"rancher-partner", "bitnami", "stable", "ingress-nginx", "suse-application-collection"}
	repoMap := make(map[string]bool)
	for _, repo := range repos {
		repoMap[repo.Name] = true
	}
	
	for _, expected := range expectedRepos {
		if !repoMap[expected] {
			t.Errorf("Expected default repository '%s' not found", expected)
		}
	}
}

func TestAddRepository(t *testing.T) {
	rm := NewRepositoryManager()
	
	// Clear default repositories for clean testing
	rm.repositories = make(map[string]*models.Repository)
	
	tests := []struct {
		name     string
		repoName string
		repoURL  string
		wantErr  bool
	}{
		{
			name:     "add http repository",
			repoName: "test-repo",
			repoURL:  "https://charts.example.com",
			wantErr:  false,
		},
		{
			name:     "add oci repository",
			repoName: "oci-repo",
			repoURL:  "oci://registry.example.com/charts",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.AddRepository(tt.repoName, tt.repoURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if !tt.wantErr {
				repos := rm.ListRepositories()
				found := false
				for _, repo := range repos {
					if repo.Name == tt.repoName && repo.URL == tt.repoURL {
						found = true
						
						// Verify type detection
						expectedType := "http"
						if strings.HasPrefix(tt.repoURL, "oci://") {
							expectedType = "oci"
						}
						if repo.Type != expectedType {
							t.Errorf("Expected repository type %s, got %s", expectedType, repo.Type)
						}
						break
					}
				}
				if !found {
					t.Errorf("Repository %s was not added", tt.repoName)
				}
			}
		})
	}
}

func TestAddRepositoryWithAuth(t *testing.T) {
	rm := NewRepositoryManager()
	rm.repositories = make(map[string]*models.Repository) // Clear defaults
	
	auth := &models.Authentication{
		Username: "testuser",
		Password: "testpass",
	}
	
	err := rm.AddRepositoryWithAuth("test-repo", "oci://registry.example.com/charts", "Test Repo", "oci", auth)
	if err != nil {
		t.Errorf("AddRepositoryWithAuth() failed: %v", err)
	}
	
	repos := rm.ListRepositories()
	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}
	
	repo := repos[0]
	if repo.Auth == nil {
		t.Error("Authentication not stored")
	}
	
	if repo.Auth.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", repo.Auth.Username)
	}
	
	// Check auth cache
	baseURL := rm.extractBaseURL("oci://registry.example.com/charts")
	if cachedAuth, exists := rm.authCache[baseURL]; !exists {
		t.Error("Authentication not cached")
	} else if cachedAuth.Username != "testuser" {
		t.Errorf("Cached auth username mismatch: expected 'testuser', got '%s'", cachedAuth.Username)
	}
}

func TestRemoveRepository(t *testing.T) {
	rm := NewRepositoryManager()
	rm.repositories = make(map[string]*models.Repository) // Clear defaults
	
	// Add a repository first
	rm.AddRepository("test-repo", "https://charts.example.com")
	
	// Verify it exists
	repos := rm.ListRepositories()
	if len(repos) != 1 {
		t.Errorf("Expected 1 repository before removal, got %d", len(repos))
	}
	
	// Remove it
	err := rm.RemoveRepository("test-repo")
	if err != nil {
		t.Errorf("RemoveRepository() failed: %v", err)
	}
	
	// Verify it's gone
	repos = rm.ListRepositories()
	if len(repos) != 0 {
		t.Errorf("Expected 0 repositories after removal, got %d", len(repos))
	}
	
	// Try to remove non-existent repository
	err = rm.RemoveRepository("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent repository")
	}
}

func TestSearchCharts(t *testing.T) {
	rm := NewRepositoryManager()
	
	tests := []struct {
		name       string
		query      string
		repository string
		minResults int // minimum expected results
	}{
		{
			name:       "search all charts",
			query:      "",
			repository: "",
			minResults: 1,
		},
		{
			name:       "search nginx",
			query:      "nginx",
			repository: "",
			minResults: 1,
		},
		{
			name:       "search bitnami charts",
			query:      "",
			repository: "bitnami",
			minResults: 1,
		},
		{
			name:       "wildcard search",
			query:      "*sql",
			repository: "",
			minResults: 1,
		},
		{
			name:       "no results",
			query:      "nonexistentchart12345",
			repository: "",
			minResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charts, err := rm.SearchCharts(tt.query, tt.repository)
			if err != nil {
				t.Errorf("SearchCharts() failed: %v", err)
			}
			
			if len(charts) < tt.minResults {
				t.Errorf("Expected at least %d results, got %d", tt.minResults, len(charts))
			}
			
			// Verify chart structure
			for _, chart := range charts {
				if chart.Name == "" {
					t.Error("Chart name is empty")
				}
				if chart.Version == "" {
					t.Error("Chart version is empty")
				}
				if chart.Repository == "" {
					t.Error("Chart repository is empty")
				}
			}
		})
	}
}

func TestFilterCharts(t *testing.T) {
	rm := NewRepositoryManager()
	
	testCharts := []*models.Chart{
		{
			Name:        "nginx",
			Repository:  "bitnami",
			Description: "NGINX web server",
			Keywords:    []string{"web", "http"},
		},
		{
			Name:        "mysql",
			Repository:  "bitnami",
			Description: "MySQL database",
			Keywords:    []string{"database", "sql"},
		},
		{
			Name:        "prometheus",
			Repository:  "stable",
			Description: "Monitoring system",
			Keywords:    []string{"monitoring", "metrics"},
		},
	}
	
	tests := []struct {
		name       string
		query      string
		repository string
		expected   int
	}{
		{
			name:       "no filter",
			query:      "",
			repository: "",
			expected:   3,
		},
		{
			name:       "filter by repository",
			query:      "",
			repository: "bitnami",
			expected:   2,
		},
		{
			name:       "filter by name",
			query:      "nginx",
			repository: "",
			expected:   1,
		},
		{
			name:       "filter by keyword",
			query:      "monitoring",
			repository: "",
			expected:   1,
		},
		{
			name:       "wildcard search",
			query:      "*sql",
			repository: "",
			expected:   1,
		},
		{
			name:       "combined filter",
			query:      "database",
			repository: "bitnami",
			expected:   1,
		},
		{
			name:       "no matches",
			query:      "nonexistent",
			repository: "",
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := rm.filterCharts(testCharts, tt.query, tt.repository)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestPullChart(t *testing.T) {
	rm := NewRepositoryManager()
	rm.repositories = make(map[string]*models.Repository) // Clear defaults
	
	// Add test repositories
	rm.AddRepository("bitnami", "https://charts.bitnami.com/bitnami")
	rm.AddRepositoryWithAuth("oci-repo", "oci://registry.example.com/charts", "", "oci", &models.Authentication{
		Username: "user",
		Password: "pass",
	})
	
	tests := []struct {
		name       string
		repository string
		chartName  string
		version    string
		wantErr    bool
		urlPattern string
	}{
		{
			name:       "bitnami chart",
			repository: "bitnami",
			chartName:  "nginx",
			version:    "1.0.0",
			wantErr:    false,
			urlPattern: "https://charts.bitnami.com/bitnami/nginx-1.0.0.tgz",
		},
		{
			name:       "oci chart",
			repository: "oci-repo",
			chartName:  "test-chart",
			version:    "2.0.0",
			wantErr:    false,
			urlPattern: "oci://registry.example.com/charts/test-chart:2.0.0",
		},
		{
			name:       "non-existent repository",
			repository: "non-existent",
			chartName:  "chart",
			version:    "1.0.0",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chartURL, err := rm.PullChart(tt.repository, tt.chartName, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("PullChart() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if !tt.wantErr && !strings.Contains(chartURL, tt.urlPattern) {
				t.Errorf("Expected URL to contain '%s', got '%s'", tt.urlPattern, chartURL)
			}
		})
	}
}

func TestExtractBaseURL(t *testing.T) {
	rm := NewRepositoryManager()
	
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "oci url",
			repoURL:  "oci://dp.apps.rancher.io/charts/ollama",
			expected: "dp.apps.rancher.io",
		},
		{
			name:     "http url",
			repoURL:  "https://charts.bitnami.com/bitnami",
			expected: "charts.bitnami.com",
		},
		{
			name:     "simple oci",
			repoURL:  "oci://registry.io/path",
			expected: "registry.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rm.extractBaseURL(tt.repoURL)
			if result != tt.expected {
				t.Errorf("extractBaseURL(%s) = %s, expected %s", tt.repoURL, result, tt.expected)
			}
		})
	}
}

func TestGetStorageClasses(t *testing.T) {
	rm := NewRepositoryManager()
	
	storageClasses, err := rm.GetStorageClasses()
	if err != nil {
		t.Errorf("GetStorageClasses() failed: %v", err)
	}
	
	if len(storageClasses) == 0 {
		t.Error("Expected at least one storage class")
	}
	
	// Check for default storage class
	hasDefault := false
	for _, sc := range storageClasses {
		if sc.Name == "" {
			t.Error("Storage class name is empty")
		}
		if sc.Provisioner == "" {
			t.Error("Storage class provisioner is empty")
		}
		if sc.IsDefault {
			hasDefault = true
		}
	}
	
	if !hasDefault {
		t.Error("Expected at least one default storage class")
	}
}

func TestConcurrentAccess(t *testing.T) {
	rm := NewRepositoryManager()
	rm.repositories = make(map[string]*models.Repository) // Clear defaults
	
	// Test concurrent repository operations
	done := make(chan bool)
	
	// Concurrent adds
	for i := 0; i < 10; i++ {
		go func(i int) {
			repoName := fmt.Sprintf("repo-%d", i)
			repoURL := fmt.Sprintf("https://charts.example%d.com", i)
			rm.AddRepository(repoName, repoURL)
			done <- true
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			rm.ListRepositories()
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
	
	// Verify final state
	repos := rm.ListRepositories()
	if len(repos) != 10 {
		t.Errorf("Expected 10 repositories after concurrent operations, got %d", len(repos))
	}
}

// Benchmark tests
func BenchmarkSearchCharts(b *testing.B) {
	rm := NewRepositoryManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.SearchCharts("nginx", "")
	}
}

func BenchmarkAddRepository(b *testing.B) {
	rm := NewRepositoryManager()
	rm.repositories = make(map[string]*models.Repository) // Clear defaults
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repoName := fmt.Sprintf("repo-%d", i)
		repoURL := fmt.Sprintf("https://charts.example%d.com", i)
		rm.AddRepository(repoName, repoURL)
	}
}

func BenchmarkListRepositories(b *testing.B) {
	rm := NewRepositoryManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.ListRepositories()
	}
}