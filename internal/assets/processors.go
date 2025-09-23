package assets

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CSSProcessor processes CSS files
type CSSProcessor struct {
	minify     bool
	autoprefix bool
}

// NewCSSProcessor creates a new CSS processor
func NewCSSProcessor(minify, autoprefix bool) *CSSProcessor {
	return &CSSProcessor{
		minify:     minify,
		autoprefix: autoprefix,
	}
}

// Process processes CSS content
func (p *CSSProcessor) Process(input []byte, options map[string]interface{}) ([]byte, error) {
	css := string(input)

	// Process imports
	css = p.processImports(css)

	// Add vendor prefixes if enabled
	if p.autoprefix {
		css = p.addVendorPrefixes(css)
	}

	// Minify if enabled
	if p.minify {
		css = p.minifyCSS(css)
	}

	return []byte(css), nil
}

// Extensions returns supported extensions
func (p *CSSProcessor) Extensions() []string {
	return []string{".css"}
}

// processImports processes @import statements
func (p *CSSProcessor) processImports(css string) string {
	importRegex := regexp.MustCompile(`@import\s+["']([^"']+)["'];?`)
	return importRegex.ReplaceAllStringFunc(css, func(match string) string {
		// In a real implementation, we would read and inline the imported file
		// For now, we'll just return the original import
		return match
	})
}

// addVendorPrefixes adds vendor prefixes to CSS properties
func (p *CSSProcessor) addVendorPrefixes(css string) string {
	prefixes := map[string][]string{
		"transform":       {"-webkit-", "-moz-", "-ms-", "-o-"},
		"transition":      {"-webkit-", "-moz-", "-o-"},
		"animation":       {"-webkit-", "-moz-", "-o-"},
		"box-shadow":      {"-webkit-", "-moz-"},
		"border-radius":   {"-webkit-", "-moz-"},
		"flex":            {"-webkit-", "-ms-"},
		"flexbox":         {"-webkit-", "-ms-"},
		"user-select":     {"-webkit-", "-moz-", "-ms-"},
		"background-size": {"-webkit-", "-moz-", "-o-"},
		"background-clip": {"-webkit-"},
	}

	for property, vendorPrefixes := range prefixes {
		regex := regexp.MustCompile(fmt.Sprintf(`(\s|{)((%s):[^;]+;)`, property))
		css = regex.ReplaceAllStringFunc(css, func(match string) string {
			var result strings.Builder
			parts := strings.SplitN(match, ":", 2)
			if len(parts) != 2 {
				return match
			}

			prefix := parts[0]
			value := parts[1]

			// Add vendor prefixed versions
			for _, vendor := range vendorPrefixes {
				result.WriteString(fmt.Sprintf("%s%s%s:%s", prefix, vendor, property, value))
			}
			// Add original
			result.WriteString(match)

			return result.String()
		})
	}

	return css
}

// minifyCSS minifies CSS content
func (p *CSSProcessor) minifyCSS(css string) string {
	// Remove comments
	commentRegex := regexp.MustCompile(`/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`)
	css = commentRegex.ReplaceAllString(css, "")

	// Remove unnecessary whitespace
	css = regexp.MustCompile(`\s+`).ReplaceAllString(css, " ")
	css = strings.ReplaceAll(css, ": ", ":")
	css = strings.ReplaceAll(css, "; ", ";")
	css = strings.ReplaceAll(css, " {", "{")
	css = strings.ReplaceAll(css, "{ ", "{")
	css = strings.ReplaceAll(css, " }", "}")
	css = strings.ReplaceAll(css, "} ", "}")
	css = strings.ReplaceAll(css, ", ", ",")

	// Remove last semicolon before closing brace
	css = strings.ReplaceAll(css, ";}", "}")

	return strings.TrimSpace(css)
}

// JavaScriptProcessor processes JavaScript files
type JavaScriptProcessor struct {
	minify     bool
	sourceMaps bool
}

// NewJavaScriptProcessor creates a new JavaScript processor
func NewJavaScriptProcessor(minify, sourceMaps bool) *JavaScriptProcessor {
	return &JavaScriptProcessor{
		minify:     minify,
		sourceMaps: sourceMaps,
	}
}

// Process processes JavaScript content
func (p *JavaScriptProcessor) Process(input []byte, options map[string]interface{}) ([]byte, error) {
	js := string(input)

	// Process imports/requires (basic implementation)
	js = p.processImports(js)

	// Minify if enabled
	if p.minify {
		js = p.minifyJS(js)
	}

	return []byte(js), nil
}

// Extensions returns supported extensions
func (p *JavaScriptProcessor) Extensions() []string {
	return []string{".js", ".mjs"}
}

