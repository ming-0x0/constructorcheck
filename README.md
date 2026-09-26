# constructorcheck

`constructorcheck` is a Go linter (compatible with `go/analysis` and `golangci-lint`) that enforces struct instantiation strictly through constructor functions.

If a struct type or constructor is annotated with `//constructor:required` or has a struct tag `constructor:"required"`, external packages are prohibited from instantiating it using struct literals (composite literals like `pkg.MyStruct{...}` or `&pkg.MyStruct{...}`).

---

## Features

- **Multiple marking mechanisms**:
  - **Type doc-comment**: Annotate struct types directly with `//constructor:required`.
  - **Constructor doc-comment**: Annotate the constructor function (e.g. `New...`) with `//constructor:required`.
  - **Struct field tag**: Mark fields with `constructor:"required"`.
- **Package exemption**: The defining package is permitted to instantiate the struct literal (allowing constructor functions to build and return instances).
- **Cross-package analysis via Facts**: Uses `go/analysis.Fact` to propagate required constructor metadata across packages and module boundaries efficiently.
- **GolangCI-Lint Module Plugin support**: Plugs into `golangci-lint` as a custom module plugin.

---

## How It Works

### 1. Marking a Struct as Constructor-Required

You can mark a struct in any of the following three ways:

#### Option A: Directive on Struct Type
```go
package user

// User holds user profile information.
//constructor:required
type User struct {
    ID   string
    Name string
}

func NewUser(id, name string) *User {
    return &User{ID: id, Name: name} // Allowed in defining package
}
```

#### Option B: Directive on Constructor Function
```go
package config

type Config struct {
    Port int
}

// NewConfig validates and initializes Config.
//constructor:required
func NewConfig(port int) *Config {
    return &Config{Port: port} // Allowed in defining package
}
```

#### Option C: Struct Field Tag
```go
package db

type Client struct {
    endpoint string `constructor:"required"`
}

func NewClient(endpoint string) *Client {
    return &Client{endpoint: endpoint}
}
```

---

### 2. Detection in Consumer Code

When another package attempts to initialize the struct directly:

```go
package main

import "example.com/user"

func main() {
    // Valid:
    u := user.NewUser("123", "Alice")

    // Invalid (Linter Error):
    u2 := user.User{ID: "123", Name: "Alice"}
    // ^ cannot instantiate user.User with struct literal: must be created using its constructor (marked with //constructor:required)

    u3 := &user.User{ID: "123", Name: "Alice"}
    // ^ cannot instantiate user.User with struct literal: must be created using its constructor (marked with //constructor:required)
}
```

---

## Usage

### As a Standalone CLI Analyzer

You can run `constructorcheck` using `golang.org/x/tools/go/analysis/singlechecker` or `multichecker`:

```go
package main

import (
    "github.com/ming-0x0/constructorcheck"
    "golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
    singlechecker.Main(constructorcheck.Analyzer)
}
```

Run it across your code:
```bash
go run ./cmd/constructorcheck ./...
```

---

### In `golangci-lint` (Custom Module Plugin)

`constructorcheck` provides a plugin wrapper compatible with the `golangci-lint` module plugin system.

1. Create or update `.custom-gcl.yml`:

```yaml
version: v1.64.0
plugins:
  - module: github.com/ming-0x0/constructorcheck
    path: github.com/ming-0x0/constructorcheck/plugin
```

2. Build your custom `golangci-lint` binary:

```bash
golangci-lint custom
```

3. Enable `constructorcheck` in your `.golangci.yml`:

```yaml
linters:
  enable:
    - constructorcheck
```

---

## Testing

Run tests with `go test`:

```bash
go test -v ./...
```

The test suite validates:
- Direct struct type comments (`//constructor:required`)
- Constructor function comments (`//constructor:required`)
- Struct field tags (`constructor:"required"`)
- Exemption for defining package
- Detection and diagnostics for pointer and value composite literals in consuming packages
- Module plugin registration and load mode compliance
