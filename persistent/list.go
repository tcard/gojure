package persistent

import "fmt"

type List struct {
	first interface{}
	rest  *List
}

func NewList(items ...interface{}) *List {
	var l *List = nil
	for i := len(items) - 1; i >= 0; i-- {
		l = l.Cons(items[i])
	}
	return l
}

func (l *List) First() interface{} {
	return l.first
}

func (l *List) Next() *List {
	return l.rest
}

func (l *List) Cons(x interface{}) *List {
	return &List{x, l}
}

func (l *List) String() string {
	s := "("
	if l != nil {
		s += fmt.Sprint(l.First())
		for l = l.Next(); l != nil; l = l.Next() {
			s += " " + fmt.Sprint(l.First())
		}
	}
	s += ")"
	return s
}
