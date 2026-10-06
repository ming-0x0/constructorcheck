// Package constructorcheck is a go/analysis analyzer that enforces three rules
// for types marked with "constructor:required":
//
//  1. INSTANTIATION rule: outside the defining package, the type must not
//     be created with a literal (User{...}, MySlice{...}), type conversion
//     (MyInt(x)), built-in make/new (make(MySlice), new(User)), or zero-value
//     declaration (var u User). It must be created via its constructor (NewUser).
//
//  2. READ-ONLY rule: outside the defining package, fields and elements of the type must
//     not be written (u.ID = 2, s[0] = 2, *p = 2, copy(s, ...), delete(m, ...)).
//     To change state, callers must go through methods provided by the defining package.
//
//  3. NO-COMPARISON rule: outside the defining package, neither the type
//     itself, its fields, its elements, nor the results of its methods may be compared
//     (u.ID == 2, s[0] == 2, status == "active", switch u.ID { ... }). Callers must rely on
//     encapsulated domain methods (e.g. u.IsAdmin(), s.IsActive()).
//
// A type can be marked in any of these ways:
//
//	// 1. Directive on the type declaration
//	//constructor:required
//	type User struct { ID int }
//
//	//constructor:required
//	type Status string
//
//	// 2. Directive on the constructor (the type is inferred from the
//	//    function's primary return type)
//	//constructor:required
//	func NewUser(id int) (*User, error) { return &User{ID: id}, nil }
//
//	// 3. Struct tag on any field (for structs)
//	type User struct {
//	    ID int `constructor:"required"`
//	}
package constructorcheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc is the description shown by `-help` or in golangci-lint.
const Doc = `constructorcheck verifies that types marked with //constructor:required
(via a doc-comment directive on the type or its constructor, or a
constructor:"required" struct tag) are instantiated exclusively via
constructor functions, that their fields and elements are never modified outside
the defining package, and that neither the types, their fields/elements, nor
their method results are compared directly outside the defining package.`

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
type requiredFact struct{}

func (*requiredFact) AFact() {}

func (*requiredFact) String() string { return "constructor:required" }

var required = new(requiredFact)

// ---------------------------------------------------------------------------
// Checker Context (per-pass)
// ---------------------------------------------------------------------------

type checker struct {
	pass  *analysis.Pass
	cache map[*types.TypeName]bool
}

func newChecker(pass *analysis.Pass) *checker {
	return &checker{
		pass:  pass,
		cache: make(map[*types.TypeName]bool),
	}
}

func (c *checker) isFactRequired(obj *types.TypeName) bool {
	if val, ok := c.cache[obj]; ok {
		return val
	}
	res := c.pass.ImportObjectFact(obj, required)
	c.cache[obj] = res
	return res
}

// isSamePackageOrTest returns true if pkg is the analyzed package or an external
// test package for the analyzed package (e.g. pkg "foo" and "foo_test").
func isSamePackageOrTest(pass *analysis.Pass, pkg *types.Package) bool {
	if pkg == pass.Pkg {
		return true
	}
	if pkg == nil || pass.Pkg == nil {
		return false
	}
	if pass.Pkg.Name() == pkg.Name()+"_test" {
		if strings.TrimSuffix(pass.Pkg.Path(), "_test") == pkg.Path() {
			return true
		}
	}
	return false
}

// foreignRequired returns TypeName if t (or *t) is a constructor-required
// type defined in another package (excluding external test packages).
func (c *checker) foreignRequired(t types.Type) *types.TypeName {
	if t == nil {
		return nil
	}
	named, ok := types.Unalias(deref(t)).(*types.Named)
	if !ok {
		return nil
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil || isSamePackageOrTest(c.pass, obj.Pkg()) || !c.isFactRequired(obj) {
		return nil
	}
	return obj
}

// deref strips one level of pointer and unwraps type aliases.
func deref(t types.Type) types.Type {
	if t == nil {
		return nil
	}
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		return types.Unalias(ptr.Elem())
	}
	return t
}

// ---------------------------------------------------------------------------
// Node filter
// ---------------------------------------------------------------------------

