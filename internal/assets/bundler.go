package assets

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Bundler bundles JavaScript modules
type Bundler struct {
	entryPoints  []string
	outputPath   string
	modules      map[string]*Module
	dependencies map[string][]string
	bundled      map[string]bool
	format       string // "iife", "esm", "cjs", "umd"
}

// Module represents a JavaScript module
type Module struct {
	Path         string
	Content      string
	Dependencies []string
	Exports      []string
	Imports      map[string]string
}

// NewBundler creates a new JavaScript bundler
func NewBundler(outputPath string) *Bundler {
	return &Bundler{
		entryPoints:  make([]string, 0),
		outputPath:   outputPath,
		modules:      make(map[string]*Module),
		dependencies: make(map[string][]string),
		bundled:      make(map[string]bool),
		format:       "iife", // Default to immediately invoked function expression
	}
}

// AddEntryPoint adds an entry point for bundling
func (b *Bundler) AddEntryPoint(path string) {
	b.entryPoints = append(b.entryPoints, path)
}

// SetFormat sets the output format
func (b *Bundler) SetFormat(format string) {
	b.format = format
}

// Bundle bundles all entry points
func (b *Bundler) Bundle() error {
	for _, entryPoint := range b.entryPoints {
		if err := b.bundleEntryPoint(entryPoint); err != nil {
			return err
		}
	}
	return nil
}

// bundleEntryPoint bundles a single entry point
func (b *Bundler) bundleEntryPoint(entryPoint string) error {
	// Parse all modules starting from entry point
	if err := b.parseModule(entryPoint); err != nil {
		return err
	}

	// Build dependency graph
	if err := b.buildDependencyGraph(); err != nil {
		return err
	}

	// Generate bundle
	bundle := b.generateBundle(entryPoint)

	// Write output
	outputFile := filepath.Join(b.outputPath, filepath.Base(entryPoint))
	return os.WriteFile(outputFile, []byte(bundle), 0644)
}

// parseModule parses a JavaScript module
func (b *Bundler) parseModule(path string) error {
	if _, exists := b.modules[path]; exists {
		return nil // Already parsed
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read module %s: %w", path, err)
	}

	module := &Module{
		Path:         path,
		Content:      string(content),
		Dependencies: make([]string, 0),
		Exports:      make([]string, 0),
		Imports:      make(map[string]string),
	}

	// Parse imports
	imports := b.parseImports(module.Content)
	for _, imp := range imports {
		resolvedPath := b.resolveImport(imp, path)
		module.Dependencies = append(module.Dependencies, resolvedPath)
		module.Imports[imp] = resolvedPath

		// Recursively parse dependencies
		if err := b.parseModule(resolvedPath); err != nil {
			return err
		}
	}

	// Parse exports
	module.Exports = b.parseExports(module.Content)

	b.modules[path] = module
	b.dependencies[path] = module.Dependencies

	return nil
}

// parseImports extracts import statements from JavaScript code
func (b *Bundler) parseImports(content string) []string {
	imports := make([]string, 0)

	// ES6 imports
	es6ImportRegex := regexp.MustCompile(`import\s+(?:.*?\s+from\s+)?["']([^"']+)["']`)
	matches := es6ImportRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}

	// CommonJS requires
	requireRegex := regexp.MustCompile(`require\(["']([^"']+)["']\)`)
	matches = requireRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}

	return imports
}

// parseExports extracts export statements from JavaScript code
func (b *Bundler) parseExports(content string) []string {
	exports := make([]string, 0)

	// ES6 exports
	exportRegex := regexp.MustCompile(`export\s+(?:default\s+)?(\w+)`)
	matches := exportRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			exports = append(exports, match[1])
		}
	}

	// CommonJS exports
	if strings.Contains(content, "module.exports") {
		exports = append(exports, "default")
	}

	return exports
}

// resolveImport resolves an import path
func (b *Bundler) resolveImport(importPath, fromPath string) string {
	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := filepath.Dir(fromPath)
		resolvedPath := filepath.Join(dir, importPath)

		// Add .js extension if missing
		if !strings.HasSuffix(resolvedPath, ".js") && !strings.HasSuffix(resolvedPath, ".mjs") {
			resolvedPath += ".js"
		}

		return resolvedPath
	}

	// Handle node_modules imports (simplified)
	if !strings.HasPrefix(importPath, "/") {
		// Look in node_modules
		nodeModulesPath := filepath.Join("node_modules", importPath)
		if _, err := os.Stat(nodeModulesPath); err == nil {
			return nodeModulesPath
		}

		// Look for index.js
		indexPath := filepath.Join(nodeModulesPath, "index.js")
		if _, err := os.Stat(indexPath); err == nil {
			return indexPath
		}
	}

	return importPath
}

