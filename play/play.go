package main

import (
	"fmt"
	"os"

	"go/printer"
	"go/token"
	"github.com/tcard/gojure/compiler"
)

func main() {
	a, err := compiler.CompileString(`

(import "fmt")

(fmt/Println "holas")
(fmt/Println '"holas")
(fmt/Println '(a b c))

`)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// fmt.Printf("%#v\n", a)
	printer.Fprint(os.Stdout, token.NewFileSet(), a)
}
