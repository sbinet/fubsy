package types

import (
	"testing"
	//"fmt"
	"github.com/stretchrcom/testify/assert"
)

func Test_FuString_String(t *testing.T) {
	s := FuString("bip bop")
	assert.Equal(t, "bip bop", s.String())
}

func Test_FuString_CommandString(t *testing.T) {
	s := FuString("don't start")
	assert.Equal(t, "\"don't start\"", s.CommandString())
}

func Test_basictypes_Equal(t *testing.T) {
	s1 := FuString("foo")
	s2 := FuString("foo")
	s3 := FuString("bar")
	l1 := FuList([]FuObject {s1, s3})
	l2 := FuList([]FuObject {s2, s3})
	l3 := FuList([]FuObject {s3})

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
	s1 := FuString("hello")
	s2 := FuString("world")
	var result FuObject
	var err error

	// s1 + s1
	result, err = s1.Add(s1)
	assert.Nil(t, err)
	assert.Equal(t, "hellohello", result.(FuString))

	// s1 + s2
	result, err = s1.Add(s2)
	assert.Nil(t, err)
	assert.Equal(t, "helloworld", result.(FuString))

	// s1 + s2 + s1 + s2
	// (equivalent to ((s1.Add(s2)).Add(s1)).Add(s2), except we have
	// to worry about error handling)
	result, err = s1.Add(s2)
	assert.Nil(t, err)
	result, err = result.Add(s1)
	assert.Nil(t, err)
	result, err = result.Add(s2)
	assert.Nil(t, err)
	assert.Equal(t, "helloworldhelloworld", result.(FuString))

	// neither s1 nor s2 is affected by all this adding
	assert.Equal(t, "hello", string(s1))
	assert.Equal(t, "world", string(s2))
}

func Test_FuString_Add_list(t *testing.T) {
	cmd := FuString("ls")
	args := makeFuList("-l", "-a", "foo")
	result, err := cmd.Add(args)
	assert.Nil(t, err)
	assert.Equal(t, "[ls,-l,-a,foo]", result.String())
}

// this really just tests that I understand the regexp API
func Test_expand_re(t *testing.T) {
	s := "blah blah no matches here"
	assert.Nil(t, expand_re.FindStringSubmatchIndex(s))

	s = "here is a $variable reference"
	match := expand_re.FindStringSubmatchIndex(s)
	expect := []int {
		10, 19,
		11, 19,
		-1, -1}
	assert.Equal(t, expect, match)

	s = "and ${aNoTher_way} of putting it"
	match = expand_re.FindStringSubmatchIndex(s)
	expect = []int {
		4, 18,
		-1, -1,
		6, 17}
	assert.Equal(t, expect, match)
}

func Test_FuString_Expand(t *testing.T) {
	ns := makeNamespace("foo", "hello", "meep", "blorf")
	input := FuString("meep meep!")
	output, err := input.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, input, output)

	input = FuString("meep $foo blah")
	output, err = input.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, "meep hello blah", output.String())

	input = FuString("hello ${foo} $meep")
	output, err = input.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, "hello hello blorf", output.String())
}

func Test_FuString_Expand_recursive(t *testing.T) {
	ns := makeNamespace(
		"CC", "/usr/bin/gcc",
		"sources", "$file",
		"file", "f1.c")
	expect := "/usr/bin/gcc -c f1.c"
	input := FuString("$CC -c $sources")
	output, err := input.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, expect, output.String())

	// same thing, but now files is a list
	ns.Assign("files", FuList([]FuObject {FuString("f1.c")}))
	output, err = input.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, expect, output.String())
}

func Test_FuList_String(t *testing.T) {
	l := makeFuList("beep", "meep")
	assert.Equal(t, "[beep,meep]", l.String())

	l = makeFuList("beep", "", "meep")
	assert.Equal(t, "[beep,,meep]", l.String())
}

func Test_FuList_CommandString(t *testing.T) {
	l := makeFuList("foo", "*.c", "ding dong", "")
	assert.Equal(t, "foo '*.c' 'ding dong' ''", l.CommandString())
}

func Test_FuList_Add_list(t *testing.T) {
	l1 := makeFuList("foo", "bar")
	l2 := makeFuList("qux")

	result, err := l1.Add(l2)
	expect := makeFuList("foo", "bar", "qux")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)

	result, err = l2.Add(l1)
	expect = makeFuList("qux", "foo", "bar")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Add_string(t *testing.T) {
	cmd := makeFuList("ls", "-la")
	arg := FuString("stuff/")

	result, err := cmd.Add(arg)
	expect := makeFuList("ls", "-la", "stuff/")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Expand(t *testing.T) {
	ns := makeNamespace()
	input := makeFuList("gob", "mob")
	output, err := input.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, input, output)
}

func Test_shellQuote(t *testing.T) {
	assertquote := func(expect string, input string) {
		actual := shellQuote(input)
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
		ns[keyval[i]] = FuString(keyval[i+1])
	}
	return ns
}
