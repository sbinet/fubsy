// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"bytes"
	"fubsy/testutils"
	"github.com/stretchrcom/testify/assert"
	"testing"
)

func Test_ParseString(t *testing.T) {
	script := `
import foo.bar
bleep {
a
}`
	ast, err := ParseString("test", script)

	expect := &ASTRoot{children: []ASTNode{
		&ASTImport{plugin: []string{"foo", "bar"}},
		&ASTPhase{
			name: "bleep",
			children: []ASTNode{
				&ASTName{name: "a"}}}}}
	assert.Equal(t, 0, len(err))
	assertASTEqual(t, expect, ast)
}

// now with a syntax error
func Test_ParseString_invalid(t *testing.T) {
	script := `
import foo,bar
bleep {
}
`
	ast, err := ParseString("test", script)
	assert.Nil(t, ast)
	expect := "test:2: syntax error (near ',')"
	assertOneError(t, expect, err)
}

func TestParse_valid_1(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// dead simple: a single top-level element
	script := `
main {
<meep>

}
`
	fn := testutils.Mkfile(tmpdir, "valid_1.fubsy", script)
	expect := &ASTRoot{children: []ASTNode{
		&ASTPhase{name: "main", children: []ASTNode{
			&ASTFileFinder{patterns: []string{"meep"}}}}}}
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))
	assertASTEqual(t, expect, ast)
}

func TestParse_valid_sequence(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// sequence of top-level children
	script := `
main {
"boo"
}
plugin foo {{{o'malley & friends
}}}
blob {
 "meep"
 }`
	fn := testutils.Mkfile(tmpdir, "valid_2.fubsy", script)
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect := &ASTRoot{children: []ASTNode{
		&ASTPhase{
			name:     "main",
			children: []ASTNode{&ASTString{value: "boo"}}},
		&ASTInline{
			lang: "foo", content: "o'malley & friends\n"},
		&ASTPhase{
			name:     "blob",
			children: []ASTNode{&ASTString{value: "meep"}}},
	}}
	assertASTEqual(t, expect, ast)
}

func TestParse_internal_newlines(t *testing.T) {
	// newlines in a function call are invisible to the parser
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	script := `
main {
  x(
a.b
    )
}
`
	fn := testutils.Mkfile(tmpdir, "newlines.fubsy", script)
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect := &ASTRoot{
		children: []ASTNode{
			&ASTPhase{
				name: "main",
				children: []ASTNode{
					&ASTFunctionCall{
						function: &ASTName{name: "x"},
						args: []ASTExpression{
							&ASTSelection{
								container: &ASTName{name: "a"},
								member:    "b",
							}}}},
			}}}
	assertASTEqual(t, expect, ast)
}

func TestParse_invalid_1(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// invalid: no closing rbracket
	script := `
main{
"borf"
`
	fn := testutils.Mkfile(tmpdir, "invalid_1.fubsy", script)
	_, err := Parse(fn)
	expect := fn + ":4: syntax error (near EOF)"
	assertOneError(t, expect, err)
}

func TestParse_invalid_2(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// invalid: bad token
	script := `
main{
 *&! "whizz"
}`
	fn := testutils.Mkfile(tmpdir, "invalid_2.fubsy", script)
	_, err := Parse(fn)
	expect := fn + ":3: syntax error (near *&!)"
	assertOneError(t, expect, err)
	reset()
}

func assertOneError(t *testing.T, expect string, actual []error) {
	if len(actual) < 1 {
		t.Error("expected at least one error")
	} else if actual[0].Error() != expect {
		t.Errorf("expected error message\n%s\nbut got\n%s",
			expect, actual[0].Error())
	}
}

// this one tries to exercise many token types and grammar rules
func TestParse_omnibus_1(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	script := `
# start with a comment
import foo
import foo.bar.baz

     
# blank lines are OK!
plugin funky {{{
any ol' crap! "bring it on, 
dude" ...
}}}
main {
  a   =("foo") + b
  c=(d.e)  ()
x.y.z
  <
    lib1/*.c
    lib2/**/*.c
  >
}
`
	fn := testutils.Mkfile(tmpdir, "omnibus_1.fubsy", script)
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect :=
		`ASTRoot {
  ASTImport[foo]
  ASTImport[foo.bar.baz]
  ASTInline[funky] {{{
    any ol' crap! "bring it on, 
    dude" ...
  }}}
  ASTPhase[main] {
    ASTAssignment[a]
      ASTAdd
      op1:
        ASTString[foo]
      op2:
        ASTName[b]
    ASTAssignment[c]
      ASTFunctionCall[d.e] (0 args)
    ASTSelection[x.y: z]
    ASTFileFinder[lib1/*.c lib2/**/*.c]
  }
}
`
	var actual_ bytes.Buffer
	ast.Dump(&actual_, "")
	actual := actual_.String()
	if expect != actual {
		t.Errorf("expected AST:\n%s\nbut got:\n%s", expect, actual)
	}
}

func TestParse_omnibus_2(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	script := `
main {
  headers = <*.h>

  a + b : <*.c> + headers {
    x = a
    "cc $x"
    f(a, b)
  }
}
`
	fn := testutils.Mkfile(tmpdir, "omnibus_2.fubsy", script)
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect :=
		`ASTRoot {
  ASTPhase[main] {
    ASTAssignment[headers]
      ASTFileFinder[*.h]
    ASTBuildRule {
    targets:
      ASTAdd
      op1:
        ASTName[a]
      op2:
        ASTName[b]
    sources:
      ASTAdd
      op1:
        ASTFileFinder[*.c]
      op2:
        ASTName[headers]
    actions:
      ASTAssignment[x]
        ASTName[a]
      ASTString[cc $x]
      ASTFunctionCall[f] (2 args)
        ASTName[a]
        ASTName[b]
    }
  }
}
`
	var actual_ bytes.Buffer
	ast.Dump(&actual_, "")
	actual := actual_.String()
	if expect != actual {
		t.Errorf("expected AST:\n%s\nbut got:\n%s", expect, actual)
	}
}
