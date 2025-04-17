package tester

import (
	"errors"
	"fmt"
	"os"
	"time"

	"dash-lang.io/src/ast"
	"dash-lang.io/src/builder"
	"dash-lang.io/src/evaluator"
)

type TestResult struct {
	Name     string
	Duration time.Duration
	Error    error
	Passed   bool
}

type TestRunner struct {
	results []TestResult
	passed  int
	failed  int
	// Map of all project libraries
	libraries map[string]*ast.Library
	// Directory where tests are being run
	testDir string
}

func NewTestRunner(dir string) *TestRunner {
	return &TestRunner{
		results:   make([]TestResult, 0),
		libraries: make(map[string]*ast.Library),
		testDir:   dir,
	}
}

// Run executes all tests in the current directory
func (tr *TestRunner) RunAll() error {
	// Build entire project from directory
	var err error
	cfg := &builder.Config{
		SrcDir:      tr.testDir,
		DashHomeDir: os.Getenv("DASH_HOME"),
	}
	tr.libraries, err = builder.BuildProject(cfg)
	if err != nil {
		return fmt.Errorf("error building project: %v", err)
	}

	// Create evaluator with all libraries
	eval := evaluator.New(tr.libraries)

	// run all tests for all libraries
	for _, lib := range tr.libraries {
		// Create new context for test
		ctx := evaluator.NewContext(nil)
		eval.InitialiseLib(lib, ctx)
		for _, fn := range lib.Functions {
			if isTestFunction(fn) {
				result := tr.runTest(fn, eval, ctx)
				tr.results = append(tr.results, result)
				if result.Passed {
					tr.passed++
				} else {
					tr.failed++
				}
				tr.printTestResult(result)
			}
		}
	}

	tr.printSummary()
	return nil
}

func isTestFunction(fn *ast.FunctionExpression) bool {
	for _, attr := range fn.Attributes {
		if attr.Equal(ast.Test) {
			return true
		}
	}
	return false
}

func (tr *TestRunner) runTest(fn *ast.FunctionExpression, eval *evaluator.Evaluator, ctx *evaluator.Context) TestResult {
	result := TestResult{
		Name: fn.Name.Value,
	}

	start := time.Now()

	// Execute test function
	res := eval.Eval(fn.Body, ctx)

	// Check for errors
	ret, ok := res.(*evaluator.Return)
	if !ok {
		result.Passed = true
	} else {
		err, ok := ret.Values[len(ret.Values)-1].(*evaluator.Error)
		if !ok {
			// this is an implementation error
			panic("incorrect return type when running test")
		}
		if err != nil {
			result.Error = errors.New(err.Err)
			result.Passed = false
		}
	}

	result.Duration = time.Since(start)
	return result
}

func (tr *TestRunner) printTestResult(result TestResult) {
	if result.Passed {
		fmt.Printf("\x1b[32m[PASS]\x1b[0m %s in %v\n", result.Name, result.Duration)
	} else {
		fmt.Printf("\x1b[31m[FAIL]\x1b[0m %s\n", result.Name)
		fmt.Printf("    |- Error: %v\n", result.Error)
	}
}

func (tr *TestRunner) printSummary() {
	total := tr.passed + tr.failed
	fmt.Printf("\nTest Summary: %d total, \x1b[32m%d passed\x1b[0m, \x1b[31m%d failed\x1b[0m\n",
		total, tr.passed, tr.failed)
}
