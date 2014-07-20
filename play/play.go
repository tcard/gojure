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

(def fact
  (fn* [n]
    (if (= n 0)
      1
      (* n (fact (- n 1))))))

(println (fact 6))

(def fibo
  (fn* [n]
    (if (or (= n 0) (= n 1))
      1
      (+ (fibo (- n 1)) (fibo (- n 2))))))

(println (fibo 6))

(def Y
  (fn* [f]
    ((fn* [x] (x x))
      (fn* [g]
        (f (fn* [arg] ((g g) arg)))))))

(def fiboY
  (fn* [f]
    (fn* [n]
      (if (or (= n 0) (= n 1))
        1
        (+ (f (- n 1)) (f (- n 2)))))))

(println ((Y fiboY) 6))

`)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// fmt.Printf("%#v\n", a)
	printer.Fprint(os.Stdout, token.NewFileSet(), a)
}