var nodeFilter = []ast.Node{
	(*ast.CompositeLit)(nil),
	(*ast.CallExpr)(nil),
	(*ast.ValueSpec)(nil),
	(*ast.AssignStmt)(nil),
	(*ast.IncDecStmt)(nil),
	(*ast.RangeStmt)(nil),
	(*ast.UnaryExpr)(nil),
	(*ast.BinaryExpr)(nil),
	(*ast.SwitchStmt)(nil),
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

func run(pass *analysis.Pass) (any, error) {
	// STEP 1: export facts for marked types in the CURRENT package.
	exportRequiredFacts(pass)

	// Optimization: if no imports, cannot reference types from other packages.
	if len(pass.Pkg.Imports()) == 0 {
		return nil, nil
	}

	// Optimization: if no imported facts from other packages, skip AST walk.
	hasForeign := false
	for _, f := range pass.AllObjectFacts() {
		if !isSamePackageOrTest(pass, f.Object.Pkg()) {
			hasForeign = true
			break
		}
	}
	if !hasForeign {
		return nil, nil
	}

	// Identify generated files to skip checking them.
	type fileRange struct {
		start token.Pos
		end   token.Pos
	}
	var generatedRanges []fileRange
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			generatedRanges = append(generatedRanges, fileRange{
				start: file.FileStart,
				end:   file.FileEnd,
			})
		}
	}

	isGenerated := func(pos token.Pos) bool {
		for _, r := range generatedRanges {
			if pos >= r.start && pos < r.end {
				return true
			}
		}
		return false
	}

	// STEP 2: walk the AST looking for violations.
	c := newChecker(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder(nodeFilter, func(n ast.Node) {
		if isGenerated(n.Pos()) {
			return
		}

		switch n := n.(type) {

		case *ast.CompositeLit:
			c.checkCompositeLit(n)

		case *ast.CallExpr:
			c.checkCallExpr(n)

		case *ast.ValueSpec:
			c.checkValueSpec(n)

		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				return
			}
			for _, lhs := range n.Lhs {
				c.checkWrite(lhs, "assign to")
			}

		case *ast.IncDecStmt:
			c.checkWrite(n.X, "modify")

		case *ast.RangeStmt:
			if n.Tok != token.ASSIGN {
				return
			}
			if n.Key != nil {
				c.checkWrite(n.Key, "assign to")
			}
			if n.Value != nil {
				c.checkWrite(n.Value, "assign to")
			}

		case *ast.UnaryExpr:
			if n.Op == token.AND {
				c.checkWrite(n.X, "take the address of")
			}

		case *ast.BinaryExpr:
			c.checkComparison(n)

		case *ast.SwitchStmt:
			c.checkSwitchStmt(n)
		}
	})

	return nil, nil
}

// ---------------------------------------------------------------------------
// Rule 1: instantiation via literal, type conversion, make, new, or var declaration
// ---------------------------------------------------------------------------

func (c *checker) checkCompositeLit(lit *ast.CompositeLit) {
	tv, ok := c.pass.TypesInfo.Types[lit]
	if !ok {
		return
	}
	obj := c.foreignRequired(tv.Type)
	if obj == nil {
		return
	}

	kind := "literal"
	if _, isStruct := types.Unalias(obj.Type().Underlying()).(*types.Struct); isStruct {
		kind = "struct literal"
	}

	c.pass.Reportf(
		lit.Pos(),
		"cannot instantiate %s.%s with %s: must be created using its constructor (marked with //constructor:required)",
		obj.Pkg().Name(),
		obj.Name(),
		kind,
	)
}

