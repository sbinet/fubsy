 // -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

%{
package dsl

import (
	"io"
	"io/ioutil"
	"fmt"
)

type Scanner struct {
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

func NewScanner(filename string, input []byte) *Scanner {
	return &Scanner{filename: filename, input: input}
}

func NewFileScanner(filename string, reader io.Reader) (*Scanner, error) {
	input, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return NewScanner(filename, input), nil
}

func (self *Scanner) nextchar() {
	if self.cur > 0 {
		self.buf = append(self.buf, self.cur)
	}
	if self.cur == '\n' {
		// increment lineno after consuming the newline (self.cur will
		// be replaced by the next char momentarily), but before we
		// produce the EOL token (see maybeeol() for compensation)
		self.lineno++
	}
	if self.pos >= len(self.input) {
		//fmt.Printf("nextchar: at eof (post=%d, lineno=%d)\n",
		//	self.pos, self.lineno)
		self.cur = 0				// not a very good eof marker!
	} else {
		//fmt.Printf("nextchar: pos=%d, lineno=%d: advancing from %#v to %#v\n",
		//	self.pos, self.lineno, string(self.cur), string(self.input[self.pos]))
		self.prev = self.cur
		self.cur = self.input[self.pos]
		self.pos++
	}
}

func (self *Scanner) scan() {
	//fmt.Printf("scan: input=>%s<\n", self.input)

	// start conditions (lexical modes)
	const (
		SC_INITIAL = iota
		SC_INLINE
		SC_FILELIST
		maxsc
	)

	condition := SC_INITIAL
	var inline []byte
	var inlinestart int

	// current line contains only whitespace or comments (no tokens)
	blank := true

	// nesting level inside () (to allow newline in function call args)
	depth := 0

	begin := func(cond int) {
		if cond < SC_INITIAL || cond >= maxsc {
			panic(fmt.Sprintf("invalid start condition: %d", cond))
		}
		condition = cond
	}
	addtok := func(filename string, lineno int, token int, text []byte) {
		tt := toktext{filename, lineno, token, string(text)}
		self.tokens = append(self.tokens, tt)
		blank = false
	}
	checkbad := func() {
		if len(self.badtext) > 0 {
			//fmt.Printf("line %d: invalid token: %s\n", self.badlineno, self.badtext)
			addtok(self.filename, self.badlineno, BADTOKEN, self.badtext)
			self.badtext = self.badtext[:0]
		}
	}
	tokfound := func(token int) {
		checkbad()
		// fmt.Printf("token: %d %#v (prev=%#v, cur=%#v, pos=%d, lineno=%d)\n",
		// 	token, string(self.buf), string(self.prev), string(self.cur),
		// 	self.pos, self.lineno)
		addtok(self.filename, self.lineno, token, self.buf)
	}
	badchar := func() {
		//fmt.Printf("badchar: >%c<\n", self.buf[0])
		// setting badlineno on every badchar is valid as long
		// as badtext does not span lines
		self.badlineno = self.lineno
		self.badtext = append(self.badtext, self.buf[0])
	}
	maybeeol := func() {
		// Don't want the parser to worry about blank/whitespace/comment
		// lines, or about newlines inside (say) function args. Thus, only
		// report "significant" newlines -- ie. newlines not inside parens,
		// at the end of a line that contains at least one other token.
		checkbad()
		if !blank && depth == 0 {
			// decrement lineno to compensate for lineno++ in nextchar()
			addtok(self.filename, self.lineno-1, EOL, self.buf)
		}
		blank = true
	}

	startinline := func() {
		begin(SC_INLINE)
		inlinestart = self.lineno
		tokfound(L3BRACE)
	}
	stopinline := func() {
		addtok(self.filename, inlinestart, INLINE, inline)
		inline = inline[:0]
		begin(SC_INITIAL)
		tokfound(R3BRACE)
	}
	inlinecontent := func() {
		inline = append(inline, self.buf...)
	}

	startfilelist := func() {
		begin(SC_FILELIST)
		tokfound('<')
	}
	filepattern := func() {
		tokfound(FILEPATTERN)
	}
	stopfilelist := func() {
		begin(SC_INITIAL)
		tokfound('>')
	}

	self.lineno = 1
	self.nextchar()

%}

%yyc self.cur
%yyn self.nextchar()
%yyb self.prev == 0 || self.prev == '\n'
%yyt condition

/* start conditions (lexical modes) */
%x SC_INLINE
%x SC_FILELIST

%%

 // truncate current token text before every scan cycle
 self.buf = self.buf[:0]

<*>\0						goto eof
[ \t]+						checkbad()
\n							maybeeol()
\#.*						checkbad()

"import"					tokfound(IMPORT)
"plugin"					tokfound(PLUGIN)

[a-zA-Z_][a-zA-Z_0-9]*		tokfound(NAME)
\{							tokfound('{')
\}							tokfound('}')
\(							depth++; tokfound('(')
\)							depth--; tokfound(')')
\.							tokfound('.')
\,							tokfound(',')
=							tokfound('=')
\+							tokfound('+')
\"[^\"]+\"					tokfound(QSTRING)

\{\{\{						startinline()
<SC_INLINE>\}\}\}			stopinline()
<SC_INLINE>.*				inlinecontent()
<SC_INLINE>\n				inlinecontent()

\<							startfilelist()
<SC_FILELIST>[^ \t\n\>]+	filepattern()
<SC_FILELIST>[ \t\n]		// ignore
<SC_FILELIST>\>				stopfilelist()

.							badchar()

%%

eof:
	if self.tokens[len(self.tokens)-1].token != EOL {
		// synthetic EOL so the parser only has to worry about EOL as a
		// statement terminator (not EOF too)
		tokfound(EOL)
	}
	// special EOF token to improve syntax error reporting
	tokfound(EOF)
}
