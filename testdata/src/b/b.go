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

	// Slices and maps mutate elements of struct fields and are prohibited outside defining package:
	d.Slice[0] = 42               // want `cannot assign to field Slice of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	d.Slice[0]++                  // want `cannot modify field Slice of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = &d.Slice[0]               // want `cannot take the address of field Slice of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	d.Map["key"] = 42             // want `cannot assign to field Map of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
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

func testComparisons() {
	d := a.NewDirectCommentStruct(1)
	d2 := a.NewDirectCommentStruct(2)

	// Comparing struct fields with values:
	_ = d.Value == 2              // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	_ = 2 == d.Value              // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	_ = d.Value != 2              // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	_ = d.Value < 2               // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	_ = d.Value <= 2              // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	_ = d.Value > 2               // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	_ = d.Value >= 2              // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`

	// Comparing fields between structs:
	_ = d.Value == d2.Value       // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`

	// Comparing nested fields:
	_ = d.Address.City == "Hanoi" // want `cannot compare field Address of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`

	// Comparing inline array elements:
	_ = d.Arr[0] == 1             // want `cannot compare field Arr of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`

	// Comparing struct instances directly (values):
	c1 := a.NewComparableStruct(1, "one")
	c2 := a.NewComparableStruct(2, "two")
	_ = c1 == c2                  // want `cannot compare a\.ComparableStruct: //constructor:required types cannot be compared directly outside their package`
	_ = c1 != c2                  // want `cannot compare a\.ComparableStruct: //constructor:required types cannot be compared directly outside their package`

	// Comparing pointer instances:
	t1 := a.NewTagStruct("test")
	t2 := a.NewTagStruct("test")
	_ = t1 == t2                  // want `cannot compare a\.TagStruct: //constructor:required types cannot be compared directly outside their package`
	_ = t1 != t2                  // want `cannot compare a\.TagStruct: //constructor:required types cannot be compared directly outside their package`

	// Dereferenced pointer comparison (struct value):
	_ = *t1 == *t2                // want `cannot compare a\.TagStruct: //constructor:required types cannot be compared directly outside their package`

	// Nil comparisons are PERMITTED:
	_ = t1 == nil                 // OK
	_ = nil == t1                 // OK
	_ = t1 != nil                 // OK
	_ = nil != t1                 // OK

	// Embedded struct field comparison:
	ed := EmbeddedDirect{
		DirectCommentStruct: a.NewDirectCommentStruct(1),
	}
	_ = ed.Value == 10            // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`

	// Unrestricted struct comparisons are PERMITTED:
	u1 := a.NewUnrestrictedStruct(42)
	u2 := a.NewUnrestrictedStruct(42)
	_ = u1 == u2                  // OK
	_ = u1.Count == 42            // OK
	_ = u1.Count < 100            // OK

	// Method call comparisons are BANNED:
	_ = d.GetValue() == 1       // want `cannot compare result of method GetValue of a\.DirectCommentStruct: methods of //constructor:required types cannot be used in comparisons outside their package \(encapsulate domain logic in boolean methods like user\.IsAdmin\(\) instead\)`
	_ = 1 == d.GetValue()       // want `cannot compare result of method GetValue of a\.DirectCommentStruct: methods of //constructor:required types cannot be used in comparisons outside their package \(encapsulate domain logic in boolean methods like user\.IsAdmin\(\) instead\)`
	_ = t1.GetName() == "admin" // want `cannot compare result of method GetName of a\.TagStruct: methods of //constructor:required types cannot be used in comparisons outside their package \(encapsulate domain logic in boolean methods like user\.IsAdmin\(\) instead\)`
	_ = "admin" == t1.GetName() // want `cannot compare result of method GetName of a\.TagStruct: methods of //constructor:required types cannot be used in comparisons outside their package \(encapsulate domain logic in boolean methods like user\.IsAdmin\(\) instead\)`

	// Boolean methods used as direct conditions are PERMITTED:
	if d.IsPositive() { // OK
	}
	if !t1.IsAdmin() { // OK
	}
}

