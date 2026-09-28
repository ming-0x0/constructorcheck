// Package constructorcheck is a go/analysis analyzer that enforces two rules
// for structs marked with "constructor:required":
//
//  1. INSTANTIATION rule: outside the defining package, the struct must not
//     be created with a composite literal (User{...}, &User{...}). It must be
//     created via its constructor (NewUser).
//
//  2. READ-ONLY rule: outside the defining package, fields of the struct must
//     not be written (u.ID = 2, u.ID++, &u.ID, ...). To change state, callers
//     must go through methods provided by the defining package.
//
// A struct can be marked in any of these ways:
//
//	// 1. Directive on the type declaration
//	//constructor:required
//	type User struct { ID int }
//
//	// 2. Directive on the constructor (the type is inferred from the
//	//    function's return type)
//	//constructor:required
//	func NewUser(id int) *User { return &User{ID: id} }
//
//	// 3. Struct tag on any field
//	type User struct {
//	    ID int `constructor:"required"`
//	}
package constructorcheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc is the description shown by `-help` or in golangci-lint.
const Doc = `constructorcheck verifies that structs marked with //constructor:required
(via a doc-comment directive on the type or its constructor, or a
constructor:"required" struct tag) are instantiated exclusively via
constructor functions, and that their fields are never modified
(assigned, incremented, or address-taken) from outside the defining package.`

// Analyzer is the entry point of the analyzer. Register it with
// singlechecker/multichecker or as a golangci-lint plugin.
var Analyzer = &analysis.Analyzer{
	Name: "constructorcheck",
	Doc:  Doc,
	// inspect.Analyzer provides a pre-built *inspector.Inspector that is
	// shared by all analyzers, so traversing the AST is faster than running
	// ast.Inspect ourselves.
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
	// Declares the Fact types this analyzer exports/imports. This is
	// mandatory: without it, ExportObjectFact/ImportObjectFact will panic.
	FactTypes: []analysis.Fact{new(requiredFact)},
}

// ---------------------------------------------------------------------------
// Fact
// ---------------------------------------------------------------------------

// requiredFact marks a *types.TypeName as constructor-required. It carries no
// payload; its mere presence on an object is the signal.
//
// Why use a Fact?
// When analyzing package B, which imports package A, we need to know which of
// A's types are marked. A Fact lets A "send" that information to B without B
// having to re-parse A's source. The driver also caches facts, so repeated
// runs (e.g. under golangci-lint) are fast.
type requiredFact struct{}

// AFact is the marker method required for requiredFact to satisfy the
// analysis.Fact interface.
func (*requiredFact) AFact() {}

// String is used when printing the fact (debugging, analysistest).
func (*requiredFact) String() string { return "constructor:required" }

// required is a single shared instance. Since requiredFact is empty and the
// driver only reads it, we can reuse one pointer for every Export/Import call
// instead of allocating a new one each time.
var required = new(requiredFact)

// ---------------------------------------------------------------------------
// Node filter
// ---------------------------------------------------------------------------

// nodeFilter lists the node kinds we care about. The inspector only invokes
// the callback for these kinds and skips the rest of the AST.
//
//   - CompositeLit: instantiation rule (User{...})
//   - AssignStmt:   u.ID = 2, u.ID += 1, u.A, u.B = 1, 2
//   - IncDecStmt:   u.ID++, u.ID--
//   - RangeStmt:    for u.ID = range xs
//   - UnaryExpr:    &u.ID (taking an address allows indirect writes)
var nodeFilter = []ast.Node{
	(*ast.CompositeLit)(nil),
	(*ast.AssignStmt)(nil),
	(*ast.IncDecStmt)(nil),
	(*ast.RangeStmt)(nil),
	(*ast.UnaryExpr)(nil),
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

func run(pass *analysis.Pass) (any, error) {
	// STEP 1: export a fact for every marked type in the CURRENT package.
	// Packages importing this one will read these facts in STEP 2.
	// This must run BEFORE the early return below, because a package with no
	// imports can still be the one that defines a required struct.
	exportRequiredFacts(pass)

	// Optimization: a package with no imports cannot reference types from
	// other packages, so it cannot contain a violation. Skip the AST walk.
	if len(pass.Pkg.Imports()) == 0 {
		return nil, nil
	}

	// STEP 2: walk the AST looking for violations.
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {

		case *ast.CompositeLit:
			// Instantiation rule: User{...} / &User{...}
			checkCompositeLit(pass, n)

		case *ast.AssignStmt:
			// `:=` only declares new identifiers, never a selector such as
			// u.ID, so skip it. All other tokens (`=`, `+=`, `-=`, ...) must
			// have every left-hand side checked, since multi-assignment is
			// possible: u.A, other.B = 1, 2
			if n.Tok == token.DEFINE {
				return
			}
			for _, lhs := range n.Lhs {
				checkFieldWrite(pass, lhs, "assign to")
			}

		case *ast.IncDecStmt:
			// u.ID++ / u.ID--
			checkFieldWrite(pass, n.X, "modify")

		case *ast.RangeStmt:
			// `for u.ID = range xs` (Tok == ASSIGN) writes to u.ID on every
			// iteration. `for i := range xs` (Tok == DEFINE) only declares
			// new variables.
			if n.Tok != token.ASSIGN {
				return
			}
			if n.Key != nil {
				checkFieldWrite(pass, n.Key, "assign to")
			}
			if n.Value != nil {
				checkFieldWrite(pass, n.Value, "assign to")
			}

		case *ast.UnaryExpr:
			// &u.ID yields a writable pointer, which counts as an indirect
			// write. Note: &User{} is also a UnaryExpr, but its operand is a
			// CompositeLit, so checkFieldWrite ignores it (default branch);
			// the instantiation rule is already handled by the CompositeLit
			// case above.
			if n.Op == token.AND {
				checkFieldWrite(pass, n.X, "take the address of")
			}
		}
	})

	return nil, nil
}

