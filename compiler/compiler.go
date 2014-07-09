package compiler

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strconv"
	"strings"

	"github.com/tcard/gojure/lang"
	"github.com/tcard/gojure/persistent"
	"github.com/tcard/gojure/reader"
)

type SymExprsTable struct {
	parent *SymExprsTable
	m      map[string]ast.Expr
}

func (st SymExprsTable) Get(s string) (interface{}, bool) {
	v, ok := st.m[s]
	if ok {
		return v, true
	} else if !ok && st.parent != nil {
		return st.parent.Get(s)
	}
	return nil, false
}

var Symbols = &SymExprsTable{
	m: map[string]ast.Expr{
		"+": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				if len(xs) == 0 {
					return 0
				}
				ret := xs[0].(int)
				for _, x := range xs[1:] {
					ret += x.(int)
				}
				return ret
			}`)
			return e
		}(),
		"-": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				ret := xs[0].(int)
				for _, x := range xs[1:] {
					ret -= x.(int)
				}
				return ret
			}`)
			return e
		}(),
		"*": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				if len(xs) == 0 {
					return 1
				}
				ret := xs[0].(int)
				for _, x := range xs[1:] {
					ret *= x.(int)
				}
				return ret
			}	`)
			return e
		}(),
		"/": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				ret := xs[0].(int)
				for _, x := range xs[1:] {
					ret /= x.(int)
				}
				return ret
			}`)
			return e
		}(),
		"=": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				ret := xs[0].(int)
				for _, x := range xs[1:] {
					if x.(int) != ret {
						return false
					}
				}
				return true
			}`)
			return e
		}(),
		"println": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				fmt.Println(xs...)
				return nil
			}`)
			return e
		}(),
		"true": func() ast.Expr {
			e, _ := parser.ParseExpr(`true`)
			return e
		}(),
		"false": func() ast.Expr {
			e, _ := parser.ParseExpr(`false`)
			return e
		}(),
		"nil": func() ast.Expr {
			e, _ := parser.ParseExpr(`nil`)
			return e
		}(),
	},
}

var ifaceAST = &ast.InterfaceType{Methods: &ast.FieldList{}}
var fnAST = &ast.FuncType{
	Params: &ast.FieldList{List: []*ast.Field{
		{
			Names: []*ast.Ident{{Name: "xs", Obj: &ast.Object{}}},
			Type:  &ast.Ellipsis{Elt: ifaceAST}}}},
	Results: &ast.FieldList{List: []*ast.Field{
		{Type: ifaceAST},
	}}}

func Compile(r io.Reader) (*ast.File, error) {
	gr := reader.From(r)

	env := &SymExprsTable{m: map[string]ast.Expr{}}
	for k, v := range Symbols.m {
		env.m[k] = v
	}

	main := &ast.FuncDecl{
		Name: &ast.Ident{Name: "main", Obj: &ast.Object{}},
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{List: []ast.Stmt{}},
	}

	symMap := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   &ast.Ident{Name: "string", Obj: &ast.Object{}},
			Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
		Elts: []ast.Expr{}}
	for k, v := range env.m {
		symMap.Elts = append(symMap.Elts, &ast.KeyValueExpr{
			Key:   &ast.BasicLit{Kind: token.STRING, Value: "`" + k + "`"},
			Value: v,
		})
	}

	file, _ := parser.ParseFile(&token.FileSet{}, "", `
				package main

				import (
					"fmt"
					"github.com/tcard/gojure/persistent"
					"github.com/tcard/gojure/lang"
				)

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
					panic("Undefined symbol "+k)
				}`, 0)

	for k, v := range env.m {
		main.Body.List = append(main.Body.List,
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.IndexExpr{
						X: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "symbols", Obj: &ast.Object{}},
							Sel: &ast.Ident{Name: "m", Obj: &ast.Object{}}},
						Index: &ast.BasicLit{Kind: token.STRING, Value: "`" + k + "`"},
					}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{v}})
	}

	var expr ast.Expr
	form, err := gr.Read()
	for err == nil {
		expr, env, err = CompileForm(form, env)
		if err != nil {
			break
		}
		if expr != nil {
			main.Body.List = append(main.Body.List, &ast.ExprStmt{expr})
		}
		form, err = gr.Read()
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	file.Decls = append(file.Decls,
		&ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{{Name: "symbols", Obj: &ast.Object{}}},
				Values: []ast.Expr{&ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: &ast.Ident{Name: "SymTable", Obj: &ast.Object{}},
						Elts: []ast.Expr{
							&ast.KeyValueExpr{
								Key: &ast.Ident{Name: "m", Obj: &ast.Object{}},
								Value: &ast.CompositeLit{
									Type: &ast.MapType{
										Key:   &ast.Ident{Name: "string", Obj: &ast.Object{}},
										Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
									Elts: []ast.Expr{}}}}}}}}}},
		main)

	return file, nil
}

