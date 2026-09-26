package a

// DirectCommentStruct requires constructor via direct comment.
//constructor:required
type DirectCommentStruct struct { // want DirectCommentStruct:`constructor:required`
	Value int
}

func NewDirectCommentStruct(v int) DirectCommentStruct {
	return DirectCommentStruct{Value: v} // OK: defining package is exempt
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
}

func NewUnrestrictedStruct(c int) UnrestrictedStruct {
	return UnrestrictedStruct{Count: c}
}