// ---------------------------------------------------------------------------
// Rule 1: instantiation via composite literal
// ---------------------------------------------------------------------------

// checkCompositeLit reports an error if lit creates a constructor-required
// struct that belongs to another package.
func checkCompositeLit(pass *analysis.Pass, lit *ast.CompositeLit) {
	tv, ok := pass.TypesInfo.Types[lit]
	if !ok {
		return
	}

	// &User{} and User{} are both treated as the same type, User.
	named, ok := deref(tv.Type).(*types.Named)
	if !ok {
		return
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return
	}

	// The defining package is exempt: the constructor has to be able to
	// build the value. Pointer comparison is valid and cheaper than comparing
	// paths: within a single pass, every object declared in the analyzed
	// package has Pkg() == pass.Pkg by construction.
	if obj.Pkg() == pass.Pkg {
		return
	}

	// Is this type marked? (the fact is exported by the defining package)
	if !pass.ImportObjectFact(obj, required) {
		return
	}

	pass.Reportf(
		lit.Pos(),
		"cannot instantiate %s.%s with struct literal: must be created using its constructor (marked with //constructor:required)",
		obj.Pkg().Name(),
		obj.Name(),
	)
}

// ---------------------------------------------------------------------------
// Rule 2: fields are read-only outside the defining package
// ---------------------------------------------------------------------------

// checkFieldWrite checks whether target (the left-hand side of a write, or
// the operand of &) resolves to a field of a constructor-required struct
// that lives in another package. If so, it reports an error.
//
// The function walks the expression chain from the outside in, so nested
// writes are attributed to the required struct that actually owns the memory:
//
//	u.ID = 1              -> selector ID on User             (caught immediately)
//	u.Address.City = "x"  -> City belongs to Address (not marked), but Address
//	                         is stored INLINE in User -> caught at Address
//	u.Arr[0] = 1          -> Arr is an array stored inline in User -> caught at Arr
//	u.Ptr.X = 1           -> Ptr is a pointer; X lives in different memory that
//	                         User does NOT own -> stop, no report (unless X's
//	                         own type is constructor-required, in which case it
//	                         is caught at X's selector step)
func checkFieldWrite(pass *analysis.Pass, target ast.Expr, verb string) {
	expr := target

	for {
		switch e := expr.(type) {

		case *ast.ParenExpr:
			// (u.ID) = 1 -> unwrap the parentheses.
			expr = e.X

		case *ast.IndexExpr:
			// Only ARRAYS are stored inline in the struct: u.Arr[0] = 1
			// changes the struct's own memory.
			// SLICES and MAPS are different: u.Tags[0] = "x" or
			// u.Meta["k"] = v only mutate shared backing storage, not the
			// field itself -> NOT reported.
			// We deliberately do not deref here: an array behind a pointer
			// (*[3]int) also lives in different memory, so it is not owned
			// by the struct.
			t := pass.TypesInfo.TypeOf(e.X)
			if t == nil {
				return
			}
			if _, isArr := t.Underlying().(*types.Array); !isArr {
				return
			}
			expr = e.X

		case *ast.SelectorExpr:
			// Selections only contains field/method selectors. A
			// package-qualified selector (pkg.Var) is not in the map, so
			// sel == nil for it.
			sel := pass.TypesInfo.Selections[e]
			if sel == nil || sel.Kind() != types.FieldVal {
				// pkg.Var, method value (u.Method), ... -> not a field write.
				return
			}

			// Does this field belong to a constructor-required struct from
			// another package? This also handles fields promoted through
			// embedded structs.
			if owner := requiredOwner(pass, sel); owner != nil {
				pass.Reportf(
					target.Pos(),
					"cannot %s field %s of %s.%s: fields of //constructor:required types are read-only outside their package (use a method provided by the package instead)",
					verb,
					sel.Obj().Name(),
					owner.Pkg().Name(),
					owner.Name(),
				)
				return
			}

			// No forbidden owner found yet. Continue inward (e.X) to see
			// whether the enclosing expression is stored inline in a
			// forbidden struct.
			//
			// EXCEPTION: if e.X is a pointer, the write goes through the
			// pointer and that memory is not owned by the enclosing struct,
			// so stop here to avoid false positives. (When e.X is the root
			// variable `u` of type *User, it was already checked in
			// requiredOwner above.)
			if xt := pass.TypesInfo.TypeOf(e.X); xt != nil {
				if _, isPtr := xt.Underlying().(*types.Pointer); isPtr {
					return
				}
			}
			expr = e.X

		default:
			// A bare identifier (x = 1), function call, composite literal,
			// ... -> not a write to a field of a forbidden struct.
			return
		}
	}
}

