package a

// DirectCommentStruct requires constructor via direct comment.
//constructor:required
type DirectCommentStruct struct { // want DirectCommentStruct:`constructor:required`
	Value   int
	Arr     [2]int
	Slice   []int
	Map     map[string]int
	Address Address
}

type Address struct {
	City string
}

func NewDirectCommentStruct(v int) DirectCommentStruct {
	return DirectCommentStruct{
		Value:   v,
		Arr:     [2]int{v, v},
		Slice:   []int{v},
		Map:     map[string]int{"k": v},
		Address: Address{City: "default"},
	} // OK: defining package is exempt
}

func MutateDirectCommentStruct(d *DirectCommentStruct) {
	// Defining package is exempt from field mutations:
	d.Value = 42
	d.Value++
	d.Value--
	d.Arr[0] = 1
	_ = &d.Value
	_ = &d.Arr[0]
	for d.Value = range []int{1} {
	}
	for _, d.Value = range []int{1} {
	}
}

type TagStruct struct { // want TagStruct:`constructor:required`
	Name string `constructor:"required"`
}

func NewTagStruct(name string) *TagStruct {
	return &TagStruct{Name: name} // OK: defining package is exempt
}

type FuncCommentStruct struct { // want FuncCommentStruct:`constructor:required`
	Secret string
}

// NewFuncCommentStruct creates a FuncCommentStruct.
//constructor:required
func NewFuncCommentStruct(secret string) *FuncCommentStruct {
	return &FuncCommentStruct{Secret: secret} // OK: defining package is exempt
}

type UnrestrictedStruct struct {
	Count int
	Arr   [2]int
}

func NewUnrestrictedStruct(c int) UnrestrictedStruct {
	return UnrestrictedStruct{Count: c}
}

