%{
package fubsy

import (
	"io"
	"io/ioutil"
	//"fmt"
)

type Lexer struct {
	filename string
	input []byte
	pos int
	lineno int

	// argh: all of these bytes should probably be runes
	prev byte
	cur byte
	buf []byte

	badlineno int                       // start of current badtext
	badtext []byte                      // sequence of bad chars

	tokens []toktext
}

func NewLexer(filename string, input []byte) *Lexer {
	return &Lexer{filename: filename, input: input}
}

func NewFileLexer(filename string, reader io.Reader) (*Lexer, error) {
	input, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return NewLexer(filename, input), nil
}

func (self *Lexer) nextchar() {
	if self.cur > 0 {
		self.buf = append(self.buf, self.cur)
	}
	if self.pos >= len(self.input) {
		self.cur = 0				// not a very good eof marker!
	} else {
		if self.cur == '\n' {
			self.lineno += 1
		}
		self.prev = self.cur
		self.cur = self.input[self.pos]
		self.pos += 1
	}
}

func (self *Lexer) scan() {
	//fmt.Printf("scan: input=>%s<\n", self.input)

	checkbad := func() {
		if len(self.badtext) > 0 {
			//fmt.Printf("line %d: invalid token: %s\n", self.badlineno, self.badtext)
			tt := toktext{
				self.filename, self.badlineno, BADTOKEN, string(self.badtext)}
			self.tokens = append(self.tokens, tt)
			self.badtext = self.badtext[:0]
		}
	}
	tokfound := func(token int) {
		checkbad()
		//fmt.Printf("token: >%s<\n", self.buf)
		tt := toktext{self.filename, self.lineno, token, string(self.buf)}
		self.tokens = append(self.tokens, tt)
	}
	badchar := func() {
		//fmt.Printf("badchar: >%c<\n", self.buf[0])
		// setting badlineno on every badchar is valid as long
		// as badtext does not span lines
		self.badlineno = self.lineno
		self.badtext = append(self.badtext, self.buf[0])
	}

	self.lineno = 1
	self.nextchar()

%}

%yyc self.cur
%yyn self.nextchar()
%yyb self.prev == 0 || self.prev == '\n'
%yyt 0

%%

 // truncate current token text before every scan cycle
				self.buf = self.buf[:0]

[ \t\n]+		checkbad()
\[				tokfound('[')
\]				tokfound(']')
\"[^\"]+\"		tokfound(QSTRING)
.				badchar()

%%

	checkbad()
}
