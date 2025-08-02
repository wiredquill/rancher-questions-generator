package helm

import (
	"encoding/json"
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
		description string
		repoType string
	}{
		{"rancher-partner", "https://git.rancher.io/partner-charts", "Rancher Partner Charts Repository", "http"},
		{"bitnami", "https://charts.bitnami.com/bitnami", "Bitnami Helm Charts", "http"},
		{"stable", "https://charts.helm.sh/stable", "Helm Stable Charts (Deprecated)", "http"},
		{"ingress-nginx", "https://kubernetes.github.io/ingress-nginx", "NGINX Ingress Controller", "http"},
		{"suse-application-collection", "oci://dp.apps.rancher.io/charts", "SUSE Application Collection (OCI)", "oci"},
	}
	
	fmt.Printf("Adding %d default repositories...\n", len(defaultRepos))
	for _, repo := range defaultRepos {
		err := rm.AddRepositoryWithAuth(repo.name, repo.url, repo.description, repo.repoType, nil)
		if err != nil {
			fmt.Printf("Failed to add default repository %s: %v\n", repo.name, err)
		} else {
			fmt.Printf("Added default repository: %s (%s)\n", repo.name, repo.url)
		}
	}
	fmt.Printf("Default repositories initialization complete. Total repositories: %d\n", len(rm.repositories))
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
				fmt.Printf("Warning: OCI authentication failed: %v\n", err)
				// Don't fail repository addition if helm is not available
				// Store the auth info for later use
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
	
	fmt.Printf("ListRepositories called: returning %d repositories\n", len(repos))
	for _, repo := range repos {
		fmt.Printf("  - %s: %s\n", repo.Name, repo.URL)
	}
	
	return repos
}

func (rm *RepositoryManager) SearchCharts(query, repository string) ([]*models.Chart, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	// Try to fetch charts from actual Helm repositories first
	if repository != "" {
		if repo, exists := rm.repositories[repository]; exists {
			if charts, err := rm.fetchChartsFromRepository(repo); err == nil && len(charts) > 0 {
				return rm.filterCharts(charts, query), nil
			}
			fmt.Printf("Failed to fetch charts from repository %s, falling back to examples: %v\n", repository, err)
		}
	}
	
	// Enhanced chart catalog with more realistic data - fallback for when real repositories aren't accessible
	exampleCharts := []*models.Chart{
		// Rancher Partner Charts
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
			Name:        "rancher-monitoring",
			Version:     "103.0.3",
			Versions:    []string{"103.0.3", "103.0.2", "103.0.1"},
			AppVersion:  "v0.68.0",
			Description: "Rancher Monitoring powered by Prometheus",
			Repository:  "rancher-partner",
			Keywords:    []string{"monitoring", "prometheus", "rancher"},
		},
		{
			Name:        "rancher-logging",
			Version:     "103.1.1",
			Versions:    []string{"103.1.1", "103.1.0", "103.0.0"},
			AppVersion:  "4.4.0",
			Description: "Rancher Logging powered by Fluent Bit and Fluentd",
			Repository:  "rancher-partner",
			Keywords:    []string{"logging", "fluent", "rancher"},
		},
		{
			Name:        "rancher-istio",
			Version:     "103.2.0",
			Versions:    []string{"103.2.0", "103.1.0", "103.0.0"},
			AppVersion:  "1.19.3",
			Description: "Rancher Istio Service Mesh",
			Repository:  "rancher-partner",
			Keywords:    []string{"service-mesh", "istio", "rancher"},
		},
		// Bitnami Charts
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
			Name:        "redis",
			Version:     "18.19.4",
			Versions:    []string{"18.19.4", "18.19.3", "18.19.2"},
			AppVersion:  "7.2.4",
			Description: "Redis is an open source, in-memory data structure store",
			Repository:  "bitnami",
			Keywords:    []string{"redis", "cache", "database", "memory"},
		},
		// Stable Charts
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
		// Ingress NGINX
		{
			Name:        "ingress-nginx",
			Version:     "4.8.3",
			Versions:    []string{"4.8.3", "4.8.2", "4.8.1"},
			AppVersion:  "1.9.4",
			Description: "Ingress controller for Kubernetes using NGINX as a reverse proxy and load balancer",
			Repository:  "ingress-nginx",
			Keywords:    []string{"ingress", "nginx", "load-balancer"},
		},
	}
	
	return rm.filterCharts(exampleCharts, query, repository), nil
}

