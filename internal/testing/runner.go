package testing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ar4mirez/gor/pkg/gor"
)

// TestRunner manages test execution
type TestRunner struct {
	projectPath string
	coverage    bool
	verbose     bool
	timeout     time.Duration
	pattern     string
	tags        []string
}

// NewTestRunner creates a new test runner
func NewTestRunner(projectPath string) *TestRunner {
	return &TestRunner{
		projectPath: projectPath,
		timeout:     10 * time.Minute,
	}
}

// WithCoverage enables coverage reporting
func (tr *TestRunner) WithCoverage() *TestRunner {
	tr.coverage = true
	return tr
}

// WithVerbose enables verbose output
func (tr *TestRunner) WithVerbose() *TestRunner {
	tr.verbose = true
	return tr
}

// WithTimeout sets the test timeout
func (tr *TestRunner) WithTimeout(timeout time.Duration) *TestRunner {
	tr.timeout = timeout
	return tr
}

// WithPattern sets the test name pattern
func (tr *TestRunner) WithPattern(pattern string) *TestRunner {
	tr.pattern = pattern
	return tr
}

// WithTags sets build tags
func (tr *TestRunner) WithTags(tags ...string) *TestRunner {
	tr.tags = tags
	return tr
}

// RunAll runs all tests
func (tr *TestRunner) RunAll() error {
	return tr.run("./...")
}

// RunPackage runs tests for a specific package
func (tr *TestRunner) RunPackage(pkg string) error {
	return tr.run(pkg)
}

// RunFile runs tests in a specific file
func (tr *TestRunner) RunFile(file string) error {
	dir := filepath.Dir(file)
	return tr.run(dir)
}

// RunUnit runs unit tests only
func (tr *TestRunner) RunUnit() error {
	tr.tags = append(tr.tags, "unit")
	return tr.run("./...")
}

// RunIntegration runs integration tests only
func (tr *TestRunner) RunIntegration() error {
	tr.tags = append(tr.tags, "integration")
	return tr.run("./...")
}

// RunE2E runs end-to-end tests only
func (tr *TestRunner) RunE2E() error {
	tr.tags = append(tr.tags, "e2e")
	return tr.run("./...")
}

// run executes the go test command
func (tr *TestRunner) run(target string) error {
	args := []string{"test"}

	// Add target
	args = append(args, target)

	// Add flags
	if tr.verbose {
		args = append(args, "-v")
	}

	if tr.coverage {
		args = append(args, "-cover", "-coverprofile=coverage.out")
	}

	if tr.timeout > 0 {
		args = append(args, fmt.Sprintf("-timeout=%s", tr.timeout))
	}

	if tr.pattern != "" {
		args = append(args, "-run", tr.pattern)
	}

	if len(tr.tags) > 0 {
		args = append(args, "-tags", strings.Join(tr.tags, ","))
	}

	// Run the command
	cmd := exec.Command("go", args...)
	cmd.Dir = tr.projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// Generate coverage report if enabled
	if tr.coverage {
		return tr.generateCoverageReport()
	}

	return nil
}

// generateCoverageReport generates an HTML coverage report
func (tr *TestRunner) generateCoverageReport() error {
	cmd := exec.Command("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
	cmd.Dir = tr.projectPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate coverage report: %w", err)
	}

	fmt.Println("Coverage report generated: coverage.html")
	return nil
}

// Benchmark runs benchmarks
func (tr *TestRunner) Benchmark(target string) error {
	args := []string{"test", "-bench=."}

	if target != "" {
		args = append(args, target)
	} else {
		args = append(args, "./...")
	}

	if tr.verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = tr.projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// TestSuite manages a collection of test cases
type TestSuite struct {
	name      string
	testCases []TestCaseFunc
	setup     func()
	teardown  func()
}

// TestCaseFunc represents a test case function
type TestCaseFunc func(*TestCase)

// NewTestSuite creates a new test suite
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		name:      name,
		testCases: make([]TestCaseFunc, 0),
	}
}

// AddTest adds a test case to the suite
func (ts *TestSuite) AddTest(name string, testFunc TestCaseFunc) {
	ts.testCases = append(ts.testCases, testFunc)
}

// SetSetup sets the setup function
func (ts *TestSuite) SetSetup(setup func()) {
	ts.setup = setup
}

// SetTeardown sets the teardown function
func (ts *TestSuite) SetTeardown(teardown func()) {
	ts.teardown = teardown
}

// Run executes all test cases in the suite
func (ts *TestSuite) Run(t *testing.T, app gor.Application) {
	if ts.setup != nil {
		ts.setup()
	}

	for i, testFunc := range ts.testCases {
		t.Run(fmt.Sprintf("%s_%d", ts.name, i), func(t *testing.T) {
			tc := NewTestCase(t, app)
			defer tc.TearDown()
			testFunc(tc)
		})
	}

	if ts.teardown != nil {
		ts.teardown()
	}
}