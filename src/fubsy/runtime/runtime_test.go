package runtime

import (
	"testing"
	"fubsy/testutils"
	"fubsy/dsl"
)

func Test_Runtime_assign(t *testing.T) {
	// AST for a = "foo"
	node := dsl.NewASTAssignment(
		stubtoken{"a"}, stringnode("foo"))
	rt := &Runtime{}
	ns := NewNamespace()

	rt.assign(node, ns)
	expect := FuString("foo")
	assertIn(t, ns, "a", expect)
}

// evaluate simple expressions (no operators)
func Test_Runtime_evaluate_simple(t *testing.T) {
	// the expression "meep" evaluates to the string "meep"
	snode := stringnode("meep")
	rt := &Runtime{}
	expect := FuString("meep")
	assertEvaluateOK(t, rt, expect, snode)

	// the expression foo evaluates to the string "meep" if foo is set
	// to that string in the local namespace...
	ns := NewNamespace()
	rt.locals = ns
	ns["foo"] = expect
	nnode := dsl.NewASTName(stubtoken{"foo"})
	assertEvaluateOK(t, rt, expect, nnode)

	// ... and to an error if the variable is not defined
	nnode = dsl.NewASTName(stubtoken{"boo"})
	assertEvaluateFail(t, rt, "undefined variable 'boo'", nnode)
}

func stringnode(value string) *dsl.ASTString {
	// NewASTString takes a token, which comes quoted
	value = "\"" + value + "\""
	return dsl.NewASTString(stubtoken{value})
}

func assertIn(t *testing.T, ns Namespace, name string, expect FuObject) {
	if actual, ok := ns[name]; ok {
		if actual != expect {
			t.Errorf("expected %#v, but got %#v", expect, actual)
		}
	} else {
		t.Errorf("expected to find name '%s' in namespace", name)
	}
}

func assertEvaluateOK(
	t *testing.T,
	rt *Runtime,
	expect FuObject,
	input dsl.ASTExpression) {

	obj, err := rt.evaluate(input)
	testutils.AssertNoError(t, err)
	if expect != obj {
		t.Errorf("expected %#v, but got %#v", expect, obj)
	}
}

func assertEvaluateFail(
	t *testing.T,
	rt *Runtime,
	expecterr string,
	input dsl.ASTExpression) {

	obj, err := rt.evaluate(input)
	testutils.AssertError(t, expecterr, err)
	if obj != nil {
		t.Errorf("expected obj == nil, but got %#v", obj)
	}
}

type stubtoken struct {
	text string
}

func (tok stubtoken) Location() dsl.Location {
	return dsl.Location{}
}

func (tok stubtoken) Text() string {
	return tok.text
}

