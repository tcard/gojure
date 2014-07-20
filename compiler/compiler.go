package compiler

import (
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"strconv"
	"strings"

	"github.com/tcard/gojure/lang"
	"github.com/tcard/gojure/persistent"
	"github.com/tcard/gojure/reader"
)

// Compile Gojure source coe into a Go AST.
func Compile(r io.Reader) (*ast.File, error) {
	gr := reader.From(r)

	env := &SymExprsTable{
		imports: make(map[string][]string),
		m:       map[string]ast.Expr{},
	}
	for k, v := range Symbols.m {
		env.m[k] = v
	}
	for k, v := range Symbols.imports {
		env.imports[k] = v
	}

	main := &ast.FuncDecl{
		Name: identExpr("main"),
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{List: []ast.Stmt{}},
	}

	symMap := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   identExpr("string"),
			Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
		Elts: []ast.Expr{}}
	for k, v := range env.m {
		symMap.Elts = append(symMap.Elts, &ast.KeyValueExpr{
			Key:   &ast.BasicLit{Kind: token.STRING, Value: "`" + k + "`"},
			Value: v,
		})
	}

	for k, v := range env.m {
		main.Body.List = append(main.Body.List,
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.IndexExpr{
						X: &ast.SelectorExpr{
							X:   identExpr("symbols"),
							Sel: identExpr("m")},
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
			main.Body.List = append(main.Body.List, &ast.DeclStmt{&ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{&ast.ValueSpec{
					Names:  []*ast.Ident{identExpr("_")},
					Values: []ast.Expr{expr.(ast.Expr)}}}}})
		}
		form, err = gr.Read()
	}
	if err != nil && err != io.EOF {
		return nil, err
	}

	imports := "import ("
	for name, paths := range env.imports {
		for _, path := range paths {
			imports += "\n" + name + " \"" + path + "\""
		}
	}
	imports += ")"

	file, _ := parser.ParseFile(&token.FileSet{}, "", `
		package main

		`+imports+`

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
			panic("Undefined symbol "+k)
		}`, 0)

	file.Decls = append(file.Decls,
		&ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{identExpr("symbols")},
				Values: []ast.Expr{&ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: identExpr("SymTable"),
						Elts: []ast.Expr{
							&ast.KeyValueExpr{
								Key: identExpr("m"),
								Value: &ast.CompositeLit{
									Type: &ast.MapType{
										Key:   identExpr("string"),
										Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
									Elts: []ast.Expr{}}}}}}}}}},
		main)

	return file, nil
}

// Compile Gojure source coe into a Go AST.
func CompileString(s string) (*ast.File, error) {
	return Compile(strings.NewReader(s))
}

// Compiles a single Gojure form into a Go expression, returning side effects on the symbol
// table (definitions, etc.) in a new value.
func CompileForm(form interface{}, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	switch vform := form.(type) {
	case int:
		return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(vform)}, env, nil
	case bool:
		if vform {
			return identExpr("true"), env, nil
		} else {
			return identExpr("false"), env, nil
		}
	case nil:
		return &ast.CallExpr{
			Fun:  ifaceAST,
			Args: []ast.Expr{identExpr("nil")},
		}, env, nil
	case string:
		return &ast.BasicLit{Kind: token.STRING, Value: `"` + vform + `"`}, env, nil
	case lang.Symbol:
		return compileSymbol(vform, env)
	case *persistent.List:
		opform := vform.First()
		sym, isSym := opform.(lang.Symbol)

		if isSym {
			switch sym.Name {
			case "def":
				return compileDef(vform.Rest(), env)
			case "fn*":
				return compileFn(vform.Rest(), env)
			case "if":
				return compileIf(vform.Rest(), env)
			case "quote":
				if vform.Rest() == nil {
					return CompileForm(nil, env)
				}
				q, err := quote(vform.Rest().First())
				return q, env, err
			case "import":
				if vform.Rest() == nil {
					return CompileForm(nil, env)
				}
				alias := ""
				if vform.Rest().Rest() != nil {
					alias = vform.Rest().Rest().First().(lang.Symbol).Name
				}
				err := env.Import(vform.Rest().First().(string), alias)
				if err != nil {
					return nil, env, err
				}
				return CompileForm(nil, env)
			}
		}

		return compileCall(vform, env)
	case *persistent.Vector:
		return compileVector(vform, env, false)
	}
	return nil, env, nil
}

