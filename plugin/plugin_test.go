package plugin_test

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"github.com/ming-0x0/constructorcheck"
	"github.com/ming-0x0/constructorcheck/plugin"
)

func TestPlugin(t *testing.T) {
	p, err := plugin.New(nil)
	if err != nil {
		t.Fatalf("unexpected error from New: %v", err)
	}

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("unexpected error from BuildAnalyzers: %v", err)
	}

	if len(analyzers) != 1 {
		t.Fatalf("expected 1 analyzer, got %d", len(analyzers))
	}

	if analyzers[0] != constructorcheck.Analyzer {
		t.Errorf("expected constructorcheck.Analyzer, got %v", analyzers[0])
	}

	if mode := p.GetLoadMode(); mode != register.LoadModeTypesInfo {
		t.Errorf("expected load mode %q, got %q", register.LoadModeTypesInfo, mode)
	}
}
