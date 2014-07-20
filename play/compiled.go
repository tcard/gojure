package main

import (
	reflect "reflect"
	persistent "github.com/tcard/gojure/persistent"
	lang "github.com/tcard/gojure/lang"
	fmt "fmt"
)

var _ *persistent.List
var _ lang.Symbol
var _ reflect.Type

type SymTable struct {
	parent	*SymTable
	m	map[string]interface{}
}

func (st SymTable) Get(k string) interface{} {
	v, ok := st.m[k]
	if ok {
		return v
	} else if !ok && st.parent != nil {
		return st.parent.Get(k)
	}
	panic("Undefined symbol " + k)
}

var symbols = &SymTable{m: map[string]interface {
}{}}

func main() {
	symbols.m[`false`] = false
	symbols.m[`apply`] = func(xs ...interface{}) interface{} {
		if len(xs) != 2 {
			panic("bad number of arguments to apply.")
		}
		args := []interface{}{}
		argsL := xs[1].(*persistent.List)
		for argsL != nil {
			args = append(args, argsL.First())
			argsL = argsL.Rest()
		}
		return xs[0].(func(...interface{}) interface{})(args...)
	}
	symbols.m[`nil`] = nil
	symbols.m[`-`] = func(xs ...interface{}) interface{} {
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			ret -= x.(int)
		}
		return ret
	}
	symbols.m[`println`] = func(xs ...interface{}) interface{} {
		fmt.Println(xs...)
		return nil
	}
	symbols.m[`/`] = func(xs ...interface{}) interface{} {
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			ret /= x.(int)
		}
		return ret
	}
	symbols.m[`or`] = func(xs ...interface{}) interface{} {
		for _, x := range xs {
			if !lang.IsFalse(x) {
				return x
			}
		}
		return nil
	}
	symbols.m[`and`] = func(xs ...interface{}) interface{} {
		for _, x := range xs {
			if lang.IsFalse(x) {
				return x
			}
		}
		return nil
		return true
	}
	symbols.m[`true`] = true
	symbols.m[`=`] = func(xs ...interface{}) interface{} {
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			if x.(int) != ret {
				return false
			}
		}
		return true
	}
	symbols.m[`+`] = func(xs ...interface{}) interface{} {
		if len(xs) == 0 {
			return 0
		}
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			ret += x.(int)
		}
		return ret
	}
	symbols.m[`*`] = func(xs ...interface{}) interface{} {
		if len(xs) == 0 {
			return 1
		}
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			ret *= x.(int)
		}
		return ret
	}
	var _ = interface{}(nil)
	var _ = lang.GetImport(fmt.Println).(func(xs ...interface{}) interface{})("holas")
	var _ = lang.GetImport(fmt.Println).(func(xs ...interface{}) interface{})("holas")
	var _ = lang.GetImport(fmt.Println).(func(xs ...interface{}) interface{})(persistent.NewList(lang.Symbol{"", "a"}, lang.Symbol{"", "b"}, lang.Symbol{"", "c"}))
	var _ = func(xs ...interface{}) interface{} {
		symbols.m[`fact`] = interface{}(func(xs ...interface{}) interface{} {
			symbols := &SymTable{parent: symbols, m: map[string]interface {
			}{}}
			symbols.m[`n`] = xs[0]
			return func() interface{} {
				var ifRet interface{}
				ifCond := interface{}(symbols.Get(`=`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 0))
				if lang.IsFalse(ifCond) {
					ifRet = symbols.Get(`*`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), symbols.Get(`fact`).(func(xs ...interface{}) interface{})(symbols.Get(`-`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 1)))
				} else {
					ifRet = 1
				}
				return ifRet
			}()
		})
		return nil
	}()
	var _ = symbols.Get(`println`).(func(xs ...interface{}) interface{})(symbols.Get(`fact`).(func(xs ...interface{}) interface{})(6))
	var _ = func(xs ...interface{}) interface{} {
		symbols.m[`fibo`] = interface{}(func(xs ...interface{}) interface{} {
			symbols := &SymTable{parent: symbols, m: map[string]interface {
			}{}}
			symbols.m[`n`] = xs[0]
			return func() interface{} {
				var ifRet interface{}
				ifCond := interface{}(symbols.Get(`or`).(func(xs ...interface{}) interface{})(symbols.Get(`=`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 0), symbols.Get(`=`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 1)))
				if lang.IsFalse(ifCond) {
					ifRet = symbols.Get(`+`).(func(xs ...interface{}) interface{})(symbols.Get(`fibo`).(func(xs ...interface{}) interface{})(symbols.Get(`-`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 1)), symbols.Get(`fibo`).(func(xs ...interface{}) interface{})(symbols.Get(`-`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 2)))
				} else {
					ifRet = 1
				}
				return ifRet
			}()
		})
		return nil
	}()
	var _ = symbols.Get(`println`).(func(xs ...interface{}) interface{})(symbols.Get(`fibo`).(func(xs ...interface{}) interface{})(6))
	var _ = func(xs ...interface{}) interface{} {
		symbols.m[`Y`] = interface{}(func(xs ...interface{}) interface{} {
			symbols := &SymTable{parent: symbols, m: map[string]interface {
			}{}}
			symbols.m[`f`] = xs[0]
			return interface{}(func(xs ...interface{}) interface{} {
				symbols := &SymTable{parent: symbols, m: map[string]interface {
				}{}}
				symbols.m[`x`] = xs[0]
				return symbols.Get(`x`).(func(xs ...interface{}) interface{})(symbols.Get(`x`))
			}).(func(xs ...interface{}) interface{})(interface{}(func(xs ...interface{}) interface{} {
				symbols := &SymTable{parent: symbols, m: map[string]interface {
				}{}}
				symbols.m[`g`] = xs[0]
				return symbols.Get(`f`).(func(xs ...interface{}) interface{})(interface{}(func(xs ...interface{}) interface{} {
					symbols := &SymTable{parent: symbols, m: map[string]interface {
					}{}}
					symbols.m[`arg`] = xs[0]
					return symbols.Get(`g`).(func(xs ...interface{}) interface{})(symbols.Get(`g`)).(func(xs ...interface{}) interface{})(symbols.Get(`arg`))
				}))
			}))
		})
		return nil
	}()
	var _ = func(xs ...interface{}) interface{} {
		symbols.m[`fiboY`] = interface{}(func(xs ...interface{}) interface{} {
			symbols := &SymTable{parent: symbols, m: map[string]interface {
			}{}}
			symbols.m[`f`] = xs[0]
			return interface{}(func(xs ...interface{}) interface{} {
				symbols := &SymTable{parent: symbols, m: map[string]interface {
				}{}}
				symbols.m[`n`] = xs[0]
				return func() interface{} {
					var ifRet interface{}
					ifCond := interface{}(symbols.Get(`or`).(func(xs ...interface{}) interface{})(symbols.Get(`=`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 0), symbols.Get(`=`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 1)))
					if lang.IsFalse(ifCond) {
						ifRet = symbols.Get(`+`).(func(xs ...interface{}) interface{})(symbols.Get(`f`).(func(xs ...interface{}) interface{})(symbols.Get(`-`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 1)), symbols.Get(`f`).(func(xs ...interface{}) interface{})(symbols.Get(`-`).(func(xs ...interface{}) interface{})(symbols.Get(`n`), 2)))
					} else {
						ifRet = 1
					}
					return ifRet
				}()
			})
		})
		return nil
	}()
	var _ = symbols.Get(`println`).(func(xs ...interface{}) interface{})(symbols.Get(`Y`).(func(xs ...interface{}) interface{})(symbols.Get(`fiboY`)).(func(xs ...interface{}) interface{})(6))
}
