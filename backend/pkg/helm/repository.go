package helm

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"rancher-questions-generator/internal/models"
)

type RepositoryManager struct {
	repositories map[string]*models.Repository
	authCache    map[string]*models.Authentication // baseURL -> auth
	helmHome     string
	mutex        sync.RWMutex
}

func NewRepositoryManager() *RepositoryManager {
	helmHome := "/tmp/helm-home"
	os.MkdirAll(helmHome, 0755)
	
	rm := &RepositoryManager{
		repositories: make(map[string]*models.Repository),
		authCache:    make(map[string]*models.Authentication),
		helmHome:     helmHome,
	}
	
	// Initialize helm
	rm.initHelm()
	
	// Add default repositories
	rm.addDefaultRepositories()
	
	return rm
}

func (rm *RepositoryManager) addDefaultRepositories() {
	defaultRepos := []struct {
		name string
		url  string
		repoType string
	}{
		{"rancher-partner", "https://git.rancher.io/partner-charts", "http"},
		{"bitnami", "https://charts.bitnami.com/bitnami", "http"},
		{"stable", "https://charts.helm.sh/stable", "http"},
		{"ingress-nginx", "https://kubernetes.github.io/ingress-nginx", "http"},
	}
	
	for _, repo := range defaultRepos {
		rm.AddRepositoryWithAuth(repo.name, repo.url, "", repo.repoType, nil)
	}
}

func (rm *RepositoryManager) initHelm() error {
	// Set HELM_CONFIG_HOME environment variable
	os.Setenv("HELM_CONFIG_HOME", rm.helmHome)
	os.Setenv("HELM_CACHE_HOME", filepath.Join(rm.helmHome, "cache"))
	os.Setenv("HELM_DATA_HOME", filepath.Join(rm.helmHome, "data"))
	
	// Create necessary directories
	os.MkdirAll(filepath.Join(rm.helmHome, "cache"), 0755)
	os.MkdirAll(filepath.Join(rm.helmHome, "data"), 0755)
	
	return nil
}

func (rm *RepositoryManager) AddRepository(name, url string) error {
	return rm.AddRepositoryWithAuth(name, url, "", "http", nil)
}

func (rm *RepositoryManager) AddRepositoryWithAuth(name, repoURL, description, repoType string, auth *models.Authentication) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	// Determine repository type if not specified
	if repoType == "" {
		if strings.HasPrefix(repoURL, "oci://") {
			repoType = "oci"
		} else {
			repoType = "http"
		}
	}
	
	// Extract base URL for credential caching
	var baseURL string
	if auth != nil {
		baseURL = rm.extractBaseURL(repoURL)
		auth.BaseURL = baseURL
		
		// Cache authentication for reuse
		rm.authCache[baseURL] = auth
		
		// Perform helm login for OCI repositories
		if repoType == "oci" {
			if err := rm.performHelmLogin(repoURL, auth); err != nil {
				return fmt.Errorf("failed to authenticate with OCI registry: %w", err)
			}
		}
	} else if repoType == "oci" {
		// Check if we have cached credentials for this base URL
		baseURL = rm.extractBaseURL(repoURL)
		if cachedAuth, exists := rm.authCache[baseURL]; exists {
			auth = cachedAuth
		}
	}
	
	repo := &models.Repository{
		Name:        name,
		URL:         repoURL,
		Description: description,
		Type:        repoType,
		Auth:        auth,
		AddedAt:     time.Now(),
	}
	
	rm.repositories[name] = repo
	
	return nil
}

func (rm *RepositoryManager) RemoveRepository(name string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	if _, exists := rm.repositories[name]; !exists {
		return fmt.Errorf("repository %s not found", name)
	}
	
	delete(rm.repositories, name)
	
	return nil
}

func (rm *RepositoryManager) ListRepositories() []*models.Repository {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	repos := make([]*models.Repository, 0, len(rm.repositories))
	for _, repo := range rm.repositories {
		repos = append(repos, repo)
	}
	
	return repos
}

