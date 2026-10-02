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

func CompareDirectCommentStruct(d1, d2 DirectCommentStruct) bool {
	// Defining package is exempt from comparison restrictions:
	_ = d1.Value == 42
	_ = d1.Value < 100
	_ = d1.Address.City == "default"
	return d1.Value == d2.Value
}

// ComparableStruct is comparable since its fields are comparable types.
//constructor:required
type ComparableStruct struct { // want ComparableStruct:`constructor:required`
	ID   int
	Name string
}

func NewComparableStruct(id int, name string) ComparableStruct {
	return ComparableStruct{ID: id, Name: name}
}

func CompareComparableStruct(c1, c2 ComparableStruct) bool {
	return c1 == c2 || c1 != c2
}

func (d DirectCommentStruct) GetValue() int {
	return d.Value
}

func (d DirectCommentStruct) IsPositive() bool {
	return d.Value > 0
}

func (t *TagStruct) GetName() string {
	return t.Name
}

func (t *TagStruct) IsAdmin() bool {
	return t.Name == "admin"
}