// buildDependencyGraph builds the dependency graph
func (b *Bundler) buildDependencyGraph() error {
	// Simple topological sort to detect circular dependencies
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for module := range b.modules {
		if !visited[module] {
			if b.hasCycle(module, visited, recStack) {
				return fmt.Errorf("circular dependency detected")
			}
		}
	}

	return nil
}

// hasCycle detects cycles in the dependency graph
func (b *Bundler) hasCycle(module string, visited, recStack map[string]bool) bool {
	visited[module] = true
	recStack[module] = true

	for _, dep := range b.dependencies[module] {
		if !visited[dep] {
			if b.hasCycle(dep, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[module] = false
	return false
}

// generateBundle generates the final bundle
func (b *Bundler) generateBundle(entryPoint string) string {
	var bundle bytes.Buffer

	// Add wrapper based on format
	switch b.format {
	case "iife":
		bundle.WriteString("(function() {\n")
	case "umd":
		bundle.WriteString(b.generateUMDWrapper())
	}

	// Add module loader
	bundle.WriteString(b.generateModuleLoader())

	// Reset bundled tracker
	b.bundled = make(map[string]bool)

	// Add modules in dependency order
	b.addModuleToBundle(&bundle, entryPoint)

	// Add entry point execution
	bundle.WriteString(fmt.Sprintf("\n// Execute entry point\n__require('%s');\n", entryPoint))

	// Close wrapper
	switch b.format {
	case "iife":
		bundle.WriteString("})();\n")
	case "umd":
		bundle.WriteString("});\n")
	}

	return bundle.String()
}

// addModuleToBundle adds a module and its dependencies to the bundle
func (b *Bundler) addModuleToBundle(bundle *bytes.Buffer, modulePath string) {
	if b.bundled[modulePath] {
		return
	}

	module := b.modules[modulePath]
	if module == nil {
		return
	}

	// Add dependencies first
	for _, dep := range module.Dependencies {
		b.addModuleToBundle(bundle, dep)
	}

	// Add the module
	bundle.WriteString(fmt.Sprintf("\n// Module: %s\n", modulePath))
	bundle.WriteString(fmt.Sprintf("__modules['%s'] = function(exports, require, module) {\n", modulePath))

	// Transform imports to use the module loader
	content := b.transformImports(module.Content, module.Imports)
	bundle.WriteString(content)

	bundle.WriteString("\n};\n")

	b.bundled[modulePath] = true
}

// transformImports transforms import statements to use the module loader
func (b *Bundler) transformImports(content string, imports map[string]string) string {
	// Replace ES6 imports with requires
	for original, resolved := range imports {
		// Replace import statements
		importRegex := regexp.MustCompile(fmt.Sprintf(`import\s+(.*?)\s+from\s+["']%s["']`, regexp.QuoteMeta(original)))
		content = importRegex.ReplaceAllString(content, fmt.Sprintf(`const $1 = __require('%s')`, resolved))

		// Replace require calls
		requireRegex := regexp.MustCompile(fmt.Sprintf(`require\(["']%s["']\)`, regexp.QuoteMeta(original)))
		content = requireRegex.ReplaceAllString(content, fmt.Sprintf(`__require('%s')`, resolved))
	}

	// Replace export statements
	content = strings.ReplaceAll(content, "export default", "module.exports =")
	content = regexp.MustCompile(`export\s+(const|let|var|function|class)\s+(\w+)`).ReplaceAllString(
		content, "$1 $2; exports.$2 = $2",
	)

	return content
}

// generateModuleLoader generates the module loader code
func (b *Bundler) generateModuleLoader() string {
	return `
// Module loader
const __modules = {};
const __cache = {};

function __require(id) {
  if (__cache[id]) {
    return __cache[id].exports;
  }

  const module = { exports: {} };
  __cache[id] = module;

  if (__modules[id]) {
    __modules[id](module.exports, __require, module);
  } else {
    throw new Error('Module not found: ' + id);
  }

  return module.exports;
}
`
}

// generateUMDWrapper generates UMD wrapper
func (b *Bundler) generateUMDWrapper() string {
	return `(function(root, factory) {
  if (typeof define === 'function' && define.amd) {
    // AMD
    define([], factory);
  } else if (typeof module === 'object' && module.exports) {
    // CommonJS
    module.exports = factory();
  } else {
    // Browser globals
    root.Bundle = factory();
  }
}(typeof self !== 'undefined' ? self : this, function() {
`
}
