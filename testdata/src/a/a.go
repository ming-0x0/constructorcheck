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

// Status represents user status as a defined string.
//constructor:required
type Status string // want Status:`constructor:required`

func NewStatus(s string) Status {
	return Status(s) // OK: defining package is exempt
}

func (s Status) Raw() string {
	return string(s)
}

func (s Status) IsActive() bool {
	return s == "active"
}

// MyInt represents a custom integer.
//constructor:required
type MyInt int // want MyInt:`constructor:required`

func NewMyInt(v int) MyInt {
	return MyInt(v) // OK: defining package is exempt
}

func (m MyInt) IsZero() bool {
	return m == 0
}

// MySlice represents a slice of ints.
//constructor:required
type MySlice []int // want MySlice:`constructor:required`

func NewMySlice(items ...int) MySlice {
	return MySlice(items) // OK: defining package is exempt
}

// MyMap represents a map of string to int.
//constructor:required
type MyMap map[string]int // want MyMap:`constructor:required`

func NewMyMap() MyMap {
	return make(MyMap) // OK: defining package is exempt
}

// MyArray represents a fixed array of ints.
//constructor:required
type MyArray [2]int // want MyArray:`constructor:required`

func NewMyArray(a, b int) MyArray {
	return MyArray{a, b} // OK: defining package is exempt
}

// StructSlice represents a slice of structs.
//constructor:required
type StructSlice []DirectCommentStruct // want StructSlice:`constructor:required`

func NewStructSlice() StructSlice {
	return StructSlice{{Value: 1}}
}

// StatusAlias aliases Status.
type StatusAlias = Status

// FalseTagStruct has misleading tags that must not trigger requiredFact.
type FalseTagStruct struct {
	ID int `xconstructor:"required" other:"constructor:required"`
}

// FalseDirectiveStruct has a comment starting with constructor:requiredFoo which must not match.
//constructor:requiredFoo
type FalseDirectiveStruct struct {
	ID int
}

// SecondaryType is returned as a second value from a constructor and should not be marked.
type SecondaryType struct {
	Data string
}

// PrimaryType is the primary return type from constructor.
type PrimaryType struct { // want PrimaryType:`constructor:required`
	Count int
}

// NewWithSecondary returns PrimaryType, SecondaryType, and error.
//constructor:required
func NewWithSecondary() (*PrimaryType, *SecondaryType, error) {
	return &PrimaryType{Count: 1}, &SecondaryType{Data: "meta"}, nil
}

// Repo is a constructor-required interface.
//constructor:required
type Repo interface { // want Repo:`constructor:required`
	Get(id int) string
}

type repoImpl struct{}

func (repoImpl) Get(id int) string { return "ok" }

func NewRepo() Repo {
	return repoImpl{}
}

