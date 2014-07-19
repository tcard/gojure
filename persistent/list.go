package persistent

// This implementation is practically copied from Clojure's
// clojure.lang.PersistentList.

import "fmt"

// A persistent List is a sequential data structure which provides
// fast prepending (inserting at the start), and linear-time appending.
// A List value is immutable; every operation on it produces a new, independent
// value from it.
// A List value has at least one element. A empty list is represented by a empty
// *List value.
type List struct {
	first interface{}
	rest  *List
}

// Makes a new List containing these items.
func NewList(items ...interface{}) *List {
	var l *List = nil
	for i := len(items) - 1; i >= 0; i-- {
		l = l.Cons(items[i])
	}
	return l
}

// Gives the first element on the list. The list must not be empty.
func (l *List) First() interface{} {
	return l.first
}

// Gives a list with the rest of the elements of the list, skipping the first.
func (l *List) Rest() *List {
	return l.rest
}

// Makes a new list by prepending an element.
func (l *List) Cons(x interface{}) *List {
	return &List{x, l}
}

func (l *List) String() string {
	s := "("
	if l != nil {
		s += fmt.Sprint(l.First())
		for l = l.Rest(); l != nil; l = l.Rest() {
			s += " " + fmt.Sprint(l.First())
		}
	}
	s += ")"
	return s
}
