package reader

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/tcard/gojure/lang"
	"github.com/tcard/gojure/persistent"
)

func From(r io.Reader) GojureReader {
	bufr, ok := r.(*bufio.Reader)
	if !ok {
		bufr = bufio.NewReader(r)
	}
	return GojureReader{bufr}
}

func FromString(s string) GojureReader {
	return From(strings.NewReader(s))
}

type GojureReader struct {
	*bufio.Reader
}

func (r GojureReader) ReadByte() (byte, error) {
	b, err := r.Reader.ReadByte()
	// fmt.Println("ReadByte", string(b), err)
	return b, err
}

func (r GojureReader) UnreadByte() error {
	err := r.Reader.UnreadByte()
	// fmt.Println("UnreadByte", err)
	return err
}

func (r GojureReader) Read() (interface{}, error) {
	c, err := r.skipSpace()
	if err != nil {
		return nil, err
	}
	switch c {
	case '(':
		items, err := r.readCompound(')')
		if err != nil {
			return nil, err
		}
		return persistent.NewList(items...), nil
	case '[':
		items, err := r.readCompound(']')
		if err != nil {
			return nil, err
		}
		return persistent.NewVector(items...), nil
	default:
		r.UnreadByte()
		return r.readAtom()
	}
}

func (r GojureReader) readAtom() (interface{}, error) {
	// Just symbols and ints for now.
	c, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	r.UnreadByte()
	switch {
	case unicode.IsDigit(rune(c)):
		return r.readInt()
	case c == ':':
		panic("not yet implemented")
	case c == '"':
		return r.readString()
	default:
		sym, err := r.readSymbol()
		if err != nil {
			return nil, err
		}
		if sym.NS == "" {
			switch sym.Name {
			case "true":
				return true, nil
			case "false":
				return false, nil
			case "nil":
				return nil, nil
			}
		}
		return sym, nil
	}
}

func (r GojureReader) readInt() (int, error) {
	bys := []byte{}
	c, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if c == '-' || c == '+' {
		bys = append(bys, c)
	} else {
		r.UnreadByte()
	}
	c, err = r.ReadByte()
	for err == nil && unicode.IsDigit(rune(c)) {
		bys = append(bys, c)
		c, err = r.ReadByte()
	}
	if err != nil && err != io.EOF {
		return 0, err
	} else if err == nil {
		r.UnreadByte()
	}
	return strconv.Atoi(string(bys))
}

func (r GojureReader) readString() (string, error) {
	bys := []byte{}
	quo, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	if quo != '"' {
		return "", errors.New("not a string.")
	}
	escaping := false
	c, err := r.ReadByte()
	for err == nil && (escaping || c != '"') {
		if c == '\n' {
			bys = append(bys, '\\', 'n')
		} else {
			bys = append(bys, c)
		}
		escaping = false
		if c == '\\' && !escaping {
			escaping = true
		}
		c, err = r.ReadByte()
	}
	if err != nil {
		return "", err
	}
	return string(bys), nil
}

func symbolChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		c == '*' || c == '+' || c == '!' || c == '-' || c == '_' || c == '?' || c == '/' || c == '='
}

func (r GojureReader) readSymbol() (lang.Symbol, error) {
	ret := lang.Symbol{}
	c, err := r.ReadByte()
	if err != nil {
		return ret, err
	}
	if !symbolChar(c) {
		return ret, errors.New("bad symbol, starting with rune " + string(c))
	}
	bys := []byte{}
	for err == nil && symbolChar(c) {
		if c == '/' {
			if ret.NS != "" {
				return ret, errors.New("bad symbol, more than one namespace separator.")
			}
			ret.NS = string(bys)
			bys = bys[:0]
		} else {
			bys = append(bys, c)
		}
		c, err = r.ReadByte()
	}
	if err != nil && err != io.EOF {
		return ret, err
	} else if err == nil {
		r.UnreadByte()
	}
	ret.Name = string(bys)
	return ret, nil
}

func (r GojureReader) readCompound(delim byte) ([]interface{}, error) {
	ret := []interface{}{}
	var next interface{}

	c, err := r.skipSpace()
	for err == nil && c != delim {
		r.UnreadByte()
		next, err = r.Read()
		if err == nil {
			ret = append(ret, next)
		} else {
			return ret, err
		}
		c, err = r.skipSpace()
	}

	return ret, err
}

func (r GojureReader) skipSpace() (byte, error) {
	c, err := r.ReadByte()
	for err == nil && unicode.IsSpace(rune(c)) {
		c, err = r.ReadByte()
	}
	return c, err
}