func testNonStructInstantiation() {
	// Type conversion:
	_ = a.NewStatus("active")      // OK
	_ = a.Status("active")         // want `cannot instantiate a\.Status with type conversion: must be created using its constructor \(marked with //constructor:required\)`
	_ = a.NewMyInt(10)             // OK
	_ = a.MyInt(10)                // want `cannot instantiate a\.MyInt with type conversion: must be created using its constructor \(marked with //constructor:required\)`

	// Literals:
	_ = a.NewMySlice(1, 2)         // OK
	_ = a.MySlice{1, 2}            // want `cannot instantiate a\.MySlice with literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = a.NewMyMap()               // OK
	_ = a.MyMap{"k": 1}            // want `cannot instantiate a\.MyMap with literal: must be created using its constructor \(marked with //constructor:required\)`
	_ = a.NewMyArray(1, 2)         // OK
	_ = a.MyArray{1, 2}            // want `cannot instantiate a\.MyArray with literal: must be created using its constructor \(marked with //constructor:required\)`

	// make() and new():
	_ = make(a.MySlice, 0)         // want `cannot instantiate a\.MySlice with make\(\): must be created using its constructor \(marked with //constructor:required\)`
	_ = make(a.MyMap)              // want `cannot instantiate a\.MyMap with make\(\): must be created using its constructor \(marked with //constructor:required\)`
	_ = new(a.MyInt)               // want `cannot instantiate a\.MyInt with new\(\): must be created using its constructor \(marked with //constructor:required\)`
	_ = new(a.DirectCommentStruct) // want `cannot instantiate a\.DirectCommentStruct with new\(\): must be created using its constructor \(marked with //constructor:required\)`
}

func testNonStructWrites() {
	s := a.NewMySlice(1, 2)
	s[0] = 10                      // want `cannot assign to element of a\.MySlice: elements of //constructor:required types are read-only outside their package`
	s[0]++                         // want `cannot modify element of a\.MySlice: elements of //constructor:required types are read-only outside their package`
	_ = &s[0]                      // want `cannot take the address of element of a\.MySlice: elements of //constructor:required types are read-only outside their package`

	m := a.NewMyMap()
	m["k"] = 10                    // want `cannot assign to element of a\.MyMap: elements of //constructor:required types are read-only outside their package`

	arr := a.NewMyArray(1, 2)
	arr[0] = 10                    // want `cannot assign to element of a\.MyArray: elements of //constructor:required types are read-only outside their package`
	arr[0]--                       // want `cannot modify element of a\.MyArray: elements of //constructor:required types are read-only outside their package`
	_ = &arr[0]                    // want `cannot take the address of element of a\.MyArray: elements of //constructor:required types are read-only outside their package`

	num := a.NewMyInt(10)
	pNum := &num
	*pNum = 20                     // want `cannot assign to value of a\.MyInt: //constructor:required types are read-only outside their package`
	(*pNum)++                      // want `cannot modify value of a\.MyInt: //constructor:required types are read-only outside their package`
	_ = &*pNum                     // want `cannot take the address of value of a\.MyInt: //constructor:required types are read-only outside their package`
}

func testNonStructComparisons() {
	st1 := a.NewStatus("active")
	st2 := a.NewStatus("inactive")
	_ = st1 == "active"            // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	_ = "active" == st1            // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	_ = st1 != st2                 // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	_ = st1.Raw() == "active"      // want `cannot compare result of method Raw of a\.Status: methods of //constructor:required types cannot be used in comparisons outside their package \(encapsulate domain logic in boolean methods like user\.IsAdmin\(\) instead\)`

	if st1.IsActive() {            // OK
	}

	num1 := a.NewMyInt(1)
	num2 := a.NewMyInt(2)
	_ = num1 == num2               // want `cannot compare a\.MyInt: //constructor:required types cannot be compared directly outside their package`
	_ = num1 < 2                   // want `cannot compare a\.MyInt: //constructor:required types cannot be compared directly outside their package`
	_ = 2 >= num1                  // want `cannot compare a\.MyInt: //constructor:required types cannot be compared directly outside their package`
	if num1.IsZero() {             // OK
	}

	slice := a.NewMySlice(1)
	_ = slice[0] == 1              // want `cannot compare element of a\.MySlice: elements of //constructor:required types cannot be compared outside their package`
	_ = slice == nil               // OK
	_ = nil != slice               // OK

	arr := a.NewMyArray(1, 2)
	_ = arr[0] == 1                // want `cannot compare element of a\.MyArray: elements of //constructor:required types cannot be compared outside their package`

	pNum := &num1
	_ = pNum == nil                // OK
}

