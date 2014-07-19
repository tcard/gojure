// Package persistent implements several data structures featuring
// persistence, that is, whose values don't change on any operation; instead, new,
// independent values are derived from them when needed. Thus, these data structures
// are thread safe and reduce complexity due to mutability.
//
// The main purpose of this package is providing the core data structures Gojure source
// code is made of, but it may be used independently.
//
// Implementations are heavily based on Clojure's clojure.lang.Persistent* classes.
package persistent
