package main

import (
	"fmt"
	"os"

	"go/printer"
	"go/token"
	"github.com/tcard/gojure/compiler"
	"github.com/tcard/gojure/reader"
)

func main() {
	r := reader.FromString("a 3 (1 [] [3 (4 a b ca/def)-123])")
	fmt.Println(r.Read())
	fmt.Println(r.Read())
	fmt.Println(r.Read())

	a, _ := compiler.CompileString(`

(def sum (fn* [a b] (+ a b)))
(def b 2)
(println (sum 1 b))

(println (if nil 1 2))

(def fact
  (fn* [n]
    (if (= n 0)
      1
      (* n (fact (- n 1))))))

(println (fact 6))
`)
	_ = a

	// fmt.Println(err)
	// fmt.Printf("%#v\n", a)
	printer.Fprint(os.Stdout, token.NewFileSet(), a)
	fmt.Println()
}