func testSwitchStatements() {
	d := a.NewDirectCommentStruct(1)
	switch d.Value { // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
	case 1:
	}

	st := a.NewStatus("active")
	switch st { // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	case "active":
	}

	x := a.NewStatus("inactive")
	switch x { // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	case "inactive":
	}

	var anyVal any
	switch anyVal {
	case st: // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	}

	slice := a.NewMySlice(1)
	switch slice[0] { // want `cannot compare element of a\.MySlice: elements of //constructor:required types cannot be compared outside their package`
	case 1:
	}

	switch d.GetValue() { // want `cannot compare result of method GetValue of a\.DirectCommentStruct: methods of //constructor:required types cannot be used in comparisons outside their package`
	case 1:
	}

	// Untagged switch with method calls in case condition must be PERMITTED:
	switch {
	case d.IsPositive(): // OK
	case st.IsActive():  // OK
	}
}

func testBuiltinMutations() {
	slice := a.NewMySlice(1, 2)
	deleteTarget := a.NewMyMap()

	delete(deleteTarget, "k")  // want `cannot modify element of a\.MyMap: elements of //constructor:required types are read-only outside their package`
	clear(slice)               // want `cannot modify element of a\.MySlice: elements of //constructor:required types are read-only outside their package`
	copy(slice, []int{10, 20}) // want `cannot modify element of a\.MySlice: elements of //constructor:required types are read-only outside their package`

	d := a.NewDirectCommentStruct(1)
	delete(d.Map, "k")       // want `cannot modify field Map of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	copy(d.Slice, []int{10}) // want `cannot modify field Slice of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
}

func testSliceOfStruct() {
	ss := a.NewStructSlice()
	ss[0].Value = 42      // want `cannot assign to field Value of a\.DirectCommentStruct: fields of //constructor:required types are read-only outside their package`
	_ = ss[0].Value == 42 // want `cannot compare field Value of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
}

func testTagsAndDirectivesTightMatching() {
	// FalseTagStruct has no valid constructor tag:
	_ = a.FalseTagStruct{ID: 10} // OK

	// FalseDirectiveStruct has constructor:requiredFoo which should be ignored:
	_ = a.FalseDirectiveStruct{ID: 10} // OK

	// PrimaryType is marked via constructor, but SecondaryType is not:
	_ = a.SecondaryType{Data: "allowed"} // OK
	_ = a.PrimaryType{Count: 1}          // want `cannot instantiate a\.PrimaryType with struct literal: must be created using its constructor \(marked with //constructor:required\)`
}

func testTypeAliases() {
	_ = a.StatusAlias("active") // want `cannot instantiate a\.Status with type conversion: must be created using its constructor \(marked with //constructor:required\)`
	sa := a.NewStatus("active")
	_ = sa == "active" // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
}

func testVarZeroValue() {
	var u a.DirectCommentStruct // want `cannot declare zero value of a\.DirectCommentStruct: must be created using its constructor \(marked with //constructor:required\)`
	_ = u
	var s a.Status // want `cannot declare zero value of a\.Status: must be created using its constructor \(marked with //constructor:required\)`
	_ = s
	var p *a.DirectCommentStruct // OK: pointer is nil
	_ = p
	var ps *a.Status // OK: pointer is nil
	_ = ps
	var r a.Repo // OK: interface is nil
	_ = r
	var initialized = a.NewDirectCommentStruct(1) // OK
	_ = initialized
}

func testComparisonTypeConversions() {
	st := a.NewStatus("active")
	_ = string(st) == "active" // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	_ = "active" == string(st) // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`

	num := a.NewMyInt(1)
	_ = int(num) > 0 // want `cannot compare a\.MyInt: //constructor:required types cannot be compared directly outside their package`

	// Parentheses around type conversions:
	_ = (string(st)) == "active"       // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`
	_ = ((string((st)))) == "active"   // want `cannot compare a\.Status: //constructor:required types cannot be compared directly outside their package`

	d := a.NewDirectCommentStruct(1)
	_ = string(d.Address.City) == "Hanoi" // want `cannot compare field Address of a\.DirectCommentStruct: fields of //constructor:required types cannot be compared outside their package`
}

func testPointerConversions() {
	d := a.NewDirectCommentStruct(1)
	dp := &d
	// Pointer type conversion (*T)(p) does not instantiate a new value, allowed:
	_ = (*a.DirectCommentStruct)(dp) // OK
}



