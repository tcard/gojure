package lang

import (
	"reflect"

	"github.com/tcard/gojure/persistent"
)

func GetImport(imp interface{}) interface{} {
	v := reflect.ValueOf(imp)
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
		return imp
	}
}

func IsFalse(x interface{}) bool {
	v, ok := x.(bool)
	return x == nil || !ok || !v
}
