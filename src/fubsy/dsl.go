package fubsy

type AST interface {
	ListPlugins() []string
}

func Parse(filename string) (AST, error) {
	if filename == "bogus" {
		return nil, ParseError{"that's a bogus filename"}
	}

	ast_ := ast{plugins: []string{"foo", "bar", "baz"}}
	return ast_, nil
}

type ParseError struct {
	msg string
}

func (self ParseError) Error() string {
	return self.msg
}

type ast struct {
	plugins []string
}

func (self ast) ListPlugins() []string {
	return self.plugins
}
