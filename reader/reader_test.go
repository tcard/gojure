package reader

import (
	"io"
	"reflect"
	"testing"

	"github.com/tcard/gojure/lang"
	"github.com/tcard/gojure/persistent"
)

func TestEmpty(t *testing.T) {
	cases := []string{"", " ", "    ", "\t\n \r\n  "}
	for _, s := range cases {
		r := FromString(s)
		form, err := r.Read()
		if form != nil || err != io.EOF {
			t.Errorf("String '%s' should be empty and it's not.", s)
		}
	}
}

type formTypeTestCase struct {
	shouldPass bool
	source     string
	expected   interface{}
	consume    int
}

type formTypeTest struct {
	formType   string
	assertType func(form interface{}) bool
	cases      []formTypeTestCase
}

var testCases = map[string]formTypeTest{
	"string": formTypeTest{
		formType: "string",
		assertType: func(form interface{}) bool {
			_, ok := form.(string)
			return ok
		},
		cases: []formTypeTestCase{
			{true, `  " a  " `, " a  ", len(`  " a  "`)},
			{false, `  " a  `, nil, 0},
			{true, `  "\"" `, "\"", len(`  "\""`)},
			{true, `  "ho \n l\\\"a" `, "ho \n l\\\"a", len(`  "ho \n l\\\"a"`)},
		},
	},
	"symbol": formTypeTest{
		formType: "symbol",
		assertType: func(form interface{}) bool {
			_, ok := form.(lang.Symbol)
			return ok
		},
		cases: []formTypeTestCase{
			{true, " aa ", lang.Symbol{Name: "aa"}, len(" aa")},
			{true, "abc/d", lang.Symbol{Name: "d", NS: "abc"}, len("abc/d")},
			{true, "  a=c/*d*", lang.Symbol{Name: "*d*", NS: "a=c"}, len("  a=c/*d*")},
			{true, "  /a  ", lang.Symbol{Name: "a", NS: ""}, len("  /a")},
			{false, "1notasymbol", nil, 0},
			{false, "1nota/symbol", nil, 0},
			{false, "1/notasymbol", nil, 0},
			{false, "a/b/c", nil, 0},
			{false, "ñandú", nil, 0},
			{true, ".asymbol", lang.Symbol{Name: ".asymbol"}, len(".asymbol")},
			{true, "a.b/c", lang.Symbol{Name: "c", NS: "a.b"}, len("a.b/c")},
			{true, "--0", lang.Symbol{Name: "--0"}, len("--0")},
			{true, "+-0", lang.Symbol{Name: "+-0"}, len("+-0")},
			{true, "+", lang.Symbol{Name: "+"}, len("+")},
			{true, "-", lang.Symbol{Name: "-"}, len("-")},
			{true, "/-", lang.Symbol{Name: "-", NS: ""}, len("/-")},
			{true, "ab/-3", lang.Symbol{Name: "-3", NS: "ab"}, len("ab/-3")},
		},
	},
	"int": formTypeTest{
		formType: "int",
		assertType: func(form interface{}) bool {
			_, ok := form.(int)
			return ok
		},
		cases: []formTypeTestCase{
			{true, " 123 ", 123, len(" 123")},
			{true, " +0 ", 0, len(" +0")},
			{true, " +1 ", 1, len(" +1")},
			{true, " +134 ", 134, len(" +134")},
			{true, " -0 ", 0, len(" -0")},
			{true, " -134 ", -134, len(" -134")},
			{false, "a123", nil, 0},
			{false, "--0", nil, 0},
			{false, "+-0", nil, 0},
			{false, "+", nil, 0},
			{false, "-", nil, 0},
			{false, "-/-", nil, 0},
		},
	},
	"vector": formTypeTest{
		formType: "vector",
		assertType: func(form interface{}) bool {
			_, ok := form.(*persistent.Vector)
			return ok
		},
		cases: []formTypeTestCase{
			{true, " [ ] ", persistent.NewVector(), len(" [ ]")},
			{true, "[]", persistent.NewVector(), len("[]")},
			{true, "[  1  \n\t 3 ,,,2]", persistent.NewVector(1, 3, 2), len("[  1  \n\t 3 ,,,2]")},
		},
	},
	"list": formTypeTest{
		formType: "list",
		assertType: func(form interface{}) bool {
			_, ok := form.(*persistent.List)
			return ok
		},
		cases: []formTypeTestCase{
			{true, " ( ) ", persistent.NewList(), len(" [ ]")},
			{true, "()", persistent.NewList(), len("[]")},
			{true, "(  1  \n\t 3 ,,,2)", persistent.NewList(1, 3, 2), len("[  1  \n\t 3 ,,,2]")},
		},
	},
	"quoted": formTypeTest{
		formType: "quoted",
		assertType: func(form interface{}) bool {
			_, ok := form.(*persistent.List)
			return ok
		},
		cases: []formTypeTestCase{
			{true, " '( ) ", persistent.NewList(lang.Symbol{Name: "quote"}, persistent.NewList()), len(" '( )")},
			{true, " ' 3 ", persistent.NewList(lang.Symbol{Name: "quote"}, 3), len(" ' 3")},
			{false, " '", nil, 0},
		},
	},
}