func (c *checker) checkCallExpr(call *ast.CallExpr) {
	// Case 1: Type conversion: T(x), e.g. a.MyInt(10), a.Status("active")
	if tv, ok := c.pass.TypesInfo.Types[call.Fun]; ok && tv.IsType() {
		if _, isPtr := types.Unalias(tv.Type).(*types.Pointer); !isPtr {
			if obj := c.foreignRequired(tv.Type); obj != nil {
				c.pass.Reportf(
					call.Pos(),
					"cannot instantiate %s.%s with type conversion: must be created using its constructor (marked with //constructor:required)",
					obj.Pkg().Name(),
					obj.Name(),
				)
				return
			}
		}
	}

	// Case 2: Built-ins: make, new, delete, clear, copy
	ident, ok := unwrapIdent(call.Fun)
	if !ok {
		return
	}
	builtin, ok := c.pass.TypesInfo.Uses[ident].(*types.Builtin)
	if !ok || len(call.Args) == 0 {
		return
	}

	switch builtin.Name() {
	case "make", "new":
		argTv, ok := c.pass.TypesInfo.Types[call.Args[0]]
		if !ok || !argTv.IsType() {
			return
		}
		if obj := c.foreignRequired(argTv.Type); obj != nil {
			c.pass.Reportf(
				call.Pos(),
				"cannot instantiate %s.%s with %s(): must be created using its constructor (marked with //constructor:required)",
				obj.Pkg().Name(),
				obj.Name(),
				builtin.Name(),
			)
		}

	case "delete", "clear", "copy":
		// If arg[0] is a field (e.g. delete(u.Meta, k), copy(u.Tags, x)), checkWrite catches it.
		if !c.checkWrite(call.Args[0], "modify") {
			// Otherwise, check if arg[0] itself is a constructor-required collection.
			argType := c.pass.TypesInfo.TypeOf(call.Args[0])
			if obj := c.foreignRequired(argType); obj != nil {
				c.pass.Reportf(
					call.Pos(),
					"cannot modify element of %s.%s: elements of //constructor:required types are read-only outside their package (use a method provided by the package instead)",
					obj.Pkg().Name(),
					obj.Name(),
				)
			}
		}
	}
}

func (c *checker) checkValueSpec(spec *ast.ValueSpec) {
	// Flag uninitialized variable declarations, e.g. var u user.User
	if spec.Type == nil || len(spec.Values) > 0 {
		return
	}
	t := c.pass.TypesInfo.TypeOf(spec.Type)
	if t == nil {
		return
	}
	// Pointers and interfaces default to nil, which is valid and does not instantiate a value.
	switch types.Unalias(t).Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return
	}
	if obj := c.foreignRequired(t); obj != nil {
		c.pass.Reportf(
			spec.Pos(),
			"cannot declare zero value of %s.%s: must be created using its constructor (marked with //constructor:required)",
			obj.Pkg().Name(),
			obj.Name(),
		)
	}
}

func unwrapIdent(expr ast.Expr) (*ast.Ident, bool) {
	for {
		if paren, ok := expr.(*ast.ParenExpr); ok {
			expr = paren.X
			continue
		}
		break
	}
	ident, ok := expr.(*ast.Ident)
	return ident, ok
}

// ---------------------------------------------------------------------------
// Rule 2: fields and elements are read-only outside the defining package
// ---------------------------------------------------------------------------

