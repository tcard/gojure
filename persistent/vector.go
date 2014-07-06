package persistent

import (
	"errors"
	"fmt"
	"strings"
)

const (
	vectorNodeShift = 5
	vectorNodeLen   = 1 << vectorNodeShift
)

type Vector struct {
	count int
	shift uint
	root  vectorNode
	tail  []interface{}
}

var (
	outOfBounds = errors.New("index of out bounds")
)

func NewVector(items ...interface{}) *Vector {
	// ret := emptyVector.AsTransient()
	ret := &Vector{0, vectorNodeShift, emptyVectorNode, []interface{}{}}
	for _, x := range items {
		ret = ret.Conj(x)
	}
	return ret
}

func (v *Vector) Count() int {
	return v.count
}

var emptyVectorNode = vectorNode{items: make([]interface{}, vectorNodeLen)}
var emptyVector = &Vector{0, vectorNodeShift, emptyVectorNode, []interface{}{}}

type vectorNode struct {
	items []interface{}
}

func (v *Vector) tailoff() int {
	if v.count < vectorNodeLen {
		return 0
	}
	return ((v.count - 1) >> vectorNodeShift) << vectorNodeShift
}

func (v *Vector) arrayFor(i int) []interface{} {
	if i < 0 || i >= v.count {
		panic(outOfBounds)
	}
	if i >= v.tailoff() {
		return v.tail
	}
	n := v.root
	for level := v.shift; level > 0; level -= vectorNodeShift {
		n = n.items[(i>>level)&((vectorNodeShift)-1)].(vectorNode)
	}
	return n.items
}

func (v *Vector) Nth(i int) interface{} {
	subsl := v.arrayFor(i)
	return subsl[i&(vectorNodeLen-1)]
}

func (v *Vector) Assoc(i int, x interface{}) *Vector {
	if i < 0 || i > v.count {
		panic(outOfBounds)
	}
	if i == v.count {
		return v.Conj(x)
	}
	if i >= v.tailoff() {
		newTail := make([]interface{}, len(v.tail))
		copy(newTail, v.tail)
		newTail[i&(1<<(v.shift-1))] = x
		return &Vector{v.count, v.shift, v.root, newTail}
	}
	return &Vector{v.count, v.shift, doAssoc(v.shift, v.root, i, x), v.tail}
}

func doAssoc(shift uint, node vectorNode, i int, x interface{}) vectorNode {
	ret := vectorNode{items: make([]interface{}, len(node.items))}
	copy(ret.items, node.items)
	if shift == 0 {
		ret.items[i&(vectorNodeLen-1)] = x
	} else {
		subi := (i >> shift) & (vectorNodeLen - 1)
		ret.items[subi] = doAssoc(shift-vectorNodeShift, node.items[subi].(vectorNode), i, x)
	}
	return ret
}

func (v *Vector) Conj(x interface{}) *Vector {
	if v.count-v.tailoff() < vectorNodeLen {
		newTail := make([]interface{}, len(v.tail)+1)
		copy(newTail, v.tail)
		newTail[len(v.tail)] = x
		return &Vector{v.count + 1, v.shift, v.root, newTail}
	}
	newRoot := vectorNode{}
	tailNode := vectorNode{v.tail}
	newShift := v.shift
	if (v.count >> vectorNodeShift) > (1 << v.shift) {
		newRoot = vectorNode{make([]interface{}, vectorNodeLen)}
		newRoot.items[0] = v.root
		newRoot.items[1] = newPath(v.shift, tailNode)
		newShift += vectorNodeShift
	} else {
		newRoot = v.pushTail(v.shift, v.root, tailNode)
	}
	return &Vector{v.count + 1, newShift, newRoot, []interface{}{x}}
}

func (v *Vector) pushTail(shift uint, parent vectorNode, tailNode vectorNode) vectorNode {
	subi := ((v.count - 1) >> shift) & (vectorNodeLen - 1)
	ret := vectorNode{make([]interface{}, len(parent.items))}
	copy(ret.items, parent.items)
	nodeToInsert := vectorNode{}
	if shift == vectorNodeShift {
		nodeToInsert = tailNode
	} else {
		child, ok := parent.items[subi].(vectorNode)
		if ok {
			nodeToInsert = v.pushTail(shift-vectorNodeShift, child, tailNode)
		} else {
			nodeToInsert = newPath(shift-vectorNodeShift, tailNode)
		}
	}
	ret.items[subi] = nodeToInsert
	return ret
}

func newPath(shift uint, node vectorNode) vectorNode {
	if shift == 0 {
		return node
	}
	ret := vectorNode{make([]interface{}, vectorNodeLen)}
	ret.items[0] = newPath(shift-vectorNodeShift, node)
	return ret
}

func (v *Vector) StringRaw() string {
	var f func(interface{}, int) string
	f = func(x interface{}, lvl int) string {
		switch tx := x.(type) {
		case vectorNode:

			s := "\n" + strings.Repeat(" ", lvl) + "{\n"
			lvl += 1
			for i, v := range tx.items {
				if i > 0 {
					s += " "
				}
				s += f(v, lvl)
			}
			lvl -= 1
			s += "\n" + strings.Repeat(" ", lvl) + "}"
			return s
		default:
			return fmt.Sprint(x)
		}
	}
	return f(v.root, 0) + " + " + fmt.Sprint(v.tail)
}

/*func (v *Vector) AsTransient() *TransientVector {
	ret := &TransientVector{Vector{
		v.count,
		v.shift,
		vectorNode{make([]interface{}, len(v.node.items))},
		make([]interface{}, len(v.node.items)),
	}}
	copy(ret.root.items, v.node.items)
	copy(ret.tail, v.tail)
	return retn
}

type TransientVector struct {
	Vector
}

func (v *TransientVector) Conj(x interface{}) *TransientVector {
}*/