// processImports processes import statements
func (p *JavaScriptProcessor) processImports(js string) string {
	// Basic import processing
	// In a real implementation, this would handle ES6 modules properly
	return js
}

// minifyJS minifies JavaScript content
func (p *JavaScriptProcessor) minifyJS(js string) string {
	// Remove single-line comments
	js = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(js, "")

	// Remove multi-line comments
	js = regexp.MustCompile(`/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`).ReplaceAllString(js, "")

	// Remove unnecessary whitespace (basic)
	js = regexp.MustCompile(`\s+`).ReplaceAllString(js, " ")
	js = strings.ReplaceAll(js, " = ", "=")
	js = strings.ReplaceAll(js, " + ", "+")
	js = strings.ReplaceAll(js, " - ", "-")
	js = strings.ReplaceAll(js, " * ", "*")
	js = strings.ReplaceAll(js, " / ", "/")
	js = strings.ReplaceAll(js, " {", "{")
	js = strings.ReplaceAll(js, "{ ", "{")
	js = strings.ReplaceAll(js, " }", "}")
	js = strings.ReplaceAll(js, "} ", "}")
	js = strings.ReplaceAll(js, " (", "(")
	js = strings.ReplaceAll(js, "( ", "(")
	js = strings.ReplaceAll(js, " )", ")")
	js = strings.ReplaceAll(js, ") ", ")")

	return strings.TrimSpace(js)
}

// ImageProcessor processes image files
type ImageProcessor struct {
	optimize  bool
	maxWidth  int
	maxHeight int
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(optimize bool, maxWidth, maxHeight int) *ImageProcessor {
	return &ImageProcessor{
		optimize:  optimize,
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
	}
}

// Process processes image content
func (p *ImageProcessor) Process(input []byte, options map[string]interface{}) ([]byte, error) {
	// In a real implementation, this would:
	// - Resize images if needed
	// - Optimize file size
	// - Convert formats if needed
	// For now, we'll just pass through
	return input, nil
}

// Extensions returns supported extensions
func (p *ImageProcessor) Extensions() []string {
	return []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"}
}

// SassProcessor processes SASS/SCSS files
type SassProcessor struct {
	minify bool
}

// NewSassProcessor creates a new SASS processor
func NewSassProcessor(minify bool) *SassProcessor {
	return &SassProcessor{
		minify: minify,
	}
}

// Process processes SASS/SCSS content
func (p *SassProcessor) Process(input []byte, options map[string]interface{}) ([]byte, error) {
	// In a real implementation, this would compile SASS to CSS
	// For now, we'll treat it as CSS
	cssProcessor := NewCSSProcessor(p.minify, false)
	return cssProcessor.Process(input, options)
}

// Extensions returns supported extensions
func (p *SassProcessor) Extensions() []string {
	return []string{".sass", ".scss"}
}

// TypeScriptProcessor processes TypeScript files
type TypeScriptProcessor struct {
	minify     bool
	sourceMaps bool
	target     string // ES5, ES6, etc.
}

// NewTypeScriptProcessor creates a new TypeScript processor
func NewTypeScriptProcessor(minify, sourceMaps bool, target string) *TypeScriptProcessor {
	return &TypeScriptProcessor{
		minify:     minify,
		sourceMaps: sourceMaps,
		target:     target,
	}
}

// Process processes TypeScript content
func (p *TypeScriptProcessor) Process(input []byte, options map[string]interface{}) ([]byte, error) {
	// In a real implementation, this would compile TypeScript to JavaScript
	// For now, we'll treat it as JavaScript
	jsProcessor := NewJavaScriptProcessor(p.minify, p.sourceMaps)
	return jsProcessor.Process(input, options)
}

// Extensions returns supported extensions
func (p *TypeScriptProcessor) Extensions() []string {
	return []string{".ts", ".tsx"}
}

// ConcatenateProcessor concatenates multiple files
type ConcatenateProcessor struct {
	separator string
}

// NewConcatenateProcessor creates a new concatenate processor
func NewConcatenateProcessor(separator string) *ConcatenateProcessor {
	if separator == "" {
		separator = "\n"
	}
	return &ConcatenateProcessor{
		separator: separator,
	}
}

// Process concatenates multiple files
func (p *ConcatenateProcessor) Process(input []byte, options map[string]interface{}) ([]byte, error) {
	if options == nil {
		return input, nil
	}

	if files, ok := options["files"].([]string); ok {
		var buffer bytes.Buffer
		buffer.Write(input)

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file, err)
			}
			buffer.WriteString(p.separator)
			buffer.Write(content)
		}

		return buffer.Bytes(), nil
	}

	return input, nil
}

// Extensions returns supported extensions
func (p *ConcatenateProcessor) Extensions() []string {
	return []string{} // Works with any file type
}
