// lexical scanner for fubsy DSL

package fubsy

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"strings"
	"regexp"
)

type Token struct {
	filename string
	line int
	id int
	value string
}

const (
	tokQSTRING = iota
	tok3LBRACE
	tok3RBRACE
	tokLBRACKET
	tokRBRACKET
	tokSPACE
	tokNEWLINE
	numTokens
)

func (self Token) String() string {
	return fmt.Sprintf("{%s:%d: tok %d: %#v}",
		self.filename, self.line, self.id, self.value)
}

func Scan(filename string, infile io.Reader) ([]Token, error) {
	lexre := setupScanner()
	content, err := ioutil.ReadAll(infile)
	if err != nil {
		return nil, err
	}
	lineno := 1
	prev := 0
	remaining := content
	tokens := make([]Token, 0)
	for len(remaining) > 0 {
		//fmt.Printf("scanning from prev=%d >%s<\n", prev, remaining)
		match := lexre.FindSubmatchIndex(remaining)
		if len(match) < 2 {
			break
		}
		match = match[2:]		// don't care about the whole match
		var tokid int
		var tokvalue string
		for tokid = 0; tokid < numTokens; tokid++ {
			start := match[tokid * 2]
			if start == -1 { // did not match this subgroup (tokid)
				continue
			}
			if start > 0 {
				fmt.Fprintf(os.Stderr,
					"%s:%d: unrecognized token: %#v\n",
					filename, lineno, string(remaining[0:start]))
			}
			end := match[tokid * 2 + 1]
			tokvalue = string(remaining[start:end])
			//fmt.Printf("matched [%d:%d]: tokid %d: >%s<\n",
			//	start+prev, end+prev, tokid, tokvalue);
			prev += end			// offset into content
			remaining = remaining[end:]
			break
		}
		if tokid == tokSPACE {	// eat non-newline whitespace
			continue
		}
		token := Token{
			filename: filename,
			line: lineno,
			id: tokid,
			value: tokvalue,
		}
		tokens = append(tokens, token)
		if tokid == tokNEWLINE {
			lineno++
		}
	}
	return tokens, nil
}


func setupScanner() *regexp.Regexp {
	patterns := make([]string, numTokens)
	patterns[tokQSTRING]  = `\"[^\"]*\"`
	patterns[tok3LBRACE]  = `\{\{\{`
	patterns[tok3RBRACE]  = `\}\}\}`
	patterns[tokLBRACKET] = `\[`
	patterns[tokRBRACKET] = `\]`
	patterns[tokSPACE]    = `[ \t]+`
	patterns[tokNEWLINE]  = `\n`
	fmt.Println("patterns =", patterns)

	bigpattern := "(" + strings.Join(patterns, ")|(") + ")"
	fmt.Println("bigpattern =", bigpattern)

	return regexp.MustCompile(bigpattern)
}
