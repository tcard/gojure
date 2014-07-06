package main

import (
	"fmt"

	"github.com/tcard/gojure/reader"
)

func main() {
	r := reader.FromString("a 3 (1 [] [3 (4 a b ca/def)-123])")
	fmt.Println(r.Read())
	fmt.Println(r.Read())
	fmt.Println(r.Read())
}
