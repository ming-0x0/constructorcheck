package constructorcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `constructorcheck verifies that structs marked with //constructor:required
(via a doc-comment directive on the type or its constructor, or a
constructor:"required" struct tag) are
instantiated exclusively via constructor functions.`

var Analyzer = &analysis.Analyzer{
	Name:      "constructorcheck",
	Doc:       Doc,
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	Run:       run,
	FactTypes: []analysis.Fact{new(requiredFact)},
}

// requiredFact marks a *types.TypeName as constructor-required. It carries
// no payload; its mere presence on an object is the signal.
//
// Facts are the idiomatic go/analysis mechanism for propagating
// per-object information computed while analyzing package A to any
// package B that imports A, without B ever needing A's source. The
// analysis driver also persists facts as part of its result cache, so
// repeated runs (e.g. under golangci-lint) don't recompute them either.
type requiredFact struct{}

func (*requiredFact) AFact()         {}
func (*requiredFact) String() string { return "constructor:required" }

// required is a single shared, stateless fact instance. Since requiredFact
// carries no payload, it is safe to reuse the same pointer for every
// ExportObjectFact/ImportObjectFact call instead of allocating a fresh one
// per composite literal or per type — the driver only ever reads its
// (empty) contents.
var required = new(requiredFact)

// compositeLitFilter is the node filter passed to inspector.Preorder. It
// never changes, so it's hoisted to a package-level var instead of being
// re-allocated on every run() call (one per analyzed package).
var compositeLitFilter = []ast.Node{(*ast.CompositeLit)(nil)}

func run(pass *analysis.Pass) (any, error) {
	// Export a fact for every type in *this* package that requires a
	// constructor, whether flagged via doc-comment directive or struct
	// tag. Downstream packages will later resolve these with a single
	// ImportObjectFact call instead of reparsing this package.
	exportRequiredFacts(pass)

	// A composite literal can only reference a Named type from another
	// package if that package is imported. A package with no imports
	// therefore cannot possibly contain a violation, so skip the AST
	// walk entirely in that case.
	if len(pass.Pkg.Imports()) == 0 {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder(compositeLitFilter, func(n ast.Node) {
		lit := n.(*ast.CompositeLit)

		tv, ok := pass.TypesInfo.Types[lit]
		if !ok {
			return
		}

		t := tv.Type
		if ptr, isPtr := t.(*types.Pointer); isPtr {
			t = ptr.Elem()
		}

		named, ok := t.(*types.Named)
		if !ok {
			return
		}

		obj := named.Obj()
		if obj == nil || obj.Pkg() == nil {
			return
		}

		// The defining package itself is exempt: that's how the
		// constructor is allowed to build the value in the first place.
		// Pointer comparison is valid and cheaper than comparing paths:
		// within a single pass, every object declared in the analyzed
		// package has obj.Pkg() == pass.Pkg by construction.
		if obj.Pkg() == pass.Pkg {
			return
		}

		if !pass.ImportObjectFact(obj, required) {
			return
		}

		pass.Reportf(
			lit.Pos(),
			"cannot instantiate %s.%s with struct literal: must be created using its constructor (marked with //constructor:required)",
			obj.Pkg().Name(),
			obj.Name(),
		)
	})

	return nil, nil
}

// exportRequiredFacts scans the current package's declarations and struct
// tags for the constructor:required marker and exports a fact for each
// matching type so other packages can query it in O(1).
func exportRequiredFacts(pass *analysis.Pass) {
	byDirective := scanPackageRequiredTypes(pass.Files)

	scope := pass.Pkg.Scope()
	for _, name := range scope.Names() {
		obj, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		if byDirective[name] || hasRequiredTag(obj) {
			pass.ExportObjectFact(obj, required)
		}
	}
}

func hasRequiredTag(obj *types.TypeName) bool {
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		tag := st.Tag(i)
		if strings.Contains(tag, `constructor:"required"`) {
			return true
		}
	}
	return false
}

// scanPackageRequiredTypes finds types marked //constructor:required either
// directly on their type declaration, or indirectly via a constructor
// function (func NewFoo() *Foo) carrying the same directive.
func scanPackageRequiredTypes(files []*ast.File) map[string]bool {
	// Lazily allocated: the common case is a package with zero
	// //constructor:required directives, so avoid the map allocation
	// (and its zero-value probing on every scope name) entirely then.
	var found map[string]bool
	mark := func(name string) {
		if found == nil {
			found = make(map[string]bool)
		}
		found[name] = true
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				hasDeclComment := hasConstructorDirective(genDecl.Doc)
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if hasDeclComment || hasConstructorDirective(typeSpec.Doc) || hasConstructorDirective(typeSpec.Comment) {
							mark(typeSpec.Name.Name)
						}
					}
				}
			}

			if fnDecl, ok := decl.(*ast.FuncDecl); ok {
				if hasConstructorDirective(fnDecl.Doc) {
					for _, retType := range getReturnNamedTypes(fnDecl.Type) {
						mark(retType)
					}
				}
			}
		}
	}

	return found
}

func getReturnNamedTypes(fnType *ast.FuncType) []string {
	if fnType == nil || fnType.Results == nil {
		return nil
	}
	var names []string
	for _, field := range fnType.Results.List {
		t := field.Type
		if star, ok := t.(*ast.StarExpr); ok {
			t = star.X
		}
		if ident, ok := t.(*ast.Ident); ok {
			names = append(names, ident.Name)
		}
	}
	return names
}

func hasConstructorDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "*/"))
		if strings.HasPrefix(text, "constructor:required") || text == "constructor:required" {
			return true
		}
	}
	return false
}
