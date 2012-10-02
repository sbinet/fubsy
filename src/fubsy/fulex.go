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
	content, err := ioutil.ReadAll(infile)
	if err != nil {
		return nil, err
	}

	regexes := setupScanner()
	lineno := 1
	remaining := content
	tokens := make([]Token, 0)
	for {
		leftmost := -1
		tokid := -1
		var tokvalue string
		var tokend int
		for i := 0; i < numTokens; i++ {
			match := regexes[i].FindIndex(remaining)
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
				tokid = i
			}
			if start == 0 {
				// it won't get any better than this
				break
			}

		}
		if tokid == -1 {
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


func setupScanner() []*regexp.Regexp {
	patterns := make([]string, numTokens)
	patterns[tokQSTRING]  = `\"[^\"]*\"`
	patterns[tok3LBRACE]  = `\{\{\{`
	patterns[tok3RBRACE]  = `\}\}\}`
	patterns[tokLBRACKET] = `\[`
	patterns[tokRBRACKET] = `\]`
	patterns[tokSPACE]    = `[ \t]+`
	patterns[tokNEWLINE]  = `\n`

	regexes := make([]*regexp.Regexp, numTokens)
	for i := 0; i < numTokens; i++ {
		regexes[i] = regexp.MustCompile(patterns[i])
	}
	return regexes
}
