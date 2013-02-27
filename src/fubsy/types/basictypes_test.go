// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package types

import (
	"testing"
	//"fmt"

	"github.com/stretchrcom/testify/assert"
)

func Test_FuString_stringify(t *testing.T) {
	var s FuObject

	s = MakeFuString("hello")
	assert.Equal(t, "\"hello\"", s.String())
	assert.Equal(t, "hello", s.ValueString())
	assert.Equal(t, "hello", s.CommandString())

	s = MakeFuString("bip bop")
	assert.Equal(t, "\"bip bop\"", s.String())
	assert.Equal(t, "bip bop", s.ValueString())
	assert.Equal(t, "'bip bop'", s.CommandString())

	s = MakeFuString("don't start")
	assert.Equal(t, "\"don't start\"", s.String())
	assert.Equal(t, "don't start", s.ValueString())
	assert.Equal(t, "\"don't start\"", s.CommandString())
}

func Test_basictypes_Equal(t *testing.T) {
	s1 := MakeFuString("foo")
	s2 := MakeFuString("foo")
	s3 := MakeFuString("bar")
	l1 := MakeFuList(s1, s3)
	l2 := MakeFuList(s2, s3)
	l3 := MakeFuList(s3)

	// FuString.Equal is just like builtin string ==
	assert.True(t, s1.Equal(s1))
	assert.True(t, s1.Equal(s2))
	assert.False(t, s1.Equal(s3))

	// a FuString is never equal to a FuList
	assert.False(t, s1.Equal(l1))
	assert.False(t, s3.Equal(l3))
	assert.False(t, l3.Equal(s3))

	// FuList.Equal() compares list contents
	assert.True(t, l1.Equal(l1))
	assert.True(t, l1.Equal(l2))
	assert.False(t, l1.Equal(l3))
}

func Test_FuString_Add_strings(t *testing.T) {
	s1 := MakeFuString("hello")
	s2 := MakeFuString("world")
	var result FuObject
	var err error

	// s1 + s1
	result, err = s1.Add(s1)
	assert.Nil(t, err)
	assert.Equal(t, "hellohello", result.(FuString).value)

	// s1 + s2
	result, err = s1.Add(s2)
	assert.Nil(t, err)
	assert.Equal(t, "helloworld", result.(FuString).value)

	// s1 + s2 + s1 + s2
	// (equivalent to ((s1.Add(s2)).Add(s1)).Add(s2), except we have
	// to worry about error handling)
	result, err = s1.Add(s2)
	assert.Nil(t, err)
	result, err = result.Add(s1)
	assert.Nil(t, err)
	result, err = result.Add(s2)
	assert.Nil(t, err)
	assert.Equal(t, "helloworldhelloworld", result.(FuString).value)

	// neither s1 nor s2 is affected by all this adding
	assert.Equal(t, "hello", s1.value)
	assert.Equal(t, "world", s2.value)
}

func Test_FuString_Add_list(t *testing.T) {
	cmd := MakeFuString("ls")
	args := MakeStringList("-l", "-a", "foo")
	result, err := cmd.Add(args)
	assert.Nil(t, err)
	assert.Equal(t, `["ls", "-l", "-a", "foo"]`, result.String())
}

func Test_FuString_Lookup(t *testing.T) {
	// strings have no attributes
	s := MakeFuString("blah")
	val, ok := s.Lookup("foo")
	assert.Nil(t, val)
	assert.False(t, ok)
}

// this really just tests that I understand the regexp API
func Test_expand_re(t *testing.T) {
	s := "blah blah no matches here"
	assert.Nil(t, expand_re.FindStringSubmatchIndex(s))

	s = "here is a $variable reference"
	match := expand_re.FindStringSubmatchIndex(s)
	expect := []int{
		10, 19,
		11, 19,
		-1, -1}
	assert.Equal(t, expect, match)

	s = "and ${aNoTher_way} of putting it"
	match = expand_re.FindStringSubmatchIndex(s)
	expect = []int{
		4, 18,
		-1, -1,
		6, 17}
	assert.Equal(t, expect, match)
}

func Test_FuString_ActionExpand(t *testing.T) {
	ns := makeNamespace("foo", "hello", "meep", "blorf")
	input := MakeFuString("meep meep!")
	output, err := input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, input, output)

	input = MakeFuString("meep $foo blah")
	output, err = input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, "meep hello blah", output.ValueString())

	input = MakeFuString("hello ${foo} $meep")
	output, err = input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, "hello hello blorf", output.ValueString())

	ns.Assign("foo", nil)
	output, err = input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, "hello  blorf", output.ValueString())

	ns.Assign("foo", MakeFuString("ping$pong"))
	output, err = input.ActionExpand(ns, nil)
	assert.Equal(t, "undefined variable 'pong' in string", err.Error())
	assert.Nil(t, output)
}

