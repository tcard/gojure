package main

import (
	"fmt"
	"os"

	"go/printer"
	"go/token"
	"github.com/tcard/gojure/compiler"
)

func main() {
	a, _ := compiler.CompileString(`

(println "ab

\tc")

`)
	_ = a

	// fmt.Println(err)
	// fmt.Printf("%#v\n", a)
	printer.Fprint(os.Stdout, token.NewFileSet(), a)
	fmt.Println()
}