func compileSymbol(sym lang.Symbol, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	if e, ok := env.Get(sym.Name, sym.NS); !ok {
		return nil, env, errors.New("Undefined symbol: " + sym.String())
	} else if sym.NS != "" {
		return e, env, nil
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   identExpr("symbols"),
			Sel: identExpr("Get")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "`" + sym.Name + "`"}},
	}, env, nil
}

func compileDef(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	ident := form.First().(lang.Symbol).Name
	env.m[ident] = nil // &ast.BasicLit{Kind: token.STRING, Value: "`placeholder`"}
	def, env, err := CompileForm(form.Rest().First(), env)
	if err != nil {
		return nil, env, err
	}
	env.m[ident] = def
	return &ast.CallExpr{
		Args: []ast.Expr{},
		Fun: &ast.FuncLit{
			Type: fnAST,
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.IndexExpr{
							X: &ast.SelectorExpr{
								X:   identExpr("symbols"),
								Sel: identExpr("m")},
							Index: &ast.BasicLit{Kind: token.STRING, Value: "`" + ident + "`"},
						}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{def}},
				&ast.ReturnStmt{Results: []ast.Expr{identExpr("nil")}}}}}}, env, nil
}

func compileFn(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	args := form.First().(*persistent.Vector)
	bodyf := form.Rest().First()
	fnEnv := &SymExprsTable{parent: env, m: map[string]ast.Expr{}}
	for i := 0; i < args.Count(); i++ {
		fnEnv.m[args.Nth(i).(lang.Symbol).Name] = &ast.IndexExpr{
			X:     identExpr("xs"),
			Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)},
		}
	}
	newSyms := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   identExpr("string"),
			Value: &ast.InterfaceType{Methods: &ast.FieldList{}}},
		Elts: []ast.Expr{}}
	newTable := &ast.UnaryExpr{
		Op: token.AND,
		X: &ast.CompositeLit{
			Type: identExpr("SymTable"),
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   identExpr("parent"),
					Value: identExpr("symbols")},
				&ast.KeyValueExpr{
					Key:   identExpr("m"),
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
					Lhs: []ast.Expr{identExpr("symbols")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{newTable}}}}}
	for k, v := range fnEnv.m {
		ret.Body.List = append(ret.Body.List,
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					&ast.IndexExpr{
						X: &ast.SelectorExpr{
							X:   identExpr("symbols"),
							Sel: identExpr("m")},
						Index: &ast.BasicLit{Kind: token.STRING, Value: "`" + k + "`"},
					}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{v}})
	}
	ret.Body.List = append(ret.Body.List, &ast.ReturnStmt{Results: []ast.Expr{body}})
	return &ast.CallExpr{
		Fun:  ifaceAST,
		Args: []ast.Expr{ret}}, env, nil
}

func compileCall(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	op, env, err := CompileForm(form.First(), env)
	if err != nil {
		return nil, env, err
	}
	args := []ast.Expr{}
	for rest := form.Rest(); rest != nil; rest = rest.Rest() {
		arg, env, err := CompileForm(rest.First(), env)
		if err != nil {
			return nil, env, err
		}
		args = append(args, arg)
	}
	ret := &ast.CallExpr{
		Args: args,
	}
	ret.Fun = &ast.TypeAssertExpr{
		X:    op,
		Type: fnAST,
	}

	return ret, env, nil
}

func compileIf(form *persistent.List, env *SymExprsTable) (ast.Expr, *SymExprsTable, error) {
	cond, env, err := CompileForm(form.First(), env)
	if err != nil {
		return nil, env, err
	}
	yes, env, err := CompileForm(form.Rest().First(), env)
	if err != nil {
		return nil, env, err
	}
	no, env, err := CompileForm(form.Rest().Rest().First(), env)
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
						Names: []*ast.Ident{identExpr("ifRet")},
					}}}},
				&ast.AssignStmt{
					Lhs: []ast.Expr{identExpr("ifCond")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{&ast.CallExpr{
						Fun:  ifaceAST,
						Args: []ast.Expr{cond},
					}},
				},
				&ast.IfStmt{
					Cond: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   identExpr("lang"),
							Sel: identExpr("IsFalse"),
						},
						Args: []ast.Expr{identExpr("ifCond")},
					},
					Body: &ast.BlockStmt{List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{identExpr("ifRet")},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{no},
						}}},
					Else: &ast.BlockStmt{List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{identExpr("ifRet")},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{yes},
						}}},
				},
				&ast.ReturnStmt{Results: []ast.Expr{identExpr("ifRet")}},
			}}},
	}, env, nil
}