func CompileString(s string) (*ast.File, error) {
	return Compile(strings.NewReader(s))
}

func CompileForm(form interface{}, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	switch vform := form.(type) {
	case int:
		return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(vform)}, env, nil
	case bool:
		if vform {
			return &ast.Ident{Name: "true", Obj: &ast.Object{}}, env, nil
		} else {
			return &ast.Ident{Name: "false", Obj: &ast.Object{}}, env, nil
		}
	case nil:
		return &ast.Ident{Name: "nil", Obj: &ast.Object{}}, env, nil
	case lang.Symbol:
		return compileSymbol(vform, env)
	case *persistent.List:
		opform := vform.First()
		sym, isSym := opform.(lang.Symbol)

		if isSym {
			switch sym.Name {
			case "def":
				return compileDef(vform.Next(), env)
			case "fn*":
				return compileFn(vform.Next(), env)
			case "if":
				return compileIf(vform.Next(), env)
			case "quote":
				if vform.Next() == nil {
					return CompileForm(nil, env)
				}
				q, err := quote(vform.Next().First())
				return q, env, err
			}
		}

		return compileCall(vform, env)
	case *persistent.Vector:
		return compileVector(vform, env, false)
	}
	return nil, env, nil
}

func compileSymbol(sym lang.Symbol, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	if _, ok := env.Get(sym.Name); !ok {
		return nil, env, errors.New("Undefined symbol: " + sym.String())
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "symbols", Obj: &ast.Object{}},
			Sel: &ast.Ident{Name: "Get", Obj: &ast.Object{}}},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "`" + sym.Name + "`"}},
	}, env, nil
}

func compileDef(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	ident := form.First().(lang.Symbol).Name
	env.m[ident] = nil // &ast.BasicLit{Kind: token.STRING, Value: "`placeholder`"}
	def, env, err := CompileForm(form.Next().First(), env)
	if err != nil {
		return nil, env, err
	}
	env.m[ident] = def
	return &ast.CallExpr{
		Args: []ast.Expr{},
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.IndexExpr{
							X: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "symbols", Obj: &ast.Object{}},
								Sel: &ast.Ident{Name: "m", Obj: &ast.Object{}}},
							Index: &ast.BasicLit{Kind: token.STRING, Value: "`" + ident + "`"},
						}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{def}}}}}}, env, nil
}

func compileFn(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	args := form.First().(*persistent.Vector)
	bodyf := form.Next().First()
	fnEnv := &SymExprsTable{parent: env, m: map[string]ast.Expr{}}
	for i := 0; i < args.Count(); i++ {
		fnEnv.m[args.Nth(i).(lang.Symbol).Name] = &ast.IndexExpr{
			X:     &ast.Ident{Name: "xs", Obj: &ast.Object{}},
			Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)},
		}
	}
	newSyms := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   &ast.Ident{Name: "string", Obj: &ast.Object{}},
			Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
		Elts: []ast.Expr{}}
	newTable := &ast.UnaryExpr{
		Op: token.AND,
		X: &ast.CompositeLit{
			Type: &ast.Ident{Name: "SymTable", Obj: &ast.Object{}},
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   &ast.Ident{Name: "parent", Obj: &ast.Object{}},
					Value: &ast.Ident{Name: "symbols", Obj: &ast.Object{}}},
				&ast.KeyValueExpr{
					Key:   &ast.Ident{Name: "m", Obj: &ast.Object{}},
					Value: newSyms,
				}}}}

	body, fnEnv, err := CompileForm(bodyf, fnEnv)
	if err != nil {
		return nil, nil, err
	}
	ret := &ast.FuncLit{
		Type: fnAST,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "symbols", Obj: &ast.Object{}}},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{newTable}}}}}
	for k, v := range fnEnv.m {
		ret.Body.List = append(ret.Body.List,
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.IndexExpr{
						X: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "symbols", Obj: &ast.Object{}},
							Sel: &ast.Ident{Name: "m", Obj: &ast.Object{}}},
						Index: &ast.BasicLit{Kind: token.STRING, Value: "`" + k + "`"},
					}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{v}})
	}
	ret.Body.List = append(ret.Body.List, &ast.ReturnStmt{Results: []ast.Expr{body}})
	return ret, env, nil
}

