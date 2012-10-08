// lexical scanner for fubsy DSL

package fubsy

import (
	"fmt"
	"strings"
	"io"
	"io/ioutil"
	"regexp"
)

type Token struct {
	filename string
	line int
	id int
	value string
}

func (self Token) String() string {
	return self.value
}

// implement ASTNode because lemon insists that terminals (tokens) and
// non-terminals (AST nodes) be of the same type
func (self Token) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%s{tok %d (%s:%d): %#v}",
		indent, self.id, self.filename, self.line, self.value)
}

func (self Token) Equal(other ASTNode) bool {
	panic("Token.Equal() not implemented")
}

type TokenDef struct {
	id int
	name string
	re *regexp.Regexp
}

func (self TokenDef) String() string {
	pattern := "<none>"
	if self.re != nil {
		pattern = self.re.String()
	}
	return fmt.Sprintf("{%s: %s}", self.name, pattern)
}

// len() of this slice MUST match number of tokens in the grammar
// (fugrammar_tokens.go)!
var tokenDefs = make([]*TokenDef, 3 + 1)

func init() {
	add := func(id int, name string, pattern string) {
		var re *regexp.Regexp
		if len(pattern) > 0 {
			re = regexp.MustCompile(pattern)
		}
		def := &TokenDef{id, name, re}
		tokenDefs[id] = def
	}

	// these MUST match the tokens emitted by lemon (fugrammar_tokens.go)!
	// (hmmmm: token 0 is implicitly EOF)
	add(0,        "eof",      "")
	add(LBRACKET, "lbracket", `\[`)
	add(QSTRING,  "qstring",  `\"[^\"]*\"`)
	add(RBRACKET, "rbracket", `\]`)

	// add("3lbrace", 	 `\{\{\{`)
	// add("3rbrace", 	 `\}\}\}`)
	// add("name",      `[a-zA-Z_][a-zA-Z0-9]*`)
	// add("space",     `[ \t]+`)
	// add("newline",   `\n`)
}

type BadToken struct {
	filename string
	line int
	badtext []byte
}

func (self BadToken) Error() string {
	return fmt.Sprintf("%s:%d: invalid token: %#v",
		self.filename, self.line, string(self.badtext))
}

type ScanErrors []error

func (self ScanErrors) Error() string {
	messages := make([]string, len(self))
	for i, err := range self {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "\n")
}

func Scan(filename string, infile io.Reader) ([]Token, error) {
	content, readerr := ioutil.ReadAll(infile)
	if readerr != nil {
		return nil, readerr
	}

	errs := make(ScanErrors, 0)

	lineno := 1
	remaining := content
	tokens := make([]Token, 0)
	for {
		leftmost := -1
		var tokvalue string
		var tokend int
		var tokdef *TokenDef
		for _, trydef := range tokenDefs {
			if trydef.re == nil { // EOF is a dummy token
				continue
			}
			match := trydef.re.FindIndex(remaining)
			if match == nil {	// nope, it's not this token
				continue
			}
			start := match[0]
			end := match[1]
			if leftmost > 0 && start > leftmost {
				// not good enough: hold out for an earlier match
				continue
			}
			if leftmost < 0 || start < leftmost {
				// this is the new leftmost match
				tokvalue = string(remaining[start:end])
				leftmost = start
				tokend = end
				tokdef = trydef
			}
			if start == 0 {
				// it won't get any better than this
				break
			}
		}
		if tokdef == nil {
			// no tokens matched: any remaining text is junk at EOF
			if len(remaining) > 0 {
				errs = append(errs,
					BadToken{filename, lineno, remaining})
			}
			// but in any case, we are at EOF
			break
		}
		if leftmost > 0 {
			// leftmost match wasn't leftmost enough
			errs = append(errs,
				BadToken{filename, lineno, remaining[0:leftmost]})
		}
		remaining = remaining[tokend:]

		//fmt.Printf("matched tokid %d: %#v\n", tokid, tokvalue)
		if tokdef.name == "space" {	// eat non-newline whitespace
			continue
		}
		token := Token{
			filename: filename,
			line: lineno,
			id: tokdef.id,
			value: tokvalue,
		}
		tokens = append(tokens, token)
		if tokdef.name == "newline" {
			lineno++
		}
	}

	if len(errs) == 0 {
		return tokens, nil
	}
	return tokens, errs
}
