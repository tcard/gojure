package lang

type Keyword string

type Symbol struct {
	NS   string
	Name string
}

func (s Symbol) String() string {
	if s.NS != "" {
		return s.NS + "/" + s.Name
	}
	return s.Name
}