func compileCall(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	op, env, err := CompileForm(form.First(), env)
	if err != nil {
		return nil, env, err
	}
	args := []ast.Expr{}
	for rest := form.Next(); rest != nil; rest = rest.Next() {
		arg, env, err := CompileForm(rest.First(), env)
		if err != nil {
			return nil, env, err
		}
		args = append(args, arg)
	}
	return &ast.CallExpr{
		Args: args,
		Fun: &ast.TypeAssertExpr{
			X:    op,
			Type: fnAST,
		},
	}, env, nil
}

func compileIf(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	cond, env, err := CompileForm(form.First(), env)
	if err != nil {
		return nil, env, err
	}
	yes, env, err := CompileForm(form.Next().First(), env)
	if err != nil {
		return nil, env, err
	}
	no, env, err := CompileForm(form.Next().Next().First(), env)
	if err != nil {
		return nil, env, err
	}
	return &ast.CallExpr{
		Args: []ast.Expr{},
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{}},
				Results: &ast.FieldList{List: []*ast.Field{
					{Type: ifaceAST},
				}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.DeclStmt{&ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{&ast.ValueSpec{
						Type:  ifaceAST,
						Names: []*ast.Ident{{Name: "ifRet", Obj: &ast.Object{}}},
					}}}},
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "ifCond", Obj: &ast.Object{}}},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{&ast.CallExpr{
						Fun:  ifaceAST,
						Args: []ast.Expr{cond},
					}},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{Name: "b", Obj: &ast.Object{}},
						&ast.Ident{Name: "isBool", Obj: &ast.Object{}},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.TypeAssertExpr{
							X:    &ast.Ident{Name: "ifCond", Obj: &ast.Object{}},
							Type: &ast.Ident{Name: "bool", Obj: &ast.Object{}},
						},
					},
				},
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.Ident{Name: "ifCond", Obj: &ast.Object{}},
							Op: token.EQL,
							Y:  &ast.Ident{Name: "nil", Obj: &ast.Object{}},
						},
						Op: token.LOR,
						Y: &ast.BinaryExpr{
							X:  &ast.Ident{Name: "isBool", Obj: &ast.Object{}},
							Op: token.LAND,
							Y: &ast.BinaryExpr{
								X:  &ast.Ident{Name: "b", Obj: &ast.Object{}},
								Op: token.EQL,
								Y:  &ast.Ident{Name: "false", Obj: &ast.Object{}},
							},
						},
					},
					Body: &ast.BlockStmt{List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{&ast.Ident{Name: "ifRet", Obj: &ast.Object{}}},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{no},
						}}},
					Else: &ast.BlockStmt{List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{&ast.Ident{Name: "ifRet", Obj: &ast.Object{}}},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{yes},
						}}},
				},
				&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "ifRet", Obj: &ast.Object{}}}},
			}}},
	}, env, nil
}

func compileVector(v *persistent.Vector, env *SymExprsTable, quoting bool) (ast.Expr, *SymExprsTable, error) {
	ret := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "persistent", Obj: &ast.Object{}},
			Sel: &ast.Ident{Name: "NewVector", Obj: &ast.Object{}},
		},
		Args: []ast.Expr{},
	}
	var item ast.Expr
	var err error
	for i := 0; i < v.Count(); i++ {
		if quoting {
			item, err = quote(v.Nth(i))
			if err != nil {
				return nil, env, err
			}
		} else {
			item, env, err = CompileForm(v.Nth(i), env)
			if err != nil {
				return nil, env, err
			}
		}
		ret.Args = append(ret.Args, item)
	}
	return ret, env, err
}

func quote(thingy interface{}) (ast.Expr, error) {
	switch v := thingy.(type) {
	case lang.Symbol:
		return &ast.CompositeLit{
			Type: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "lang", Obj: &ast.Object{}},
				Sel: &ast.Ident{Name: "Symbol", Obj: &ast.Object{}},
			},
			Elts: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: `"` + v.NS + `"`},
				&ast.BasicLit{Kind: token.STRING, Value: `"` + v.Name + `"`},
			},
		}, nil
	case *persistent.List:
		l := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "persistent", Obj: &ast.Object{}},
				Sel: &ast.Ident{Name: "NewList", Obj: &ast.Object{}},
			},
			Args: []ast.Expr{},
		}
		for v != nil {
			item, err := quote(v.First())
			if err != nil {
				return nil, err
			}
			l.Args = append(l.Args, item)
			v = v.Next()
		}
		return l, nil
	case *persistent.Vector:
		e, _, err := compileVector(v, nil, true)
		return e, err
	}
	v, _, err := CompileForm(thingy, nil)
	return v, err
}