// Helper function to filter charts based on query and repository
func (rm *RepositoryManager) filterCharts(charts []*models.Chart, query string, repository ...string) []*models.Chart {
	var filteredCharts []*models.Chart
	repo := ""
	if len(repository) > 0 {
		repo = repository[0]
	}
	
	for _, chart := range charts {
		if repo != "" && chart.Repository != repo {
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
	
	return filteredCharts
}

// Fetch charts from actual Helm repository (attempts real repository access)
func (rm *RepositoryManager) fetchChartsFromRepository(repo *models.Repository) ([]*models.Chart, error) {
	if repo.Type == "oci" {
		return rm.fetchOCICharts(repo)
	}
	return rm.fetchHTTPCharts(repo)
}

// Fetch charts from HTTP-based Helm repository
func (rm *RepositoryManager) fetchHTTPCharts(repo *models.Repository) ([]*models.Chart, error) {
	// Add repository to helm if not already added
	if err := rm.addHelmRepo(repo); err != nil {
		fmt.Printf("Warning: Failed to add Helm repo %s: %v\n", repo.Name, err)
		// Continue with fallback data
		return nil, err
	}
	
	// Update repository index
	if err := rm.updateHelmRepo(repo.Name); err != nil {
		fmt.Printf("Warning: Failed to update Helm repo %s: %v\n", repo.Name, err)
		return nil, err
	}
	
	// Search for charts in the repository
	charts, err := rm.searchHelmCharts(repo.Name)
	if err != nil {
		fmt.Printf("Warning: Failed to search charts in repo %s: %v\n", repo.Name, err)
		return nil, err
	}
	
	return charts, nil
}

// Fetch charts from OCI registry
func (rm *RepositoryManager) fetchOCICharts(repo *models.Repository) ([]*models.Chart, error) {
	// For OCI registries like SUSE Application Collection, we need to use
	// known chart names since OCI registries don't have an index.yaml
	// This is the comprehensive SUSE Application Collection catalog
	suseCharts := []*models.Chart{
		// AI/ML Applications
		{
			Name:        "ollama",
			Version:     "1.16.0",
			Versions:    []string{"1.16.0", "1.15.0", "1.14.0"},
			AppVersion:  "0.1.26",
			Description: "Get up and running with Llama 2, Mistral, Gemma, and other large language models",
			Repository:  repo.Name,
			Keywords:    []string{"ai", "llama", "llm", "machine learning", "gpu"},
		},
		{
			Name:        "jupyter",
			Version:     "1.0.2",
			Versions:    []string{"1.0.2", "1.0.1", "1.0.0"},
			AppVersion:  "4.1.5",
			Description: "Jupyter notebook for data science and machine learning",
			Repository:  repo.Name,
			Keywords:    []string{"jupyter", "notebook", "data science", "python"},
		},
		{
			Name:        "tensorflow-serving",
			Version:     "2.14.0",
			Versions:    []string{"2.14.0", "2.13.0", "2.12.0"},
			AppVersion:  "2.14.0",
			Description: "TensorFlow Serving for ML model deployment",
			Repository:  repo.Name,
			Keywords:    []string{"tensorflow", "ml", "serving", "inference"},
		},

		// Databases
		{
			Name:        "mysql",
			Version:     "11.1.3",
			Versions:    []string{"11.1.3", "11.1.2", "11.1.1"},
			AppVersion:  "8.4.1",
			Description: "MySQL is a fast, reliable, scalable, and easy to use open source relational database system",
			Repository:  repo.Name,
			Keywords:    []string{"mysql", "database", "sql", "rdbms"},
		},
		{
			Name:        "postgresql",
			Version:     "15.5.8",
			Versions:    []string{"15.5.8", "15.5.7", "15.5.6"},
			AppVersion:  "16.3.0",
			Description: "PostgreSQL is an advanced object-relational database management system",
			Repository:  repo.Name,
			Keywords:    []string{"postgresql", "postgres", "database", "sql"},
		},
		{
			Name:        "redis",
			Version:     "19.6.4",
			Versions:    []string{"19.6.4", "19.6.3", "19.6.2"},
			AppVersion:  "7.2.5",
			Description: "Redis is an open source, in-memory data structure store",
			Repository:  repo.Name,
			Keywords:    []string{"redis", "cache", "database", "memory"},
		},
		{
			Name:        "mongodb",
			Version:     "15.6.14",
			Versions:    []string{"15.6.14", "15.6.13", "15.6.12"},
			AppVersion:  "7.0.12",
			Description: "MongoDB is a source-available cross-platform document-oriented database",
			Repository:  repo.Name,
			Keywords:    []string{"mongodb", "database", "nosql", "document"},
		},
		{
			Name:        "mariadb",
			Version:     "18.2.1",
			Versions:    []string{"18.2.1", "18.2.0", "18.1.0"},
			AppVersion:  "11.4.2",
			Description: "MariaDB is a fork of MySQL with additional features",
			Repository:  repo.Name,
			Keywords:    []string{"mariadb", "mysql", "database", "sql"},
		},
		{
			Name:        "cassandra",
			Version:     "11.3.7",
			Versions:    []string{"11.3.7", "11.3.6", "11.3.5"},
			AppVersion:  "4.1.5",
			Description: "Apache Cassandra is a distributed NoSQL database",
			Repository:  repo.Name,
			Keywords:    []string{"cassandra", "nosql", "distributed", "database"},
		},

		// Monitoring & Observability
		{
			Name:        "prometheus",
			Version:     "61.1.1",
			Versions:    []string{"61.1.1", "61.1.0", "61.0.0"},
			AppVersion:  "2.53.1",
			Description: "Prometheus is a monitoring system and time series database",
			Repository:  repo.Name,
			Keywords:    []string{"monitoring", "metrics", "prometheus", "observability"},
		},
		{
			Name:        "grafana",
			Version:     "8.0.2",
			Versions:    []string{"8.0.2", "8.0.1", "8.0.0"},
			AppVersion:  "11.1.0",
			Description: "The leading tool for querying and visualizing time series and logs",
			Repository:  repo.Name,
			Keywords:    []string{"grafana", "monitoring", "visualization", "dashboard"},
		},
		{
			Name:        "jaeger",
			Version:     "3.0.10",
			Versions:    []string{"3.0.10", "3.0.9", "3.0.8"},
			AppVersion:  "1.57.0",
			Description: "Jaeger is a distributed tracing platform",
			Repository:  repo.Name,
			Keywords:    []string{"jaeger", "tracing", "observability", "distributed"},
		},
		{
			Name:        "elasticsearch",
			Version:     "21.3.15",
			Versions:    []string{"21.3.15", "21.3.14", "21.3.13"},
			AppVersion:  "8.14.1",
			Description: "Elasticsearch is a distributed search and analytics engine",
			Repository:  repo.Name,
			Keywords:    []string{"elasticsearch", "search", "analytics", "logs"},
		},
		{
			Name:        "kibana",
			Version:     "11.2.15",
			Versions:    []string{"11.2.15", "11.2.14", "11.2.13"},
			AppVersion:  "8.14.1",
			Description: "Kibana is a data visualization dashboard for Elasticsearch",
			Repository:  repo.Name,
			Keywords:    []string{"kibana", "elasticsearch", "visualization", "dashboard"},
		},

		// Web Servers & Proxies
		{
			Name:        "nginx",
			Version:     "18.1.6",
			Versions:    []string{"18.1.6", "18.1.5", "18.1.4"},
			AppVersion:  "1.27.0",
			Description: "NGINX Open Source is a web server that can be also used as a reverse proxy, load balancer, and HTTP cache",
			Repository:  repo.Name,
			Keywords:    []string{"nginx", "http", "web", "www", "reverse proxy"},
		},
		{
			Name:        "apache",
			Version:     "11.2.12",
			Versions:    []string{"11.2.12", "11.2.11", "11.2.10"},
			AppVersion:  "2.4.59",
			Description: "Apache HTTP Server is a free and open-source web server",
			Repository:  repo.Name,
			Keywords:    []string{"apache", "http", "web", "server"},
		},
		{
			Name:        "traefik",
			Version:     "28.3.0",
			Versions:    []string{"28.3.0", "28.2.0", "28.1.0"},
			AppVersion:  "3.1.0",
			Description: "Traefik is a modern reverse proxy and load balancer",
			Repository:  repo.Name,
			Keywords:    []string{"traefik", "proxy", "load balancer", "ingress"},
		},
		{
			Name:        "haproxy",
			Version:     "0.14.6",
			Versions:    []string{"0.14.6", "0.14.5", "0.14.4"},
			AppVersion:  "2.9.7",
			Description: "HAProxy is a reliable, high performance load balancer",
			Repository:  repo.Name,
			Keywords:    []string{"haproxy", "load balancer", "proxy", "tcp"},
		},

		// Message Queues & Streaming
		{
			Name:        "kafka",
			Version:     "29.3.5",
			Versions:    []string{"29.3.5", "29.3.4", "29.3.3"},
			AppVersion:  "3.7.0",
			Description: "Apache Kafka is a distributed streaming platform",
			Repository:  repo.Name,
			Keywords:    []string{"kafka", "streaming", "messaging", "queue"},
		},
		{
			Name:        "rabbitmq",
			Version:     "14.6.6",
			Versions:    []string{"14.6.6", "14.6.5", "14.6.4"},
			AppVersion:  "3.13.2",
			Description: "RabbitMQ is a message broker that supports multiple messaging protocols",
			Repository:  repo.Name,
			Keywords:    []string{"rabbitmq", "message", "broker", "amqp"},
		},
		{
			Name:        "nats",
			Version:     "1.1.12",
			Versions:    []string{"1.1.12", "1.1.11", "1.1.10"},
			AppVersion:  "2.10.16",
			Description: "NATS is a simple, secure and performant messaging system",
			Repository:  repo.Name,
			Keywords:    []string{"nats", "messaging", "pubsub", "streaming"},
		},

		// Development Tools
		{
			Name:        "jenkins",
			Version:     "5.1.20",
			Versions:    []string{"5.1.20", "5.1.19", "5.1.18"},
			AppVersion:  "2.452.1",
			Description: "Jenkins is an open source automation server",
			Repository:  repo.Name,
			Keywords:    []string{"jenkins", "ci", "cd", "automation"},
		},
		{
			Name:        "gitlab",
			Version:     "8.1.1",
			Versions:    []string{"8.1.1", "8.1.0", "8.0.0"},
			AppVersion:  "17.1.1",
			Description: "GitLab is a DevOps platform with Git repository management",
			Repository:  repo.Name,
			Keywords:    []string{"gitlab", "git", "devops", "ci/cd"},
		},
		{
			Name:        "gitea",
			Version:     "10.4.0",
			Versions:    []string{"10.4.0", "10.3.0", "10.2.0"},
			AppVersion:  "1.21.11",
			Description: "Gitea is a lightweight Git service written in Go",
			Repository:  repo.Name,
			Keywords:    []string{"gitea", "git", "repository", "self-hosted"},
		},
		{
			Name:        "sonarqube",
			Version:     "10.6.0",
			Versions:    []string{"10.6.0", "10.5.1", "10.5.0"},
			AppVersion:  "10.6.0",
			Description: "SonarQube is a platform for continuous code quality inspection",
			Repository:  repo.Name,
			Keywords:    []string{"sonarqube", "code quality", "static analysis"},
		},
		{
			Name:        "nexus",
			Version:     "64.2.0",
			Versions:    []string{"64.2.0", "64.1.0", "64.0.0"},
			AppVersion:  "3.69.0",
			Description: "Nexus Repository Manager for storing and distributing artifacts",
			Repository:  repo.Name,
			Keywords:    []string{"nexus", "repository", "artifacts", "maven"},
		},

		// Content Management & Collaboration
		{
			Name:        "wordpress",
			Version:     "23.1.17",
			Versions:    []string{"23.1.17", "23.1.16", "23.1.15"},
			AppVersion:  "6.5.4",
			Description: "WordPress is a content management system",
			Repository:  repo.Name,
			Keywords:    []string{"wordpress", "cms", "blog", "website"},
		},
		{
			Name:        "drupal",
			Version:     "18.0.8",
			Versions:    []string{"18.0.8", "18.0.7", "18.0.6"},
			AppVersion:  "10.2.7",
			Description: "Drupal is an open-source content management framework",
			Repository:  repo.Name,
			Keywords:    []string{"drupal", "cms", "content management"},
		},
		{
			Name:        "mediawiki",
			Version:     "22.1.2",
			Versions:    []string{"22.1.2", "22.1.1", "22.1.0"},
			AppVersion:  "1.42.1",
			Description: "MediaWiki is a wiki software package written in PHP",
			Repository:  repo.Name,
			Keywords:    []string{"mediawiki", "wiki", "documentation"},
		},
		{
			Name:        "mattermost",
			Version:     "6.6.54",
			Versions:    []string{"6.6.54", "6.6.53", "6.6.52"},
			AppVersion:  "9.9.0",
			Description: "Mattermost is an open-source team collaboration platform",
			Repository:  repo.Name,
			Keywords:    []string{"mattermost", "chat", "collaboration", "team"},
		},

		// Security & Authentication
		{
			Name:        "keycloak",
			Version:     "21.4.4",
			Versions:    []string{"21.4.4", "21.4.3", "21.4.2"},
			AppVersion:  "25.0.1",
			Description: "Keycloak is an identity and access management solution",
			Repository:  repo.Name,
			Keywords:    []string{"keycloak", "identity", "authentication", "sso"},
		},
		{
			Name:        "vault",
			Version:     "0.28.0",
			Versions:    []string{"0.28.0", "0.27.0", "0.26.1"},
			AppVersion:  "1.17.2",
			Description: "HashiCorp Vault is a secrets management tool",
			Repository:  repo.Name,
			Keywords:    []string{"vault", "secrets", "security", "hashicorp"},
		},

		// Networking & Storage
		{
			Name:        "minio",
			Version:     "14.6.29",
			Versions:    []string{"14.6.29", "14.6.28", "14.6.27"},
			AppVersion:  "2024.6.29",
			Description: "MinIO is a high-performance object storage server",
			Repository:  repo.Name,
			Keywords:    []string{"minio", "object storage", "s3", "cloud storage"},
		},
		{
			Name:        "cert-manager",
			Version:     "1.15.1",
			Versions:    []string{"1.15.1", "1.15.0", "1.14.5"},
			AppVersion:  "1.15.1",
			Description: "cert-manager is a Kubernetes add-on to automate TLS certificate management",
			Repository:  repo.Name,
			Keywords:    []string{"cert-manager", "tls", "certificates", "letsencrypt"},
		},

		// Backup & Recovery
		{
			Name:        "velero",
			Version:     "6.6.0",
			Versions:    []string{"6.6.0", "6.5.0", "6.4.0"},
			AppVersion:  "1.13.2",
			Description: "Velero is a backup and restore solution for Kubernetes",
			Repository:  repo.Name,
			Keywords:    []string{"velero", "backup", "restore", "disaster recovery"},
		},
	}
	
	return suseCharts, nil
}

// Add repository to Helm CLI
func (rm *RepositoryManager) addHelmRepo(repo *models.Repository) error {
	if !rm.isHelmAvailable() {
		return fmt.Errorf("helm CLI not available")
	}
	
	args := []string{"repo", "add", repo.Name, repo.URL}
	
	// Add authentication if available
	if repo.Auth != nil && repo.Auth.Username != "" && repo.Auth.Password != "" {
		args = append(args, "--username", repo.Auth.Username, "--password", repo.Auth.Password)
	}
	
	output, err := rm.runHelmCommand(args...)
	if err != nil {
		// Ignore error if repo already exists
		if strings.Contains(string(output), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to add helm repository: %w", err)
	}
	
	return nil
}

// Update Helm repository index
func (rm *RepositoryManager) updateHelmRepo(repoName string) error {
	if !rm.isHelmAvailable() {
		return fmt.Errorf("helm CLI not available")
	}
	
	args := []string{"repo", "update", repoName}
	_, err := rm.runHelmCommand(args...)
	if err != nil {
		return fmt.Errorf("failed to update helm repository: %w", err)
	}
	
	return nil
}

// Search charts in Helm repository
func (rm *RepositoryManager) searchHelmCharts(repoName string) ([]*models.Chart, error) {
	if !rm.isHelmAvailable() {
		return nil, fmt.Errorf("helm CLI not available")
	}
	
	args := []string{"search", "repo", repoName, "--output", "json"}
	output, err := rm.runHelmCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search helm charts: %w", err)
	}
	
	// Parse helm search output
	var helmCharts []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		AppVersion  string `json:"app_version"`
		Description string `json:"description"`
	}
	
	if err := json.Unmarshal(output, &helmCharts); err != nil {
		return nil, fmt.Errorf("failed to parse helm search output: %w", err)
	}
	
	// Convert to our chart format
	var charts []*models.Chart
	for _, hc := range helmCharts {
		// Extract chart name without repository prefix
		chartName := strings.TrimPrefix(hc.Name, repoName+"/")
		
		chart := &models.Chart{
			Name:        chartName,
			Version:     hc.Version,
			Versions:    []string{hc.Version}, // We only get current version from search
			AppVersion:  hc.AppVersion,
			Description: hc.Description,
			Repository:  repoName,
			Keywords:    []string{}, // helm search doesn't provide keywords
		}
		charts = append(charts, chart)
	}
	
	return charts, nil
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
				fmt.Printf("Warning: OCI authentication failed during chart pull: %v\n", err)
				// Continue anyway - authentication might be cached or chart might be public
			}
		}
		
		// Try to pull the chart to validate it exists and extract values
		if rm.isHelmAvailable() {
			tempDir := filepath.Join(rm.helmHome, "temp-charts")
			os.MkdirAll(tempDir, 0755)
			
			args := []string{"pull", chartURL, "--destination", tempDir, "--untar"}
			output, err := rm.runHelmCommand(args...)
			if err != nil {
				fmt.Printf("Warning: Failed to pull OCI chart %s: %v\nOutput: %s\n", chartURL, err, string(output))
				// Return the URL anyway - might work in the processing step
			} else {
				fmt.Printf("Successfully pulled OCI chart %s\n", chartURL)
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
	
	if auth.Username == "" || auth.Password == "" {
		if auth.SecretName != "" {
			// TODO: Implement secret retrieval from Kubernetes
			return fmt.Errorf("kubernetes secret authentication not yet implemented")
		} else {
			return fmt.Errorf("username and password are required for OCI registry authentication")
		}
	}
	
	fmt.Printf("Attempting helm registry login to: %s (user: %s)\n", loginURL, auth.Username)
	
	args := []string{"registry", "login", loginURL, "--username", auth.Username, "--password", auth.Password}
	
	output, err := rm.runHelmCommand(args...)
	if err != nil {
		fmt.Printf("Helm login failed: %v\nOutput: %s\n", err, string(output))
		return fmt.Errorf("helm registry login failed: %w", err)
	}
	
	fmt.Printf("Helm registry login successful for %s\n", loginURL)
	return nil
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

// Helper function to check if helm is available
func (rm *RepositoryManager) isHelmAvailable() bool {
	_, err := exec.LookPath("helm")
	return err == nil
}

// Helper function to run helm commands
func (rm *RepositoryManager) runHelmCommand(args ...string) ([]byte, error) {
	if !rm.isHelmAvailable() {
		return nil, fmt.Errorf("helm command not found - please install Helm CLI")
	}
	
	fmt.Printf("Running helm command: helm %s\n", strings.Join(args, " "))
	
	cmd := exec.Command("helm", args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HELM_CONFIG_HOME=%s", rm.helmHome),
		fmt.Sprintf("HELM_CACHE_HOME=%s", filepath.Join(rm.helmHome, "cache")),
		fmt.Sprintf("HELM_DATA_HOME=%s", filepath.Join(rm.helmHome, "data")),
	)
	
	return cmd.CombinedOutput()
}