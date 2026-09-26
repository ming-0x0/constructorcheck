// Package plugin registers the constructorcheck analyzer as a golangci-lint
// custom module plugin. It is only pulled in when someone builds a custom
// golangci-lint binary with `golangci-lint custom`; library consumers who
// just import github.com/ming-0x0/constructorcheck never see this dependency.
//
// See https://golangci-lint.run/plugins/module-plugins/ for the plugin
// module contract this implements.
package plugin

import (
	"golang.org/x/tools/go/analysis"

	"github.com/golangci/plugin-module-register/register"
	"github.com/ming-0x0/constructorcheck"
)

func init() {
	register.Plugin("constructorcheck", New)
}

// New satisfies register.NewPlugin. golangci-lint calls it once per run,
// passing whatever `settings:` block the user configured for this linter
// in .golangci.yml. constructorcheck takes no settings today, hence the blank
// identifier — extend this if you later add configurable options.
func New(_ any) (register.LinterPlugin, error) {
	return &constructorcheckPlugin{}, nil
}

type constructorcheckPlugin struct{}

func (*constructorcheckPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{constructorcheck.Analyzer}, nil
}

func (*constructorcheckPlugin) GetLoadMode() string {
	// constructorcheck inspects go/types info (struct tags, named types), so it
	// needs full type-checking, not just syntax.
	return register.LoadModeTypesInfo
}