func compileVector(v *persistent.Vector, env *SymExprsTable, quoting bool) (ast.Expr, *SymExprsTable, error) {
	ret := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   identExpr("persistent"),
			Sel: identExpr("NewVector"),
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
				X:   identExpr("lang"),
				Sel: identExpr("Symbol"),
			},
			Elts: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: `"` + v.NS + `"`},
				&ast.BasicLit{Kind: token.STRING, Value: `"` + v.Name + `"`},
			},
		}, nil
	case *persistent.List:
		l := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   identExpr("persistent"),
				Sel: identExpr("NewList"),
			},
			Args: []ast.Expr{},
		}
		for v != nil {
			item, err := quote(v.First())
			if err != nil {
				return nil, err
			}
			l.Args = append(l.Args, item)
			v = v.Rest()
		}
		return l, nil
	case *persistent.Vector:
		e, _, err := compileVector(v, nil, true)
		return e, err
	}
	v, _, err := CompileForm(thingy, nil)
	return v, err
}

var ifaceAST = func() ast.Expr {
	expr, _ := parser.ParseExpr(`interface{}`)
	return expr
}()

var fnAST = &ast.FuncType{
	Params: &ast.FieldList{List: []*ast.Field{
		{
			Names: []*ast.Ident{identExpr("xs")},
			Type:  &ast.Ellipsis{Elt: ifaceAST}}}},
	Results: &ast.FieldList{List: []*ast.Field{
		{Type: ifaceAST},
	}}}

func identExpr(name string) *ast.Ident {
	return &ast.Ident{Name: name, Obj: &ast.Object{}}
}

type SymExprsTable struct {
	parent  *SymExprsTable
	m       map[string]ast.Expr
	imports map[string][]string
}

func (st SymExprsTable) Get(s string, ns string) (ast.Expr, bool) {
	if st.imports != nil && ns != "" {
		if _, ok := st.imports[ns]; ok {
			e, _ := parser.ParseExpr(`
				lang.GetImport(` + ns + `.` + s + `)`)
			return e, true
		}
	}
	v, ok := st.m[s]
	if ok {
		return v, true
	} else if !ok && st.parent != nil {
		return st.parent.Get(s, ns)
	}
	return nil, false
}

func (st SymExprsTable) Import(pkgName string, alias string) error {
	pkg, err := build.Import(pkgName, ".", build.AllowBinary)
	if err != nil {
		return err
	}
	tbl := &st
	for tbl.parent != nil {
		tbl = tbl.parent
	}
	if alias == "" {
		alias = pkg.Name
	}
	if alias != "." && alias != "_" && len(tbl.imports[alias]) > 0 {
		tbl.imports[alias][0] = pkgName
	} else {
		tbl.imports[alias] = append(tbl.imports[alias], pkgName)
	}
	return nil
}

var Symbols = &SymExprsTable{
	imports: map[string][]string{
		"fmt":        []string{"fmt"},
		"reflect":    []string{"reflect"},
		"persistent": []string{"github.com/tcard/gojure/persistent"},
		"lang":       []string{"github.com/tcard/gojure/lang"},
	},
	m: map[string]ast.Expr{
		"apply": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
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
			}`)
			return e
		}(),
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
		"or": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				for _, x := range xs {
					if !lang.IsFalse(x) {
						return x
					}
				}
				return nil
			}`)
			return e
		}(),
		"and": func() ast.Expr {
			e, _ := parser.ParseExpr(`
			func(xs ...interface{}) interface{} {
				for _, x := range xs {
					if lang.IsFalse(x) {
						return x
					}
				}
				return nil
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
