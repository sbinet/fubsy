 // -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

 // Copyright © 2012-2013, Greg Ward. All rights reserved.
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

	// filename and line offset table *shared by all tokens and AST
	// nodes* parsed from this file
	fileinfo *fileinfo

	condition int				// current lexical mode (start condition)
	pos int						// current offset into input
	startpos int				// start of current token
	eof bool

	// current line contains only whitespace or comments (no tokens)
	blank bool

	// nesting level inside () (to allow newline inside parens)
	depth int

	// content of current inline plugin (when condition == SC_INLINE)
	inline []byte

	// argh: all of these bytes should probably be runes
	prev byte
	cur byte
	buf []byte

	badstart int				// start of current badtext (offset into input)
	badtext []byte				// sequence of bad chars

	tokens []token
}

// start conditions (lexical modes)
const (
	SC_INITIAL = iota
	SC_INLINE
	SC_FILEFINDER
	maxsc
)

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

func (self *Scanner) begin(cond int) {
 	if cond < SC_INITIAL || cond >= maxsc {
		panic(fmt.Sprintf("invalid start condition: %d", cond))
	}
	self.condition = cond
}

func (self *Scanner) addtok(
	filename string, startpos *int, tokid int, text []byte) {
	location := newFileLocation(self.fileinfo)
	location.start = *startpos
	location.end = self.pos - 1
	tt := token{location, tokid, string(text)}
	// fmt.Printf("token: %d %#v @ [%d:%d] (prev=%#v, cur=%#v, pos=%d)\n",
	// 	tokid, string(text), location.start, location.end,
	// 	string(self.prev), string(self.cur), self.pos)
	self.tokens = append(self.tokens, tt)
	self.blank = false
	*startpos = self.pos - 1
	self.badstart = self.pos - 1
}

func (self *Scanner) checkbad() {
	if len(self.badtext) > 0 {
		// fmt.Printf("invalid token: %#v (startpos=%d, pos=%d)\n",
		// 	self.badtext, self.startpos, self.pos)
		self.addtok(self.filename, &self.badstart, BADTOKEN, self.badtext)
		self.tokens[len(self.tokens)-1].location.end = self.startpos
		self.badtext = self.badtext[:0]
	}
}

func (self *Scanner) skip() {
	self.startpos = self.pos - 1
	self.badstart = self.startpos
}

func (self *Scanner) tokfound(token int) {
	self.checkbad()
	self.addtok(self.filename, &self.startpos, token, self.buf)
}

func (self *Scanner) badchar() {
	//fmt.Printf("badchar: >%c<\n", self.buf[0])
	self.badtext = append(self.badtext, self.buf[0])
	self.startpos++
}

func (self *Scanner) maybeeol() {
	// Don't want the parser to worry about blank/whitespace/comment
	// lines, or about newlines inside (say) function args. Thus, only
	// report "significant" newlines -- ie. newlines not inside parens,
	// at the end of a line that contains at least one other token.
	self.checkbad()
	if !self.blank && self.depth == 0 {
		self.addtok(self.filename, &self.startpos, EOL, self.buf)
	}
	self.blank = true
	self.startpos = self.pos - 1
	self.badstart = self.startpos
}

func (self *Scanner) startinline() {
	self.begin(SC_INLINE)
	self.tokfound(L3BRACE)
}

func (self *Scanner) finishinline() {
	self.addtok(self.filename, &self.startpos, INLINE, self.inline)
	self.inline = self.inline[:0]
}

func (self *Scanner) stopinline() {
	self.finishinline()

	// because we don't emit the INLINE token until we have seen
	// the closing "}}}", the token's location.end points to
	// the end of "}}}"
	self.tokens[len(self.tokens)-1].location.end -= 3

	// self.addtok() helpfully messes up self.startpos (similar reason)
	self.startpos -= 3

	self.begin(SC_INITIAL)
	self.tokfound(R3BRACE)
}

func (self *Scanner) inlinecontent() {
	self.inline = append(self.inline, self.buf...)
}

func (self *Scanner) startfilefinder() {
	self.begin(SC_FILEFINDER)
	self.tokfound('<')
}

func (self *Scanner) filepattern() {
	self.tokfound(FILEPATTERN)
}

func (self *Scanner) stopfilefinder() {
	self.begin(SC_INITIAL)
	self.startpos = self.pos - 2
	self.tokfound('>')
}

func (self *Scanner) scan() {
	// fmt.Printf("scan: input=>%#v<\n", string(self.input))

	self.fileinfo = &fileinfo{self.filename, []int {0}}
	self.blank = true

	self.nextchar()
%}

%yyc self.cur
%yyn self.nextchar()
%yyb self.prev == 0 || self.prev == '\n'
%yyt self.condition

/* start conditions (lexical modes) */
%x SC_INLINE
%x SC_FILEFINDER

%%

 // truncate current token text before every scan cycle
 self.buf = self.buf[:0]

<*>\0						goto eof
[ \t]+						self.checkbad(); self.skip()
\n							self.maybeeol()
\#.*						self.checkbad(); self.skip()

"import"					self.tokfound(IMPORT)
"plugin"					self.tokfound(PLUGIN)

[a-zA-Z_][a-zA-Z_0-9]*		self.tokfound(NAME)
\{							self.tokfound('{')
\}							self.tokfound('}')
\(							self.depth++; self.tokfound('(')
\)							self.depth--; self.tokfound(')')
\[							self.depth++; self.tokfound('[')
\]							self.depth--; self.tokfound(']')
\.							self.tokfound('.')
\,							self.tokfound(',')
=							self.tokfound('=')
\+							self.tokfound('+')
:							self.tokfound(':')
\"[^\"]+\"					self.tokfound(QSTRING)

\{\{\{						self.startinline()
<SC_INLINE>\}\}\}			self.stopinline()
<SC_INLINE>.*				self.inlinecontent()
<SC_INLINE>\n				self.inlinecontent()

\<							self.startfilefinder()
<SC_FILEFINDER>[^ \t\n\>]+	self.filepattern()
<SC_FILEFINDER>[ \t\n]		self.skip()
<SC_FILEFINDER>\>			self.stopfilefinder()

.							self.badchar()

%%

eof:
	if self.condition == SC_INLINE {
		self.finishinline()
	}

	if self.tokens[len(self.tokens)-1].id != EOL {
		// synthetic EOL so the parser only has to worry about EOL as a
		// statement terminator (not EOF too)
		self.tokfound(EOL)
	}
	// special EOF token to improve syntax error reporting
	self.tokfound(EOF)
}
