# constructorcheck

`constructorcheck` is a Go linter (compatible with `go/analysis` and `golangci-lint`) that enforces **constructor-based instantiation**, **strict immutability**, and **domain encapsulation** across packages.

Once a type is marked as requiring a constructor (via directive or struct tag), external packages are restricted by three core rules:
1. **No direct instantiation**: Must be constructed using its package's constructor functions (no literals, conversions, `make`, `new`, or zero-value `var`).
2. **Read-only outside defining package**: Fields, elements, and values cannot be modified or mutated (no field writes, indexing writes, or mutating builtins like `delete`, `clear`, `copy`).
3. **No direct comparisons**: Types, fields, elements, and getter results cannot be compared with `==`, `!=`, `<`, `>`, or tagged `switch`. Domain logic must be encapsulated in boolean methods (e.g. `user.IsAdmin()`, `status.IsActive()`).

---

## The Three Core Rules

### Rule 1: Instantiation Restrictions
Outside its defining package, a constructor-required type cannot be instantiated arbitrarily:

| Syntax | Status | Details |
| :--- | :--- | :--- |
| `pkg.NewUser(...)`, `pkg.NewStatus(...)` | **Allowed** | Valid constructor call |
| `pkg.User{ID: 1}` | **Forbidden** | Struct literal |
| `pkg.MySlice{1, 2}`, `pkg.MyMap{"k": 1}` | **Forbidden** | Slice/Map/Array literal |
| `pkg.Status("active")`, `pkg.MyInt(10)` | **Forbidden** | Defined type conversion |
| `make(pkg.MySlice, 0)`, `make(pkg.MyMap)` | **Forbidden** | `make()` built-in |
| `new(pkg.User)`, `new(pkg.MyInt)` | **Forbidden** | `new()` built-in |
| `var u pkg.User`, `var s pkg.Status` | **Forbidden** | Zero-value value declaration |
| `var p *pkg.User`, `var r pkg.Repo` | **Allowed** | Nil pointer or interface declaration |
| `(*pkg.User)(ptr)` | **Allowed** | Pointer cast (reinterprets existing pointer) |

---

### Rule 2: Immutability / Mutation Restrictions
Values and fields of constructor-required types are completely read-only outside their defining package:

| Syntax | Status | Details |
| :--- | :--- | :--- |
| `_ = u.ID`, `_ = u.Slice[0]`, `_ = u.Map["k"]` | **Allowed** | Read-only field & element access |
| `u.ID = 10`, `u.ID++`, `u.ID += 5` | **Forbidden** | Direct field mutation |
| `u.Arr[0] = 1`, `u.Address.City = "..."` | **Forbidden** | Inline array & nested field mutation |
| `u.Slice[0] = 1`, `u.Map["k"] = 1` | **Forbidden** | Slice / Map field element mutation |
| `s[0] = 1`, `m["k"] = 1`, `*pNum = 20` | **Forbidden** | Defined slice, map, array, or dereference mutation |
| `&u.ID`, `&s[0]`, `&u.Arr[0]` | **Forbidden** | Taking address of fields/elements (prevents mutation via pointer) |
| `&u` | **Allowed** | Taking address of struct itself |
| `for u.ID = range ...` | **Forbidden** | Mutation as range loop target |
| `delete(u.Map, k)`, `delete(m, k)` | **Forbidden** | Built-in `delete()` |
| `clear(u.Slice)`, `clear(s)` | **Forbidden** | Built-in `clear()` |
| `copy(u.Slice, ...)`, `copy(s, ...)` | **Forbidden** | Built-in `copy()` destination |

---

### Rule 3: Domain Encapsulation / Comparison Restrictions
To maintain domain invariant encapsulation, external packages cannot compare constructor-required types, their internal fields, or getter outputs:

| Syntax | Status | Details |
| :--- | :--- | :--- |
| `if status.IsActive() { ... }` | **Allowed** | Encapsulated boolean domain method |
| `if u.IsAdmin() { ... }` | **Allowed** | Encapsulated boolean domain method |
| `p == nil`, `slice == nil` | **Allowed** | Nil checks |
| `switch { case u.IsActive(): }` | **Allowed** | Untagged boolean condition switch |
| `u1 == u2`, `u1 != u2` | **Forbidden** | Direct struct/value comparison |
| `status == "active"`, `num > 0` | **Forbidden** | Direct scalar comparison |
| `string(status) == "active"`, `((int(num))) > 0` | **Forbidden** | Type conversion bypass attempts |
| `u.ID == 10`, `u.Address.City == "..."` | **Forbidden** | Direct field comparison |
| `s[0] == 1`, `u.Arr[0] == 1` | **Forbidden** | Collection element comparison |
| `u.GetID() == 10`, `status.Raw() == "active"` | **Forbidden** | Getter return comparison (encapsulate into boolean methods) |
| `switch u.ID { case ... }`, `switch status { ... }` | **Forbidden** | Tagged switch comparison |

