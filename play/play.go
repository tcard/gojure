package main

import (
	"fmt"

	"github.com/tcard/gojure/persistent"
)

func main() {
	v := persistent.NewVector()
	for i := 0; i < 40; i++ {
		v = v.Conj(fmt.Sprintf("%s %d", "hola", i))
	}
	fmt.Println(v.StringRaw())
}
