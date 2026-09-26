package constructorcheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ming-0x0/constructorcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	testdata := filepath.Join(wd, "testdata")
	analysistest.Run(t, testdata, constructorcheck.Analyzer, "a", "b")
}
