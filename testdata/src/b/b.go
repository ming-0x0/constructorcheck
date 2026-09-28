package b

import "a"

func testStructLiterals() {
	// DirectCommentStruct:
	_ = a.NewDirectCommentStruct(1)            // OK
	_ = a.DirectCommentStruct{Value: 1}        // want `cannot instantiate a\.DirectCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = &a.DirectCommentStruct{Value: 1}       // want `cannot instantiate a\.DirectCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`

	// TagStruct:
	_ = a.NewTagStruct("test")                 // OK
	_ = a.TagStruct{Name: "test"}              // want `cannot instantiate a\.TagStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = &a.TagStruct{Name: "test"}             // want `cannot instantiate a\.TagStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`

	// FuncCommentStruct:
	_ = a.NewFuncCommentStruct("secret")       // OK
	_ = a.FuncCommentStruct{Secret: "secret"}  // want `cannot instantiate a\.FuncCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = &a.FuncCommentStruct{Secret: "secret"} // want `cannot instantiate a\.FuncCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`

	// UnrestrictedStruct:
	_ = a.NewUnrestrictedStruct(42)            // OK
	_ = a.UnrestrictedStruct{Count: 42}        // OK
	_ = &a.UnrestrictedStruct{Count: 42}       // OK
}

func testFieldAssignments() {
	d := a.NewDirectCommentStruct(1)

	// Direct assignments:
	d.Value = 10                  // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	(d.Value) = 10                // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	d.Value += 5                  // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`

	// Via pointer receiver:
	dp := &d
	dp.Value = 20                 // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`

	// Multi-value assignment:
	var x int
	d.Value, x = 1, 2             // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = x

	// Definition with := does not modify existing struct field:
	val := d.Value                // OK
	_ = val

	// Other constructor-required types:
	t := a.NewTagStruct("test")
	t.Name = "updated"            // want `cannot assign to field Name of a\.TagStruct: fields of //constructor:required types are read-only outside their package`

	f := a.NewFuncCommentStruct("secret")
	f.Secret = "exposed"          // want `cannot assign to field Secret of a\.FuncCommentStruct: fields of //constructor:required types are read-only outside their package`
}

func testFieldIncDec() {
	d := a.NewDirectCommentStruct(1)
	d.Value++                     // want `cannot modify field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	d.Value--                     // want `cannot modify field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`

	dp := &d
	dp.Value++                    // want `cannot modify field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
}

func testFieldTakeAddress() {
	d := a.NewDirectCommentStruct(1)
	_ = &d.Value                  // want `cannot take the address of field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = &(d.Value)                // want `cannot take the address of field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`

	// Taking pointer to the struct itself is allowed:
	_ = &d                        // OK

	t := a.NewTagStruct("test")
	_ = &t.Name                   // want `cannot take the address of field Name of a\.TagStruct: fields of //constructor:required types are read-only outside their package`

	f := a.NewFuncCommentStruct("secret")
	_ = &f.Secret                 // want `cannot take the address of field Secret of a\.FuncCommentStruct: fields of //constructor:required types are read-only outside their package`
}

func testIndexExpressions() {
	d := a.NewDirectCommentStruct(1)

	// Arrays stored inline: modifying array element mutates struct memory directly
	d.Arr[0] = 42                 // want `cannot assign to field Arr of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	d.Arr[0]++                    // want `cannot modify field Arr of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = &d.Arr[0]                 // want `cannot take the address of field Arr of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`

	// Slices and maps mutate shared backing storage, not the struct field itself: deliberately not flagged
	d.Slice[0] = 42               // OK
	d.Slice[0]++                  // OK
	_ = &d.Slice[0]               // OK
	d.Map["key"] = 42             // OK
}

func testRangeLoops() {
	d := a.NewDirectCommentStruct(1)

	for d.Value = range []int{1, 2} { // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	}
	for _, d.Value = range []int{1, 2} { // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	}
	var dummy int
	for d.Value, dummy = range []int{1, 2} { // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	}
	_ = dummy

	// Range with define := is not flagged:
	for _, v := range []int{1, 2} {
		_ = v
	}
}

func testNestedStruct() {
	d := a.NewDirectCommentStruct(1)
	d.Address.City = "Hanoi"      // want `cannot assign to field Address of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = &d.Address.City           // want `cannot take the address of field Address of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
}

type EmbeddedDirect struct {
	a.DirectCommentStruct
}

func testStructEmbedding() {
	ed := EmbeddedDirect{
		DirectCommentStruct: a.NewDirectCommentStruct(1),
	}

	ed.Value = 10                 // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	ed.Value++                    // want `cannot modify field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = &ed.Value                 // want `cannot take the address of field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	ed.Arr[0] = 5                 // want `cannot assign to field Arr of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
}

func testUnrestrictedStruct() {
	u := a.NewUnrestrictedStruct(42)
	u.Count = 100                 // OK
	u.Count++                     // OK
	u.Count--                     // OK
	_ = &u.Count                  // OK
	u.Arr[0] = 5                  // OK
	_ = &u.Arr[0]                 // OK

	for u.Count = range []int{1} { // OK
	}
	for _, u.Count = range []int{1} { // OK
	}
}

func testReadOnlyAccess() {
	d := a.NewDirectCommentStruct(1)
	_ = d.Value                   // OK: read access is permitted
	_ = d.Arr[0]                  // OK
	_ = d.Slice[0]                // OK
	_ = d.Map["k"]                // OK
	_ = d.Address.City            // OK

	t := a.NewTagStruct("test")
	_ = t.Name                    // OK

	f := a.NewFuncCommentStruct("secret")
	_ = f.Secret                  // OK
}