---

## Marking Mechanisms

You can mark a type in any of three ways using the strict Go directive `//constructor:required` (no space after `//`):

### 1. Directive on Type Declaration
Works on any defined type (structs, primitives, slices, maps, arrays, interfaces):
```go
package user

// User represents a user account.
//constructor:required
type User struct {
    ID   string
    Name string
}

// Status represents a custom scalar type.
//constructor:required
type Status string

// Tokens represents a slice collection.
//constructor:required
type Tokens []string
```

### 2. Directive on Constructor Function
Annotate the constructor function. `constructorcheck` inspects the primary return type and automatically marks it:
```go
package config

type Config struct {
    Port int
}

// NewConfig validates and initializes Config.
//constructor:required
func NewConfig(port int) (*Config, error) {
    return &Config{Port: port}, nil
}
```
*(Supports generic constructors such as `func NewBox[T any](v T) *Box[T]` as well).*

### 3. Struct Field Tag
Add `constructor:"required"` to one or more struct fields:
```go
package client

type Client struct {
    endpoint string `constructor:"required"`
}

func NewClient(endpoint string) *Client {
    return &Client{endpoint: endpoint}
}
```

---

## Defining Package Exemption

The package that defines the type (and its internal test files `_test.go`) is **exempt** from all three rules:
- Constructors can initialize the type using literals or `make()`.
- Mutation methods can freely modify fields, slice items, and map keys.
- Domain methods can inspect and compare fields directly to implement boolean helper methods.

Generated files (matching `^// Code generated .* DO NOT EDIT\.$`) are automatically skipped.

---

## Examples

### Package Definition (`user`)
```go
package user

//constructor:required
type Status string

const (
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
)

func NewStatus(s string) Status {
    return Status(s) // Allowed: defining package
}

func (s Status) IsActive() bool {
    return s == StatusActive // Allowed: defining package
}

//constructor:required
type User struct {
    ID    int
    Name  string
    Roles []string
}

func NewUser(id int, name string) *User {
    return &User{
        ID:    id,
        Name:  name,
        Roles: []string{"viewer"},
    } // Allowed: defining package
}

func (u *User) IsAdmin() bool {
    for _, r := range u.Roles {
        if r == "admin" {
            return true
        }
    }
    return false
}
```

### External Consumer Code
```go
package main

import "example.com/user"

func main() {
    // ----------------- Rule 1: Instantiation -----------------
    u := user.NewUser(1, "Alice")             // OK: constructor
    st := user.NewStatus("active")           // OK: constructor
    var p *user.User                         // OK: nil pointer

    _ = user.User{ID: 2}                     // LINTER ERROR: struct literal
    _ = user.Status("active")                // LINTER ERROR: type conversion
    _ = new(user.User)                       // LINTER ERROR: new()
    var bad user.User                        // LINTER ERROR: zero-value declaration

    // ----------------- Rule 2: Immutability ------------------
    _ = u.ID                                 // OK: read access
    u.Name = "Bob"                           // LINTER ERROR: field is read-only
    u.Roles[0] = "admin"                     // LINTER ERROR: element is read-only
    copy(u.Roles, []string{"super"})         // LINTER ERROR: mutating builtin copy()

    // ----------------- Rule 3: Encapsulation -----------------
    if st.IsActive() {                       // OK: boolean domain method
    }
    if u.IsAdmin() {                         // OK: boolean domain method
    }

    _ = st == "active"                       // LINTER ERROR: direct comparison
    _ = string(st) == "active"               // LINTER ERROR: direct comparison
    _ = u.ID == 1                            // LINTER ERROR: field comparison
    switch u.ID {                            // LINTER ERROR: tagged switch
    case 1:
    }
}
```

---

## Installation & Usage

### 1. Standalone CLI

You can run `constructorcheck` directly with `singlechecker`:

```bash
go run ./cmd/constructorcheck ./...
```

Or install the standalone binary:
```bash
go install github.com/ming-0x0/constructorcheck/cmd/constructorcheck@latest
constructorcheck ./...
```

### 2. GolangCI-Lint Module Plugin

`constructorcheck` provides a module plugin under package `github.com/ming-0x0/constructorcheck/plugin` compatible with `golangci-lint custom`.

Create a `.custom-gcl.yml` file:
```yaml
version: v1.64.0
plugins:
  - module: 'github.com/ming-0x0/constructorcheck'
    import: 'github.com/ming-0x0/constructorcheck/plugin'
    version: latest
```

Build your custom `golangci-lint` binary:
```bash
golangci-lint custom
```

Enable `constructorcheck` in your `.golangci.yml`:
```yaml
linters:
  enable:
    - constructorcheck
```

---

## Testing

Run tests across the entire repository:
```bash
go test -v ./...
```

---

## License

[MIT](file:///home/ming-x/code/constructorcheck/LICENSE)