func (ftt formTypeTest) testFormType(t *testing.T) {
	for _, c := range ftt.cases {
		r := FromString(c.source)
		form, err := r.Read()
		if !c.shouldPass {
			if err != nil {
				continue
			}
			if ftt.assertType(form) {
				t.Errorf("Case '%s' shouldn't give a %s, gave '%v'.", c.source, ftt.formType, form)
			}
			continue
		}
		if !ftt.assertType(form) {
			t.Errorf("Case '%s' should give a %s, gave '%v'.", c.source, ftt.formType, form)
		}
		if !reflect.DeepEqual(form, c.expected) {
			t.Errorf("Case '%s' expected to produce %s '%v', produced '%v' instead.",
				c.source, ftt.formType, c.expected, form)
		}
		if err != nil {
			t.Errorf("Unexpected error on case '%s': %v", c.source, err)
		}
		if r.Buffered() != len(c.source)-c.consume {
			t.Errorf("Case '%s' expected to consume %d characters, consumed %d instead.",
				c.source, c.consume, len(c.source)-r.Buffered())
		}
		form, err = r.Read()
		if form != nil || err != io.EOF {
			t.Errorf("Case '%s' gave extra form or non-EOF error at the end: '%v' %v",
				c.source, form, err)
		}
	}
}

func TestBaseForms(t *testing.T) {
	for _, v := range testCases {
		v.testFormType(t)
	}
}

func testMakeCompoundForms() map[string]formTypeTest {
	ret := map[string]formTypeTest{
		"list": formTypeTest{
			formType: "list",
			assertType: func(form interface{}) bool {
				_, ok := form.(*persistent.List)
				return ok
			},
			cases: []formTypeTestCase{},
		},
		"vector": formTypeTest{
			formType: "vector",
			assertType: func(form interface{}) bool {
				_, ok := form.(*persistent.Vector)
				return ok
			},
			cases: []formTypeTestCase{},
		},
	}
	for k, _ := range ret {
		s := ""
		items := []interface{}{}
		for _, bv := range testCases {
			for _, c := range bv.cases {
				if c.shouldPass {
					s += "  " + c.source
					items = append(items, c.expected)
				}
			}
		}
		var l, r string
		var mk func(...interface{}) interface{}
		if k == "list" {
			l, r, mk = "(", ")", func(items ...interface{}) interface{} {
				return persistent.NewList(items...)
			}
		} else if k == "vector" {
			l, r, mk = "[", "]", func(items ...interface{}) interface{} {
				return persistent.NewVector(items...)
			}
		}
		for i := 0; i < 1; i++ {
			s = l + s + "  " + l + s + r + " ," + r
			items = append(items, mk(items...))
			consume := len(s)
			c := ret[k]
			c.cases = append(c.cases, formTypeTestCase{
				true, s, mk(items...), consume,
			})
			ret[k] = c
		}
	}
	return ret
}

func TestCompoundForms(t *testing.T) {
	for _, v := range testMakeCompoundForms() {
		v.testFormType(t)
	}
}

func TestQuote(t *testing.T) {
	for _, v := range testCases {
		for k, _ := range v.cases {
			if !v.cases[k].shouldPass {
				continue
			}
			v.cases[k].source = "'" + v.cases[k].source
			v.cases[k].expected = persistent.NewList(lang.Symbol{Name: "quote"}, v.cases[k].expected)
			v.cases[k].consume += 1
		}
		v.assertType = func(form interface{}) bool {
			_, ok := form.(*persistent.List)
			return ok
		}
		v.testFormType(t)
	}
}
