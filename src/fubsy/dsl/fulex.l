 // -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

 // Copyright © 2012, Greg Ward. All rights reserved.
 // Use of this source code is governed by a BSD-style license that can
 // be found in the LICENSE.txt file.

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

	pos int						// current offset into input
	startpos int				// start of current token
	eof bool

	// filename and line offset table *shared by all tokens and AST
	// nodes* parsed from this file
	fileinfo *fileinfo

	// argh: all of these bytes should probably be runes
	prev byte
	cur byte
	buf []byte

	badstart int				// start of current badtext (offset into input)
	badtext []byte				// sequence of bad chars

	tokens []token
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
	if self.eof {
		return
	}
	if self.cur > 0 {
		self.buf = append(self.buf, self.cur)
	}
	if self.cur == '\n' {
		// self.pos already points to the start of the next line
		// fmt.Printf("nextchar: newline at pos %d, append %d to lineoffsets\n",
		// 	self.pos-1, self.pos)
		self.fileinfo.lineoffsets = append(self.fileinfo.lineoffsets, self.pos)
	}
	if self.pos >= len(self.input) {
		// fmt.Printf("nextchar: at eof (pos=%d)\n", self.pos)
		self.cur = 0				// not a very good eof marker!
		self.fileinfo.lineoffsets = append(self.fileinfo.lineoffsets, self.pos)
		self.pos++
		self.eof = true
	} else {
		// fmt.Printf("nextchar: pos=%d: advancing from %#v to %#v\n",
		// 	self.pos, string(self.cur), string(self.input[self.pos]))
		self.prev = self.cur
		self.cur = self.input[self.pos]
		self.pos++
	}
}

func (self *Scanner) scan() {
	// fmt.Printf("scan: input=>%#v<\n", string(self.input))

	// start conditions (lexical modes)
	const (
		SC_INITIAL = iota
		SC_INLINE
		SC_FILELIST
		maxsc
	)

	self.fileinfo = &fileinfo{self.filename, []int {0}}

	condition := SC_INITIAL
	var inline []byte
	//var inlinestart int

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
	addtok := func(filename string, startpos *int, tokid int, text []byte) {
		location := newLocation(self.fileinfo)
		location.start = *startpos
		location.end = self.pos - 1
		tt := token{location, tokid, string(text)}
		// fmt.Printf("token: %d %#v @ [%d:%d] (prev=%#v, cur=%#v, pos=%d)\n",
		// 	tokid, string(text), location.start, location.end,
		// 	string(self.prev), string(self.cur), self.pos)
		self.tokens = append(self.tokens, tt)
		blank = false
		*startpos = self.pos - 1
		self.badstart = self.pos - 1
	}
	checkbad := func() {
		if len(self.badtext) > 0 {
			// fmt.Printf("invalid token: %#v (startpos=%d, pos=%d)\n",
			// 	self.badtext, self.startpos, self.pos)
			addtok(self.filename, &self.badstart, BADTOKEN, self.badtext)
			self.tokens[len(self.tokens)-1].location.end = self.startpos
			self.badtext = self.badtext[:0]
		}
	}
	skip := func() {
		self.startpos = self.pos - 1
		self.badstart = self.startpos
	}
	tokfound := func(token int) {
		checkbad()
		addtok(self.filename, &self.startpos, token, self.buf)
	}
	badchar := func() {
		//fmt.Printf("badchar: >%c<\n", self.buf[0])
		self.badtext = append(self.badtext, self.buf[0])
		self.startpos++
	}
	maybeeol := func() {
		// Don't want the parser to worry about blank/whitespace/comment
		// lines, or about newlines inside (say) function args. Thus, only
		// report "significant" newlines -- ie. newlines not inside parens,
		// at the end of a line that contains at least one other token.
		checkbad()
		if !blank && depth == 0 {
			addtok(self.filename, &self.startpos, EOL, self.buf)
		}
		blank = true
		self.startpos = self.pos - 1
		self.badstart = self.startpos
	}

	startinline := func() {
		begin(SC_INLINE)
		//inlinestart = self.pos
		tokfound(L3BRACE)
	}
	finishinline := func() {
		addtok(self.filename, &self.startpos, INLINE, inline)
		inline = inline[:0]
	}
	stopinline := func() {
		finishinline()

		// because we don't emit the INLINE token until we have seen
		// the closing "}}}", the token's location.end points to
		// the end of "}}}"
		self.tokens[len(self.tokens)-1].location.end -= 3

		// addtok() helpfully messes up self.startpos (similar reason)
		self.startpos -= 3

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
		self.startpos = self.pos - 2
		tokfound('>')
	}

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
[ \t]+						checkbad(); skip()
\n							maybeeol()
\#.*						checkbad(); skip()

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
:							tokfound(':')
\"[^\"]+\"					tokfound(QSTRING)

\{\{\{						startinline()
<SC_INLINE>\}\}\}			stopinline()
<SC_INLINE>.*				inlinecontent()
<SC_INLINE>\n				inlinecontent()

\<							startfilelist()
<SC_FILELIST>[^ \t\n\>]+	filepattern()
<SC_FILELIST>[ \t\n]		skip()
<SC_FILELIST>\>				stopfilelist()

.							badchar()

%%

eof:
	if condition == SC_INLINE {
		finishinline()
	}

	if self.tokens[len(self.tokens)-1].id != EOL {
		// synthetic EOL so the parser only has to worry about EOL as a
		// statement terminator (not EOF too)
		tokfound(EOL)
	}
	// special EOF token to improve syntax error reporting
	tokfound(EOF)
}
