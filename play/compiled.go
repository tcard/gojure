package main

import (
	fmt "fmt"
	reflect "reflect"

	lang "github.com/tcard/gojure/lang"
	persistent "github.com/tcard/gojure/persistent"
)

var _ *persistent.List
var _ lang.Symbol
var _ reflect.Type

type SymTable struct {
	parent *SymTable
	m      map[string]interface{}
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
	symbols.m[`-`] = func(xs ...interface{}) interface{} {
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			ret -= x.(int)
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
	symbols.m[`/`] = func(xs ...interface{}) interface{} {
		ret := xs[0].(int)
		for _, x := range xs[1:] {
			ret /= x.(int)
		}
		return ret
	}
	symbols.m[`println`] = func(xs ...interface{}) interface{} {
		fmt.Println(xs...)
		return nil
	}
	symbols.m[`true`] = true
	symbols.m[`nil`] = nil
	symbols.m[`false`] = false
	var _ = interface {
	}(nil)
	var _ = func() interface{} {
		v := reflect.ValueOf(fmt.Println)
		if v.Kind() == reflect.Func {
			return func(xs ...interface{}) interface{} {
				invals := []reflect.Value{}
				for _, x := range xs {
					invals = append(invals, reflect.ValueOf(x))
				}
				refvals := v.Call(invals)
				vals := []interface{}{}
				for _, rv := range refvals {
					vals = append(vals, rv)
				}
				if len(vals) == 0 {
					return vals[0]
				} else if len(vals) > 0 {
					return persistent.NewVector(vals...)
				} else {
					return nil
				}
			}
		} else {
			return fmt.Println
		}
	}().(func(xs ...interface {
	}) interface {
	})("holas")
	var _ = func() interface{} {
		v := reflect.ValueOf(fmt.Println)
		if v.Kind() == reflect.Func {
			return func(xs ...interface{}) interface{} {
				invals := []reflect.Value{}
				for _, x := range xs {
					invals = append(invals, reflect.ValueOf(x))
				}
				refvals := v.Call(invals)
				vals := []interface{}{}
				for _, rv := range refvals {
					vals = append(vals, rv)
				}
				if len(vals) == 0 {
					return vals[0]
				} else if len(vals) > 0 {
					return persistent.NewVector(vals...)
				} else {
					return nil
				}
			}
		} else {
			return fmt.Println
		}
	}().(func(xs ...interface {
	}) interface {
	})("holas")
	var _ = func() interface{} {
		v := reflect.ValueOf(fmt.Println)
		if v.Kind() == reflect.Func {
			return func(xs ...interface{}) interface{} {
				invals := []reflect.Value{}
				for _, x := range xs {
					invals = append(invals, reflect.ValueOf(x))
				}
				refvals := v.Call(invals)
				vals := []interface{}{}
				for _, rv := range refvals {
					vals = append(vals, rv)
				}
				if len(vals) == 0 {
					return vals[0]
				} else if len(vals) > 0 {
					return persistent.NewVector(vals...)
				} else {
					return nil
				}
			}
		} else {
			return fmt.Println
		}
	}().(func(xs ...interface {
	}) interface {
	})(persistent.NewList(lang.Symbol{"", "a"}, lang.Symbol{"", "b"}, lang.Symbol{"", "c"}))
}
