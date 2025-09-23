package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Registry manages plugin discovery and installation
type Registry struct {
	repositories []Repository
	installed    map[string]InstalledPlugin
	cachePath    string
	mu           sync.RWMutex
}

// Repository represents a plugin repository
type Repository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"` // "github", "gitlab", "http"
}

// PluginInfo contains information about a plugin
type PluginInfo struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description"`
	Author        string            `json:"author"`
	License       string            `json:"license"`
	Homepage      string            `json:"homepage"`
	Repository    string            `json:"repository"`
	Tags          []string          `json:"tags"`
	Compatibility Compatibility     `json:"compatibility"`
	Dependencies  []Dependency      `json:"dependencies"`
	Assets        map[string]string `json:"assets"`
}

// Compatibility defines version compatibility
type Compatibility struct {
	MinVersion string `json:"min_version"`
	MaxVersion string `json:"max_version"`
}

// Dependency represents a plugin dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InstalledPlugin represents an installed plugin
type InstalledPlugin struct {
	Info        PluginInfo `json:"info"`
	Path        string     `json:"path"`
	InstalledAt time.Time  `json:"installed_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Enabled     bool       `json:"enabled"`
}

// NewRegistry creates a new plugin registry
func NewRegistry(cachePath string) *Registry {
	return &Registry{
		repositories: make([]Repository, 0),
		installed:    make(map[string]InstalledPlugin),
		cachePath:    cachePath,
	}
}

// AddRepository adds a repository
func (r *Registry) AddRepository(repo Repository) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.repositories = append(r.repositories, repo)
}

// Search searches for plugins
func (r *Registry) Search(query string) ([]PluginInfo, error) {
	var results []PluginInfo

	for _, repo := range r.repositories {
		plugins, err := r.searchRepository(repo, query)
		if err != nil {
			continue // Skip failed repositories
		}
		results = append(results, plugins...)
	}

	return results, nil
}

// searchRepository searches a specific repository
func (r *Registry) searchRepository(repo Repository, query string) ([]PluginInfo, error) {
	switch repo.Type {
	case "http":
		return r.searchHTTPRepository(repo.URL, query)
	case "github":
		return r.searchGitHubRepository(repo.URL, query)
	default:
		return nil, fmt.Errorf("unsupported repository type: %s", repo.Type)
	}
}

// searchHTTPRepository searches an HTTP repository
func (r *Registry) searchHTTPRepository(url, query string) ([]PluginInfo, error) {
	// Fetch plugin list from HTTP endpoint
	resp, err := http.Get(fmt.Sprintf("%s/plugins.json", url))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var plugins []PluginInfo
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return nil, err
	}

	// Filter by query
	var results []PluginInfo
	lowerQuery := strings.ToLower(query)

	for _, plugin := range plugins {
		if strings.Contains(strings.ToLower(plugin.Name), lowerQuery) ||
			strings.Contains(strings.ToLower(plugin.Description), lowerQuery) ||
			containsTag(plugin.Tags, lowerQuery) {
			results = append(results, plugin)
		}
	}

	return results, nil
}

// searchGitHubRepository searches a GitHub repository
func (r *Registry) searchGitHubRepository(url, query string) ([]PluginInfo, error) {
	// This would use GitHub API to search for plugins
	// For now, return empty
	return []PluginInfo{}, nil
}

// Install installs a plugin
func (r *Registry) Install(name, version string) error {
	// Find the plugin
	pluginInfo, repo, err := r.findPlugin(name, version)
	if err != nil {
		return err
	}

	// Check compatibility
	if err := r.checkCompatibility(pluginInfo); err != nil {
		return err
	}

	// Check dependencies
	if err := r.checkDependencies(pluginInfo); err != nil {
		return err
	}

	// Download the plugin
	pluginPath, err := r.downloadPlugin(pluginInfo, repo)
	if err != nil {
		return err
	}

	// Register as installed
	r.mu.Lock()
	r.installed[name] = InstalledPlugin{
		Info:        *pluginInfo,
		Path:        pluginPath,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
		Enabled:     true,
	}
	r.mu.Unlock()

	// Save registry
	return r.save()
}

// Uninstall uninstalls a plugin
func (r *Registry) Uninstall(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.installed[name]
	if !exists {
		return fmt.Errorf("plugin %s not installed", name)
	}

	// Remove plugin files
	if err := os.RemoveAll(plugin.Path); err != nil {
		return err
	}

	// Remove from registry
	delete(r.installed, name)

	// Save registry
	return r.save()
}

// Update updates a plugin
func (r *Registry) Update(name string) error {
	r.mu.RLock()
	plugin, exists := r.installed[name]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not installed", name)
	}

	// Find latest version
	latestInfo, _, err := r.findPlugin(name, "latest")
	if err != nil {
		return err
	}

	// Check if update is needed
	if latestInfo.Version == plugin.Info.Version {
		return nil // Already up to date
	}

	// Uninstall old version
	if err := r.Uninstall(name); err != nil {
		return err
	}

	// Install new version
	return r.Install(name, latestInfo.Version)
}

// Enable enables a plugin
func (r *Registry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plugin, exists := r.installed[name]; exists {
		plugin.Enabled = true
		r.installed[name] = plugin
		return r.save()
	}

	return fmt.Errorf("plugin %s not installed", name)
}

// Disable disables a plugin
func (r *Registry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plugin, exists := r.installed[name]; exists {
		plugin.Enabled = false
		r.installed[name] = plugin
		return r.save()
	}

	return fmt.Errorf("plugin %s not installed", name)
}

// ListInstalled lists installed plugins
func (r *Registry) ListInstalled() []InstalledPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]InstalledPlugin, 0, len(r.installed))
	for _, plugin := range r.installed {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// GetInstalledPlugin gets an installed plugin
func (r *Registry) GetInstalledPlugin(name string) (InstalledPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.installed[name]
	return plugin, exists
}

// Load loads the registry from disk
func (r *Registry) Load() error {
	registryFile := filepath.Join(r.cachePath, "registry.json")

	data, err := os.ReadFile(registryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No registry file yet
		}
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return json.Unmarshal(data, &r.installed)
}

// save saves the registry to disk
func (r *Registry) save() error {
	registryFile := filepath.Join(r.cachePath, "registry.json")

	// Ensure cache directory exists
	if err := os.MkdirAll(r.cachePath, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.installed, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(registryFile, data, 0644)
}

// findPlugin finds a plugin in repositories
func (r *Registry) findPlugin(name, version string) (*PluginInfo, *Repository, error) {
	for _, repo := range r.repositories {
		plugin, err := r.getPluginInfo(repo, name, version)
		if err == nil {
			return plugin, &repo, nil
		}
	}

	return nil, nil, fmt.Errorf("plugin %s version %s not found", name, version)
}

// getPluginInfo gets plugin info from a repository
func (r *Registry) getPluginInfo(repo Repository, name, version string) (*PluginInfo, error) {
	switch repo.Type {
	case "http":
		return r.getHTTPPluginInfo(repo.URL, name, version)
	default:
		return nil, fmt.Errorf("unsupported repository type: %s", repo.Type)
	}
}

// getHTTPPluginInfo gets plugin info from HTTP repository
func (r *Registry) getHTTPPluginInfo(url, name, version string) (*PluginInfo, error) {
	resp, err := http.Get(fmt.Sprintf("%s/plugins/%s/%s/info.json", url, name, version))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin not found")
	}

	var info PluginInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

// downloadPlugin downloads a plugin
func (r *Registry) downloadPlugin(info *PluginInfo, repo *Repository) (string, error) {
	// Create plugin directory
	pluginDir := filepath.Join(r.cachePath, "plugins", info.Name, info.Version)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return "", err
	}

	// Download plugin assets
	for filename, url := range info.Assets {
		filePath := filepath.Join(pluginDir, filename)
		if err := r.downloadFile(url, filePath); err != nil {
			return "", err
		}
	}

	return pluginDir, nil
}

// downloadFile downloads a file
func (r *Registry) downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// checkCompatibility checks if a plugin is compatible
func (r *Registry) checkCompatibility(info *PluginInfo) error {
	// This would check against the current Gor version
	// For now, always return success
	return nil
}

// checkDependencies checks if dependencies are satisfied
func (r *Registry) checkDependencies(info *PluginInfo) error {
	for _, dep := range info.Dependencies {
		if _, exists := r.installed[dep.Name]; !exists {
			return fmt.Errorf("dependency %s not installed", dep.Name)
		}
	}
	return nil
}

// Helper functions

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.ToLower(t) == tag {
			return true
		}
	}
	return false
}
