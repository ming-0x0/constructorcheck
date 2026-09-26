package b

import "a"

func testCases() {
	// DirectCommentStruct:
	_ = a.NewDirectCommentStruct(1)              // OK
	_ = a.DirectCommentStruct{Value: 1}          // want `cannot instantiate a\.DirectCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = &a.DirectCommentStruct{Value: 1}         // want `cannot instantiate a\.DirectCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`

	// TagStruct:
	_ = a.NewTagStruct("test")                   // OK
	_ = a.TagStruct{Name: "test"}                // want `cannot instantiate a\.TagStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = &a.TagStruct{Name: "test"}               // want `cannot instantiate a\.TagStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`

	// FuncCommentStruct:
	_ = a.NewFuncCommentStruct("secret")         // OK
	_ = a.FuncCommentStruct{Secret: "secret"}    // want `cannot instantiate a\.FuncCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = &a.FuncCommentStruct{Secret: "secret"}   // want `cannot instantiate a\.FuncCommentStruct with struct literal: must be created using its constructor \(marked with //constructor:required\)`

	// UnrestrictedStruct:
	_ = a.NewUnrestrictedStruct(42)              // OK
	_ = a.UnrestrictedStruct{Count: 42}          // OK
	_ = &a.UnrestrictedStruct{Count: 42}         // OK
}