func (c *checker) checkWrite(target ast.Expr, verb string) bool {
	if owner, fieldName := c.findRequiredFieldOwner(target); owner != nil {
		c.pass.Reportf(
			target.Pos(),
			"cannot %s field %s of %s.%s: fields of //constructor:required types are read-only outside their package (use a method provided by the package instead)",
			verb,
			fieldName,
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	if owner := c.findRequiredElementOwner(target); owner != nil {
		c.pass.Reportf(
			target.Pos(),
			"cannot %s element of %s.%s: elements of //constructor:required types are read-only outside their package (use a method provided by the package instead)",
			verb,
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	if owner := c.findRequiredValueOwner(target); owner != nil {
		c.pass.Reportf(
			target.Pos(),
			"cannot %s value of %s.%s: //constructor:required types are read-only outside their package (use a method provided by the package instead)",
			verb,
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	return false
}

func (c *checker) findRequiredElementOwner(target ast.Expr) *types.TypeName {
	expr := target
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.SelectorExpr:
			expr = e.X
		case *ast.IndexExpr:
			if owner := c.foreignRequired(c.pass.TypesInfo.TypeOf(e.X)); owner != nil {
				return owner
			}
			expr = e.X
		default:
			return nil
		}
	}
}

func (c *checker) findRequiredValueOwner(target ast.Expr) *types.TypeName {
	expr := target
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.StarExpr:
			t := c.pass.TypesInfo.TypeOf(e.X)
			if t != nil {
				if ptr, ok := types.Unalias(t).(*types.Pointer); ok {
					if obj := c.foreignRequired(ptr.Elem()); obj != nil {
						return obj
					}
				}
			}
			expr = e.X
		default:
			return nil
		}
	}
}

func (c *checker) findRequiredFieldOwner(target ast.Expr) (*types.TypeName, string) {
	expr := target

	for {
		switch e := expr.(type) {

		case *ast.ParenExpr:
			expr = e.X

		case *ast.IndexExpr:
			t := c.pass.TypesInfo.TypeOf(e.X)
			if t == nil {
				return nil, ""
			}
			switch types.Unalias(t).Underlying().(type) {
			case *types.Array, *types.Slice, *types.Map:
				expr = e.X
			default:
				return nil, ""
			}

		case *ast.SelectorExpr:
			sel := c.pass.TypesInfo.Selections[e]
			if sel == nil || sel.Kind() != types.FieldVal {
				return nil, ""
			}

			if owner := c.requiredOwner(sel); owner != nil {
				return owner, sel.Obj().Name()
			}

			if xt := c.pass.TypesInfo.TypeOf(e.X); xt != nil {
				if _, isPtr := types.Unalias(xt).Underlying().(*types.Pointer); isPtr {
					return nil, ""
				}
			}
			expr = e.X

		default:
			return nil, ""
		}
	}
}

// ---------------------------------------------------------------------------
// Rule 3: fields, elements, methods, and types cannot be compared outside defining package
// ---------------------------------------------------------------------------

func (c *checker) checkComparisonExpr(expr ast.Expr, pos token.Pos) bool {
	if expr == nil || isNil(expr) {
		return false
	}

	// Unwrap parentheses and type conversions T(x) e.g. (string(s)) == "x"
	for {
		if p, ok := expr.(*ast.ParenExpr); ok {
			expr = p.X
			continue
		}
		call, ok := expr.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			break
		}
		if tv, ok := c.pass.TypesInfo.Types[call.Fun]; !ok || !tv.IsType() {
			break
		}
		expr = call.Args[0]
	}

	if owner, fieldName := c.findRequiredFieldOwner(expr); owner != nil {
		c.pass.Reportf(
			pos,
			"cannot compare field %s of %s.%s: fields of //constructor:required types cannot be compared outside their package (use a method provided by the package instead)",
			fieldName,
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	if owner := c.findRequiredElementOwner(expr); owner != nil {
		c.pass.Reportf(
			pos,
			"cannot compare element of %s.%s: elements of //constructor:required types cannot be compared outside their package (use a method provided by the package instead)",
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	if owner, methodName := c.findRequiredMethodOwner(expr); owner != nil {
		c.pass.Reportf(
			pos,
			"cannot compare result of method %s of %s.%s: methods of //constructor:required types cannot be used in comparisons outside their package (encapsulate domain logic in boolean methods like user.IsAdmin() instead)",
			methodName,
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	if owner := c.requiredNamedType(expr); owner != nil {
		c.pass.Reportf(
			pos,
			"cannot compare %s.%s: //constructor:required types cannot be compared directly outside their package (use a method provided by the package instead)",
			owner.Pkg().Name(),
			owner.Name(),
		)
		return true
	}
	return false
}

func (c *checker) checkComparison(bin *ast.BinaryExpr) {
	switch bin.Op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
	default:
		return
	}

	if isNil(bin.X) || isNil(bin.Y) {
		return
	}

	for _, e := range []ast.Expr{bin.X, bin.Y} {
		if c.checkComparisonExpr(e, bin.Pos()) {
			return
		}
	}
}

func (c *checker) checkSwitchStmt(sw *ast.SwitchStmt) {
	if sw.Tag == nil {
		return // switch { case cond: } -> condition checks are handled by BinaryExpr
	}
	if c.checkComparisonExpr(sw.Tag, sw.Tag.Pos()) {
		return
	}
	for _, stmt := range sw.Body.List {
		if cc, ok := stmt.(*ast.CaseClause); ok {
			for _, expr := range cc.List {
				if c.checkComparisonExpr(expr, expr.Pos()) {
					return
				}
			}
		}
	}
}

func (c *checker) findRequiredMethodOwner(target ast.Expr) (*types.TypeName, string) {
	expr := target
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.SelectorExpr:
			expr = e.X
		case *ast.CallExpr:
			selExpr, ok := e.Fun.(*ast.SelectorExpr)
			if !ok {
				return nil, ""
			}
			sel := c.pass.TypesInfo.Selections[selExpr]
			if sel == nil || sel.Kind() != types.MethodVal {
				return nil, ""
			}
			fn, ok := sel.Obj().(*types.Func)
			if !ok {
				return nil, ""
			}
			sig, ok := fn.Type().(*types.Signature)
			if !ok || sig.Recv() == nil {
				return nil, ""
			}
			if obj := c.foreignRequired(sig.Recv().Type()); obj != nil {
				return obj, fn.Name()
			}
			return nil, ""
		default:
			return nil, ""
		}
	}
}

func (c *checker) requiredNamedType(expr ast.Expr) *types.TypeName {
	if expr == nil || isNil(expr) {
		return nil
	}
	return c.foreignRequired(c.pass.TypesInfo.TypeOf(expr))
}

func isNil(expr ast.Expr) bool {
	for {
		if paren, ok := expr.(*ast.ParenExpr); ok {
			expr = paren.X
			continue
		}
		break
	}
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Helpers shared by the rules
// ---------------------------------------------------------------------------

func (c *checker) requiredOwner(sel *types.Selection) *types.TypeName {
	t := sel.Recv()
	idx := sel.Index()

	for i := 0; ; i++ {
		t = deref(t)

		if obj := c.foreignRequired(t); obj != nil {
			return obj
		}

		if i >= len(idx)-1 {
			return nil
		}

		st, ok := types.Unalias(t).Underlying().(*types.Struct)
		if !ok {
			return nil
		}
		t = st.Field(idx[i]).Type()
	}
}

// ---------------------------------------------------------------------------
// Fact export (defining-package side)
// ---------------------------------------------------------------------------

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
	st, ok := types.Unalias(obj.Type().Underlying()).(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		tag := reflect.StructTag(st.Tag(i))
		if tag.Get("constructor") == "required" {
			return true
		}
	}
	return false
}

func scanPackageRequiredTypes(files []*ast.File) map[string]bool {
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
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if hasDeclComment ||
						hasConstructorDirective(typeSpec.Doc) ||
						hasConstructorDirective(typeSpec.Comment) {
						mark(typeSpec.Name.Name)
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
	for _, field := range fnType.Results.List {
		t := field.Type
		if star, ok := t.(*ast.StarExpr); ok {
			t = star.X
		}
		if idxExpr, ok := t.(*ast.IndexExpr); ok {
			t = idxExpr.X
		}
		if idxList, ok := t.(*ast.IndexListExpr); ok {
			t = idxList.X
		}
		if ident, ok := t.(*ast.Ident); ok {
			if ident.Name == "error" {
				continue
			}
			return []string{ident.Name}
		}
		break
	}
	return nil
}

func hasConstructorDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		text := c.Text
		if strings.HasPrefix(text, "//") {
			line := strings.TrimPrefix(text, "//")
			// Go directives must not have whitespace between // and the directive name
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			line = strings.TrimSpace(line)
			if rem, ok := strings.CutPrefix(line, "constructor:required"); ok {
				if len(rem) == 0 || rem[0] == ' ' || rem[0] == '\t' || rem[0] == '(' {
					return true
				}
			}
		} else if strings.HasPrefix(text, "/*") {
			line := strings.TrimPrefix(text, "/*")
			line = strings.TrimSuffix(line, "*/")
			line = strings.TrimSpace(line)
			if rem, ok := strings.CutPrefix(line, "constructor:required"); ok {
				if len(rem) == 0 || rem[0] == ' ' || rem[0] == '\t' || rem[0] == '(' {
					return true
				}
			}
		}
	}
	return false
}