// requiredOwner walks the field-selection path (including embedded structs)
// and returns the first constructor-required type from ANOTHER package, or nil
// if there is none.
//
// Example: for `w.ID`, where W (in the current package) embeds model.User:
//
//	type W struct{ model.User }
//
// sel.Index() = [0, k] (0 = the embedded User field, k = position of ID in
// User). We walk W -> User and find that User is marked, so we return User.
func requiredOwner(pass *analysis.Pass, sel *types.Selection) *types.TypeName {
	t := sel.Recv()
	idx := sel.Index() // path: idx[i] is the field index at level i

	for i := 0; ; i++ {
		// Each level may be a pointer or a value: strip the pointer first.
		t = deref(t)

		// Is the current level a constructor-required type in another package?
		if named, ok := t.(*types.Named); ok {
			obj := named.Obj()
			if obj != nil && obj.Pkg() != nil && obj.Pkg() != pass.Pkg &&
				pass.ImportObjectFact(obj, required) {
				return obj
			}
		}

		// The last level (the one holding the selected field) is checked.
		if i >= len(idx)-1 {
			return nil
		}

		// Descend into the next embedded level.
		st, ok := t.Underlying().(*types.Struct)
		if !ok {
			return nil
		}
		t = st.Field(idx[i]).Type()
	}
}

// deref strips one level of pointer: *T -> T, T -> T.
func deref(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

// ---------------------------------------------------------------------------
// Fact export (defining-package side)
// ---------------------------------------------------------------------------

// exportRequiredFacts scans the current package's declarations and struct
// tags for the constructor:required marker and exports a fact for each
// marked type, so other packages can query it in O(1) via ImportObjectFact.
func exportRequiredFacts(pass *analysis.Pass) {
	// Names of types marked via directive (on the type or on a constructor).
	byDirective := scanPackageRequiredTypes(pass.Files)

	scope := pass.Pkg.Scope()
	for _, name := range scope.Names() {
		obj, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue // skip vars, consts, funcs, ...
		}
		if byDirective[name] || hasRequiredTag(obj) {
			pass.ExportObjectFact(obj, required)
		}
	}
}

// hasRequiredTag reports whether the type is a struct with at least one field
// carrying the tag `constructor:"required"`.
func hasRequiredTag(obj *types.TypeName) bool {
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		if strings.Contains(st.Tag(i), `constructor:"required"`) {
			return true
		}
	}
	return false
}

// scanPackageRequiredTypes finds types marked //constructor:required by
// directive, in two ways:
//
//  1. Directly on the type declaration (the GenDecl doc, the TypeSpec doc, or
//     a trailing comment).
//  2. Indirectly via a constructor: if `func NewFoo() *Foo` carries the
//     directive, the named types in its results (Foo) are marked.
//
// It returns a map of type name -> true. The result may be nil (reading from
// a nil map is safe).
func scanPackageRequiredTypes(files []*ast.File) map[string]bool {
	// Lazily allocated: most packages have no directives at all, so avoid
	// creating an unused map.
	var found map[string]bool
	mark := func(name string) {
		if found == nil {
			found = make(map[string]bool)
		}
		found[name] = true
	}

	for _, file := range files {
		for _, decl := range file.Decls {

			// Case 1: type declarations.
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				// `//constructor:required` placed above `type (...)` or
				// `type X ...` lives in the GenDecl's doc.
				hasDeclComment := hasConstructorDirective(genDecl.Doc)
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue // import/const/var spec
					}
					if hasDeclComment ||
						hasConstructorDirective(typeSpec.Doc) ||
						hasConstructorDirective(typeSpec.Comment) {
						mark(typeSpec.Name.Name)
					}
				}
			}

			// Case 2: directive on a constructor function.
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

// getReturnNamedTypes returns the names of the types (Foo or *Foo) that appear
// in the function's results. Only simple same-package identifiers are
// supported; forms like pkg.Foo, []Foo, or map[..]Foo are ignored.
func getReturnNamedTypes(fnType *ast.FuncType) []string {
	if fnType == nil || fnType.Results == nil {
		return nil
	}
	var names []string
	for _, field := range fnType.Results.List {
		t := field.Type
		if star, ok := t.(*ast.StarExpr); ok {
			t = star.X // *Foo -> Foo
		}
		if ident, ok := t.(*ast.Ident); ok {
			names = append(names, ident.Name)
		}
	}
	return names
}

// hasConstructorDirective reports whether a comment group contains
// `constructor:required`. It accepts `//constructor:required`,
// `// constructor:required`, and the block form `/* constructor:required */`.
func hasConstructorDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		// Strip the comment markers and surrounding whitespace, leaving only
		// the plain text.
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "*/"))
		if strings.HasPrefix(text, "constructor:required") {
			return true
		}
	}
	return false
}