func Test_FuString_ActionExpand_recursive(t *testing.T) {
	ns := makeNamespace(
		"CC", "/usr/bin/gcc",
		"sources", "$file",
		"file", "f1.c")
	expect := "/usr/bin/gcc -c f1.c"
	input := MakeFuString("$CC -c $sources")
	output, err := input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, expect, output.ValueString())

	// same thing, but now files is a list
	ns.Assign("files", MakeStringList("f1.c"))
	output, err = input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, expect, output.ValueString())
}

func Test_FuString_ActionExpand_cycle(t *testing.T) {
	ns := makeNamespace(
		"a", "x.$a.y")
	s := MakeFuString("oh hi it's $a")
	_, err := s.ActionExpand(ns, nil)
	assert.Equal(t, "cyclic variable reference: a -> a", err.Error())
}

func Test_FuList_stringify(t *testing.T) {
	var l FuObject

	l = MakeStringList("beep", "meep")
	assert.Equal(t, `["beep", "meep"]`, l.String())
	assert.Equal(t, `beep meep`, l.ValueString())
	assert.Equal(t, `beep meep`, l.CommandString())

	l = MakeStringList("beep", "", "meep")
	assert.Equal(t, `["beep", "", "meep"]`, l.String())
	assert.Equal(t, `beep  meep`, l.ValueString())
	assert.Equal(t, `beep '' meep`, l.CommandString())

	l = MakeStringList("foo", "*.c", "ding dong", "")
	assert.Equal(t, `["foo", "*.c", "ding dong", ""]`, l.String())
	assert.Equal(t, `foo *.c ding dong `, l.ValueString())
	assert.Equal(t, `foo '*.c' 'ding dong' ''`, l.CommandString())
}

func Test_FuList_Add_list(t *testing.T) {
	l1 := MakeStringList("foo", "bar")
	l2 := MakeStringList("qux")

	result, err := l1.Add(l2)
	expect := MakeStringList("foo", "bar", "qux")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)

	result, err = l2.Add(l1)
	expect = MakeStringList("qux", "foo", "bar")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Add_string(t *testing.T) {
	cmd := MakeStringList("ls", "-la")
	arg := MakeFuString("stuff/")

	result, err := cmd.Add(arg)
	expect := MakeStringList("ls", "-la", "stuff/")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_ActionExpand(t *testing.T) {
	ns := makeNamespace()
	input := MakeStringList("gob", "mob")
	output, err := input.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, input, output)
}

func Test_FuList_ActionExpand_cycle(t *testing.T) {
	ns := makeNamespace(
		"a", "b",
		"b", "a",
		"c", "it's a $list",
		"foo", "ok")
	list := MakeStringList("yo", "$foo", "$c", "$b")
	ns.Assign("list", list)

	_, err := list.ActionExpand(ns, nil)
	assert.Equal(t, "cyclic variable reference: c -> list -> c", err.Error())
}

func Test_ExpandString_cycle(t *testing.T) {
	ns := makeNamespace()
	ns.Assign("a", MakeFuString("aaa$b"))
	ns.Assign("b", MakeFuString("$d bbb$c"))
	ns.Assign("c", MakeFuString("${a}ccc"))
	ns.Assign("d", MakeFuString("no problem"))

	_, _, err := ExpandString("hello $a", ns, nil)
	assert.Equal(t, "cyclic variable reference: a -> b -> c -> a", err.Error())

	// we only detect and report the first cycle
	ns.Assign("c", MakeFuString("${b}ccc${a}"))
	_, _, err = ExpandString("hello $c", ns, nil)
	assert.Equal(t, "cyclic variable reference: c -> b -> c", err.Error())

	// same treatment mixing types
	ns.Assign("s", MakeFuString("list = $l"))
	ns.Assign("l", MakeStringList("foo", "string = $s", "bar"))
	_, _, err = ExpandString("${s}", ns, nil)
	assert.Equal(t, "cyclic variable reference: s -> l -> s", err.Error())
}

func Test_ShellQuote(t *testing.T) {
	assertquote := func(expect string, input string) {
		actual := ShellQuote(input)
		assert.Equal(t, expect, actual)
	}

	s := "helloworld.txt"
	assertquote(s, s)
	//assert.Equal(t, s, q)

	// first choice: use single quotes
	s = "hello world"
	assertquote("'hello world'", s)

	// second choice: string has single quotes, so use double quotes
	s = "mr. o'reilly has $10"
	assertquote("\"mr. o'reilly has \\$10\"", s)

	// worst case: string has single and double quotes, so give up
	// on quotes and just use backslash for everything
	s = "mr. o'reilly said \"I have $10!\""
	assertquote("mr.\\ o\\'reilly\\ said\\ \\\"I\\ have\\ \\$10!\\\"", s)
}

func makeNamespace(keyval ...string) Namespace {
	ns := NewValueMap()
	for i := 0; i < len(keyval); i += 2 {
		ns[keyval[i]] = MakeFuString(keyval[i+1])
	}
	return ns
}
