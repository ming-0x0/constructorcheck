package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/ming-0x0/constructorcheck"
)

func main() {
	singlechecker.Main(constructorcheck.Analyzer)
}
