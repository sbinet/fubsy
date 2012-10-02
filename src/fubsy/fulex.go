// lexical scanner for fubsy DSL

package fubsy

import (
	"fmt"
	"os"
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
	return fmt.Sprintf("{%s:%d: tok %d: %#v}",
		self.filename, self.line, self.id, self.value)
}

type TokenDef struct {
	id int
	name string
	re *regexp.Regexp
}

func (self TokenDef) String() string {
	return fmt.Sprintf("{%s: %s}", self.name, self.re.String())
}

var tokenDefs []*TokenDef

func init() {
	add := func(name string, pattern string) {
		def := &TokenDef{len(tokenDefs), name, regexp.MustCompile(pattern)}
		tokenDefs = append(tokenDefs, def)
	}

	add("qstring", 	 `\"[^\"]*\"`)
	add("3lbrace", 	 `\{\{\{`)
	add("3rbrace", 	 `\}\}\}`)
	add("lbracket",  `\[`)
	add("rbracket",  `\]`)
	add("space",     `[ \t]+`)
	add("newline",   `\n`)
}

func Scan(filename string, infile io.Reader) ([]Token, error) {
	content, err := ioutil.ReadAll(infile)
	if err != nil {
		return nil, err
	}

	lineno := 1
	remaining := content
	tokens := make([]Token, 0)
	for {
		leftmost := -1
		var tokvalue string
		var tokend int
		var tokdef *TokenDef
		for _, trydef := range tokenDefs {
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
			// no token regexes matched: must be at EOF
			break
		}
		if leftmost > 0 {
			// leftmost match wasn't leftmost enough
			fmt.Fprintf(os.Stderr,
				"%s:%d: unrecognized token: %#v\n",
				filename, lineno, string(remaining[0:leftmost]))
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

	return tokens, nil
}