func (rm *RepositoryManager) SearchCharts(query, repository string) ([]*models.Chart, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	// Enhanced chart catalog with more realistic data including SUSE Application Collection
	exampleCharts := []*models.Chart{
		{
			Name:        "nginx",
			Version:     "15.4.4",
			Versions:    []string{"15.4.4", "15.4.3", "15.4.2", "15.4.1"},
			AppVersion:  "1.25.3",
			Description: "NGINX Open Source is a web server that can be also used as a reverse proxy, load balancer, and HTTP cache",
			Repository:  "bitnami",
			Keywords:    []string{"nginx", "http", "web", "www", "reverse proxy"},
		},
		{
			Name:        "ollama",
			Version:     "1.16.0",
			Versions:    []string{"1.16.0", "1.15.0", "1.14.0"},
			AppVersion:  "0.1.26",
			Description: "Get up and running with Llama 2, Mistral, Gemma, and other large language models",
			Repository:  "suse-application-collection",
			Keywords:    []string{"ai", "llm", "machine learning", "ollama"},
		},
		{
			Name:        "prometheus",
			Version:     "27.25.0",
			Versions:    []string{"27.25.0", "27.24.0", "27.23.0"},
			AppVersion:  "2.45.0",
			Description: "Prometheus is a monitoring system and time series database",
			Repository:  "suse-application-collection",
			Keywords:    []string{"monitoring", "metrics", "prometheus", "observability"},
		},
		{
			Name:        "mysql",
			Version:     "9.14.4",
			Versions:    []string{"9.14.4", "9.14.3", "9.14.2"},
			AppVersion:  "8.0.35",
			Description: "MySQL is a fast, reliable, scalable, and easy to use open source relational database system",
			Repository:  "bitnami",
			Keywords:    []string{"mysql", "database", "sql", "rdbms"},
		},
		{
			Name:        "postgresql",
			Version:     "13.2.24",
			Versions:    []string{"13.2.24", "13.2.23", "13.2.22"},
			AppVersion:  "16.1.0",
			Description: "PostgreSQL is an advanced object-relational database management system",
			Repository:  "bitnami",
			Keywords:    []string{"postgresql", "postgres", "database", "sql"},
		},
		{
			Name:        "n8n",
			Version:     "0.21.0",
			Versions:    []string{"0.21.0", "0.20.5", "0.20.4"},
			AppVersion:  "1.12.0",
			Description: "n8n is a workflow automation tool for technical people",
			Repository:  "rancher-partner",
			Keywords:    []string{"automation", "workflow", "integration"},
		},
		{
			Name:        "prometheus",
			Version:     "25.27.0",
			Versions:    []string{"25.27.0", "25.26.0", "25.25.0"},
			AppVersion:  "2.48.0",
			Description: "Prometheus is a monitoring system and time series database",
			Repository:  "stable",
			Keywords:    []string{"monitoring", "metrics", "prometheus"},
		},
		{
			Name:        "grafana",
			Version:     "7.0.17",
			Versions:    []string{"7.0.17", "7.0.16", "7.0.15"},
			AppVersion:  "10.2.2",
			Description: "The leading tool for querying and visualizing time series and logs",
			Repository:  "stable",
			Keywords:    []string{"grafana", "monitoring", "visualization"},
		},
	}
	
	var filteredCharts []*models.Chart
	for _, chart := range exampleCharts {
		if repository != "" && chart.Repository != repository {
			continue
		}
		if query != "" {
			queryLower := strings.ToLower(query)
			// Support wildcard matching
			if strings.Contains(query, "*") {
				// Simple wildcard support
				pattern := strings.ReplaceAll(queryLower, "*", ".*")
				matched, _ := filepath.Match(pattern, strings.ToLower(chart.Name))
				if !matched {
					continue
				}
			} else {
				// Search in name, description, and keywords
				found := false
				if strings.Contains(strings.ToLower(chart.Name), queryLower) {
					found = true
				}
				if strings.Contains(strings.ToLower(chart.Description), queryLower) {
					found = true
				}
				for _, keyword := range chart.Keywords {
					if strings.Contains(strings.ToLower(keyword), queryLower) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
		}
		filteredCharts = append(filteredCharts, chart)
	}
	
	return filteredCharts, nil
}

func (rm *RepositoryManager) PullChart(repository, chartName, version string) (string, error) {
	rm.mutex.RLock()
	repo, exists := rm.repositories[repository]
	rm.mutex.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("repository %s not found", repository)
	}
	
	// Handle OCI repositories
	if repo.Type == "oci" {
		if version == "" {
			version = "latest"
		}
		
		// For OCI registries, construct the full OCI URL
		// Example: oci://dp.apps.rancher.io/charts/ollama:1.16.0
		baseURL := strings.TrimPrefix(repo.URL, "oci://")
		chartURL := fmt.Sprintf("oci://%s/%s:%s", baseURL, chartName, version)
		
		// Ensure authentication is performed if needed
		if repo.Auth != nil {
			if err := rm.performHelmLogin(repo.URL, repo.Auth); err != nil {
				return "", fmt.Errorf("failed to authenticate with OCI registry: %w", err)
			}
		}
		
		return chartURL, nil
	}
	
	// Handle HTTP repositories
	var chartURL string
	switch repository {
	case "bitnami":
		if version == "" {
			version = "latest"
		}
		chartURL = fmt.Sprintf("https://charts.bitnami.com/bitnami/%s-%s.tgz", chartName, version)
	case "stable":
		chartURL = fmt.Sprintf("https://charts.helm.sh/stable/%s-%s.tgz", chartName, version)
	default:
		// Try to construct URL based on repository URL
		if version == "" {
			version = "latest"
		}
		chartURL = fmt.Sprintf("%s/%s-%s.tgz", strings.TrimSuffix(repo.URL, "/"), chartName, version)
	}
	
	return chartURL, nil
}

func (rm *RepositoryManager) GetRepositoryCharts(repositoryName string) ([]*models.Chart, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if _, exists := rm.repositories[repositoryName]; !exists {
		return nil, fmt.Errorf("repository %s not found", repositoryName)
	}
	
	// Return all charts for the specific repository
	return rm.SearchCharts("", repositoryName)
}

func (rm *RepositoryManager) GetStorageClasses() ([]*models.StorageClass, error) {
	// In a real implementation, this would query Kubernetes for storage classes
	// For now, return some common examples
	storageClasses := []*models.StorageClass{
		{Name: "standard", Provisioner: "kubernetes.io/gce-pd", IsDefault: true},
		{Name: "fast", Provisioner: "kubernetes.io/gce-pd-ssd", IsDefault: false},
		{Name: "slow", Provisioner: "kubernetes.io/gce-pd", IsDefault: false},
		{Name: "local-path", Provisioner: "rancher.io/local-path", IsDefault: false},
	}
	
	return storageClasses, nil
}

// Helper function to extract base URL for credential caching
func (rm *RepositoryManager) extractBaseURL(repoURL string) string {
	if strings.HasPrefix(repoURL, "oci://") {
		// For OCI URLs like oci://dp.apps.rancher.io/charts/ollama
		// Extract dp.apps.rancher.io
		cleanURL := strings.TrimPrefix(repoURL, "oci://")
		parts := strings.Split(cleanURL, "/")
		if len(parts) > 0 {
			return parts[0]
		}
	} else {
		// For HTTP URLs, extract the host
		if parsedURL, err := url.Parse(repoURL); err == nil {
			return parsedURL.Host
		}
	}
	return repoURL
}

// Perform helm registry login for OCI repositories
func (rm *RepositoryManager) performHelmLogin(registryURL string, auth *models.Authentication) error {
	if auth == nil {
		return fmt.Errorf("authentication required for OCI registry")
	}
	
	// For SUSE Application Collection, use the full path as provided in the helm login command
	// Example: helm registry login dp.apps.rancher.io/charts -u user -p pass
	loginURL := registryURL
	if strings.HasPrefix(registryURL, "oci://") {
		loginURL = strings.TrimPrefix(registryURL, "oci://")
	}
	
	args := []string{"registry", "login", loginURL}
	
	if auth.Username != "" && auth.Password != "" {
		args = append(args, "--username", auth.Username, "--password", auth.Password)
	} else if auth.SecretName != "" {
		// TODO: Implement secret retrieval from Kubernetes
		return fmt.Errorf("kubernetes secret authentication not yet implemented")
	} else {
		return fmt.Errorf("no valid authentication method provided")
	}
	
	_, err := rm.runHelmCommand(args...)
	return err
}

// Check if credentials are available for a base URL
func (rm *RepositoryManager) hasCredentialsForBaseURL(baseURL string) bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	_, exists := rm.authCache[baseURL]
	return exists
}

// Get authentication for a repository URL
func (rm *RepositoryManager) getAuthForURL(repoURL string) *models.Authentication {
	baseURL := rm.extractBaseURL(repoURL)
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if auth, exists := rm.authCache[baseURL]; exists {
		return auth
	}
	return nil
}

// Helper function to run helm commands (for future use when helm is installed)
func (rm *RepositoryManager) runHelmCommand(args ...string) ([]byte, error) {
	cmd := exec.Command("helm", args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HELM_CONFIG_HOME=%s", rm.helmHome),
		fmt.Sprintf("HELM_CACHE_HOME=%s", filepath.Join(rm.helmHome, "cache")),
		fmt.Sprintf("HELM_DATA_HOME=%s", filepath.Join(rm.helmHome, "data")),
	)
	
	return cmd.CombinedOutput()
}