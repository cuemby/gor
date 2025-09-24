package assets

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Pipeline manages asset compilation and serving
type Pipeline struct {
	sourcePath   string
	outputPath   string
	manifestPath string
	processors   map[string]Processor
	manifest     map[string]string
	mu           sync.RWMutex
	watch        bool
	fingerprint  bool
	compress     bool
	cache        map[string]*Asset
}

// Asset represents a compiled asset
type Asset struct {
	Path         string
	Content      []byte
	ContentType  string
	Hash         string
	Compressed   bool
	LastModified time.Time
}

// Processor processes a specific type of asset
type Processor interface {
	Process(input []byte, options map[string]interface{}) ([]byte, error)
	Extensions() []string
}

// NewPipeline creates a new asset pipeline
func NewPipeline(sourcePath, outputPath string) *Pipeline {
	return &Pipeline{
		sourcePath:   sourcePath,
		outputPath:   outputPath,
		manifestPath: filepath.Join(outputPath, "manifest.json"),
		processors:   make(map[string]Processor),
		manifest:     make(map[string]string),
		cache:        make(map[string]*Asset),
		fingerprint:  true,
		compress:     true,
	}
}

// RegisterProcessor registers an asset processor
func (p *Pipeline) RegisterProcessor(name string, processor Processor) {
	p.processors[name] = processor
	for _, ext := range processor.Extensions() {
		p.processors[ext] = processor
	}
}

// Compile compiles all assets
func (p *Pipeline) Compile() error {
	// Clean output directory
	if err := os.RemoveAll(p.outputPath); err != nil {
		return fmt.Errorf("failed to clean output directory: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(p.outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk source directory
	err := filepath.WalkDir(p.sourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		return p.compileAsset(path)
	})

	if err != nil {
		return fmt.Errorf("failed to compile assets: %w", err)
	}

	// Write manifest
	return p.writeManifest()
}

// compileAsset compiles a single asset
func (p *Pipeline) compileAsset(sourcePath string) error {
	// Read source file
	input, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", sourcePath, err)
	}

	// Get relative path
	relPath, err := filepath.Rel(p.sourcePath, sourcePath)
	if err != nil {
		return err
	}

	// Process asset
	ext := filepath.Ext(sourcePath)
	output := input

	if processor, ok := p.processors[ext]; ok {
		processed, err := processor.Process(input, nil)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", sourcePath, err)
		}
		output = processed
	}

	// Calculate hash if fingerprinting is enabled
	var hash string
	outputPath := filepath.Join(p.outputPath, relPath)

	if p.fingerprint {
		h := md5.Sum(output)
		hash = fmt.Sprintf("%x", h)[:8]

		// Add hash to filename
		dir := filepath.Dir(outputPath)
		base := strings.TrimSuffix(filepath.Base(outputPath), ext)
		outputPath = filepath.Join(dir, fmt.Sprintf("%s-%s%s", base, hash, ext))
	}

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Compress if enabled
	if p.compress && shouldCompress(ext) {
		compressed, err := compressAsset(output)
		if err == nil && len(compressed) < len(output) {
			// Write compressed version
			if err := os.WriteFile(outputPath+".gz", compressed, 0644); err != nil {
				return fmt.Errorf("failed to write compressed %s: %w", outputPath, err)
			}
		}
	}

	// Write output file
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	// Update manifest
	p.mu.Lock()
	p.manifest[relPath] = filepath.Base(outputPath)
	p.mu.Unlock()

	// Update cache
	p.cacheAsset(relPath, &Asset{
		Path:         outputPath,
		Content:      output,
		ContentType:  getContentType(ext),
		Hash:         hash,
		Compressed:   p.compress && shouldCompress(ext),
		LastModified: time.Now(),
	})

	return nil
}

// Watch watches for asset changes
func (p *Pipeline) Watch() error {
	if !p.watch {
		return nil
	}

	// Implementation would use fsnotify or similar
	// For now, we'll use a simple polling approach
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		lastModTimes := make(map[string]time.Time)

		for range ticker.C {
			_ = filepath.WalkDir(p.sourcePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}

				info, err := d.Info()
				if err != nil {
					return nil
				}

				modTime := info.ModTime()
				if lastMod, exists := lastModTimes[path]; !exists || modTime.After(lastMod) {
					lastModTimes[path] = modTime
					if exists {
						// File changed, recompile
						fmt.Printf("Asset changed: %s\n", path)
						_ = p.compileAsset(path)
					}
				}

				return nil
			})
		}
	}()

	return nil
}

// GetAsset retrieves a compiled asset
func (p *Pipeline) GetAsset(path string) (*Asset, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check cache
	if asset, ok := p.cache[path]; ok {
		return asset, nil
	}

	// Check manifest
	if outputName, ok := p.manifest[path]; ok {
		outputPath := filepath.Join(p.outputPath, filepath.Dir(path), outputName)
		content, err := os.ReadFile(outputPath)
		if err != nil {
			return nil, err
		}

		asset := &Asset{
			Path:        outputPath,
			Content:     content,
			ContentType: getContentType(filepath.Ext(path)),
		}

		p.cacheAsset(path, asset)
		return asset, nil
	}

	return nil, fmt.Errorf("asset not found: %s", path)
}

// AssetPath returns the path to a compiled asset
func (p *Pipeline) AssetPath(path string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if outputName, ok := p.manifest[path]; ok {
		return "/assets/" + filepath.Dir(path) + "/" + outputName
	}

	return "/assets/" + path
}

// cacheAsset caches an asset
func (p *Pipeline) cacheAsset(path string, asset *Asset) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[path] = asset
}

// writeManifest writes the manifest file
func (p *Pipeline) writeManifest() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, err := json.MarshalIndent(p.manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p.manifestPath, data, 0644)
}

// Helper functions

func shouldCompress(ext string) bool {
	compressible := []string{".css", ".js", ".json", ".html", ".xml", ".svg", ".txt"}
	for _, c := range compressible {
		if ext == c {
			return true
		}
	}
	return false
}

func getContentType(ext string) string {
	types := map[string]string{
		".css":   "text/css",
		".js":    "application/javascript",
		".json":  "application/json",
		".html":  "text/html",
		".xml":   "application/xml",
		".svg":   "image/svg+xml",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".webp":  "image/webp",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".eot":   "application/vnd.ms-fontobject",
	}

	if ct, ok := types[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

func compressAsset(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	defer gw.Close()

	if _, err := gw.Write(data); err != nil {
		return nil, err
	}

	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
