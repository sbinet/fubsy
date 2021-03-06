package mock

import (
	"errors"
	"github.com/stretchrcom/testify/assert"
	"testing"
)

/*
	Test objects
*/

// ExampleInterface represents an example interface.
type ExampleInterface interface {
	TheExampleMethod(a, b, c int) (int, error)
}

// TestExampleImplementation is a test implementation of ExampleInterface
type TestExampleImplementation struct {
	Mock
}

func (i *TestExampleImplementation) TheExampleMethod(a, b, c int) (int, error) {
	args := i.Mock.Called(a, b, c)
	return args.Int(0), args.Error(1)
}

func (i *TestExampleImplementation) TheExampleMethod2(yesorno bool) {
	i.Mock.Called(yesorno)
}

/*
	Mock
*/

func Test_Mock_On(t *testing.T) {

	// make a test impl object
	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	assert.Equal(t, mockedService.Mock.On("TheExampleMethod"), &mockedService.Mock)
	assert.Equal(t, "TheExampleMethod", mockedService.Mock.onMethodName)

}

func Test_Mock_On_WithArgs(t *testing.T) {

	// make a test impl object
	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	assert.Equal(t, mockedService.Mock.On("TheExampleMethod", 1, 2, 3), &mockedService.Mock)
	assert.Equal(t, "TheExampleMethod", mockedService.Mock.onMethodName)
	assert.Equal(t, 1, mockedService.Mock.onMethodArguments[0])
	assert.Equal(t, 2, mockedService.Mock.onMethodArguments[1])
	assert.Equal(t, 3, mockedService.Mock.onMethodArguments[2])

}

func Test_Mock_Return(t *testing.T) {

	// make a test impl object
	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	assert.Equal(t, mockedService.Mock.On("TheExampleMethod", "A", "B", true).Return(1, "two", true), &mockedService.Mock)

	// ensure the call was created
	if assert.Equal(t, 1, len(mockedService.Mock.ExpectedCalls)) {
		call := mockedService.Mock.ExpectedCalls[0]

		assert.Equal(t, "TheExampleMethod", call.Method)
		assert.Equal(t, "A", call.Arguments[0])
		assert.Equal(t, "B", call.Arguments[1])
		assert.Equal(t, true, call.Arguments[2])
		assert.Equal(t, 1, call.ReturnArguments[0])
		assert.Equal(t, "two", call.ReturnArguments[1])
		assert.Equal(t, true, call.ReturnArguments[2])

	}

}

func Test_Mock_Return_Nothing(t *testing.T) {

	// make a test impl object
	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	assert.Equal(t, mockedService.Mock.On("TheExampleMethod", "A", "B", true).Return(), &mockedService.Mock)

	// ensure the call was created
	if assert.Equal(t, 1, len(mockedService.Mock.ExpectedCalls)) {
		call := mockedService.Mock.ExpectedCalls[0]

		assert.Equal(t, "TheExampleMethod", call.Method)
		assert.Equal(t, "A", call.Arguments[0])
		assert.Equal(t, "B", call.Arguments[1])
		assert.Equal(t, true, call.Arguments[2])
		assert.Equal(t, 0, len(call.ReturnArguments))

	}

}

func Test_Mock_findExpectedCall(t *testing.T) {

	m := new(Mock)
	m.On("One", 1).Return("one")
	m.On("Two", 2).Return("two")
	m.On("Two", 3).Return("three")

	f, c := m.findExpectedCall("Two", 3)

	if assert.True(t, f) {
		if assert.NotNil(t, c) {
			assert.Equal(t, "Two", c.Method)
			assert.Equal(t, 3, c.Arguments[0])
			assert.Equal(t, "three", c.ReturnArguments[0])
		}
	}

}

func Test_callString(t *testing.T) {

	assert.Equal(t, `Method(int,bool,string)`, callString("Method", []interface{}{1, true, "something"}, false))

}

func Test_Mock_Called(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	mockedService.Mock.On("Test_Mock_Called", 1, 2, 3).Return(5, "6", true)

	returnArguments := mockedService.Mock.Called(1, 2, 3)

	if assert.Equal(t, 1, len(mockedService.Mock.Calls)) {
		assert.Equal(t, "Test_Mock_Called", mockedService.Mock.Calls[0].Method)
		assert.Equal(t, 1, mockedService.Mock.Calls[0].Arguments[0])
		assert.Equal(t, 2, mockedService.Mock.Calls[0].Arguments[1])
		assert.Equal(t, 3, mockedService.Mock.Calls[0].Arguments[2])
	}

	if assert.Equal(t, 3, len(returnArguments)) {
		assert.Equal(t, 5, returnArguments[0])
		assert.Equal(t, "6", returnArguments[1])
		assert.Equal(t, true, returnArguments[2])
	}

}

func Test_Mock_Called_Unexpected(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	// make sure it panics if no expectation was made
	assert.Panics(t, func() {
		mockedService.Mock.Called(1, 2, 3)
	}, "Calling unexpected method should panic")

}

func Test_AssertExpectationsForObjects_Helper(t *testing.T) {

	var mockedService1 *TestExampleImplementation = new(TestExampleImplementation)
	var mockedService2 *TestExampleImplementation = new(TestExampleImplementation)
	var mockedService3 *TestExampleImplementation = new(TestExampleImplementation)

	mockedService1.Mock.On("Test_AssertExpectationsForObjects_Helper", 1).Return()
	mockedService2.Mock.On("Test_AssertExpectationsForObjects_Helper", 2).Return()
	mockedService3.Mock.On("Test_AssertExpectationsForObjects_Helper", 3).Return()

	mockedService1.Called(1)
	mockedService2.Called(2)
	mockedService3.Called(3)

	assert.True(t, AssertExpectationsForObjects(t, mockedService1.Mock, mockedService2.Mock, mockedService3.Mock))

}

func Test_AssertExpectationsForObjects_Helper_Failed(t *testing.T) {

	var mockedService1 *TestExampleImplementation = new(TestExampleImplementation)
	var mockedService2 *TestExampleImplementation = new(TestExampleImplementation)
	var mockedService3 *TestExampleImplementation = new(TestExampleImplementation)

	mockedService1.Mock.On("Test_AssertExpectationsForObjects_Helper_Failed", 1).Return()
	mockedService2.Mock.On("Test_AssertExpectationsForObjects_Helper_Failed", 2).Return()
	mockedService3.Mock.On("Test_AssertExpectationsForObjects_Helper_Failed", 3).Return()

	mockedService1.Called(1)
	mockedService3.Called(3)

	tt := new(testing.T)
	assert.False(t, AssertExpectationsForObjects(tt, mockedService1.Mock, mockedService2.Mock, mockedService3.Mock))

}

func Test_Mock_AssertExpectations(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	mockedService.Mock.On("Test_Mock_AssertExpectations", 1, 2, 3).Return(5, 6, 7)

	tt := new(testing.T)
	assert.False(t, mockedService.AssertExpectations(tt))

	// make the call now
	mockedService.Mock.Called(1, 2, 3)

	// now assert expectations
	assert.True(t, mockedService.AssertExpectations(tt))

}

func Test_Mock_AssertNumberOfCalls(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	mockedService.Mock.On("Test_Mock_AssertNumberOfCalls", 1, 2, 3).Return(5, 6, 7)

	mockedService.Mock.Called(1, 2, 3)
	assert.True(t, mockedService.AssertNumberOfCalls(t, "Test_Mock_AssertNumberOfCalls", 1))

	mockedService.Mock.Called(1, 2, 3)
	assert.True(t, mockedService.AssertNumberOfCalls(t, "Test_Mock_AssertNumberOfCalls", 2))

}

func Test_Mock_AssertCalled(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	mockedService.Mock.On("Test_Mock_AssertCalled", 1, 2, 3).Return(5, 6, 7)

	mockedService.Mock.Called(1, 2, 3)

	assert.True(t, mockedService.AssertCalled(t, "Test_Mock_AssertCalled", 1, 2, 3))

}

func Test_Mock_AssertCalled_WithArguments(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	mockedService.Mock.On("Test_Mock_AssertCalled_WithArguments", 1, 2, 3).Return(5, 6, 7)

	mockedService.Mock.Called(1, 2, 3)

	tt := new(testing.T)
	assert.True(t, mockedService.AssertCalled(tt, "Test_Mock_AssertCalled_WithArguments", 1, 2, 3))
	assert.False(t, mockedService.AssertCalled(tt, "Test_Mock_AssertCalled_WithArguments", 2, 3, 4))

}

func Test_Mock_AssertNotCalled(t *testing.T) {

	var mockedService *TestExampleImplementation = new(TestExampleImplementation)

	mockedService.Mock.On("Test_Mock_AssertNotCalled", 1, 2, 3).Return(5, 6, 7)

	mockedService.Mock.Called(1, 2, 3)

	assert.True(t, mockedService.AssertNotCalled(t, "Test_Mock_NotCalled"))

}

/*
	Arguments helper methods
*/
func Test_Arguments_Get(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}

	assert.Equal(t, "string", args.Get(0).(string))
	assert.Equal(t, 123, args.Get(1).(int))
	assert.Equal(t, true, args.Get(2).(bool))

}

func Test_Arguments_Is(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}

	assert.True(t, args.Is("string", 123, true))
	assert.False(t, args.Is("wrong", 456, false))

}

func Test_Arguments_Diff(t *testing.T) {

	var args Arguments = []interface{}{"Hello World", 123, true}
	var diff string
	var count int
	diff, count = args.Diff([]interface{}{"Hello World", 456, "false"})

	assert.Equal(t, 2, count)
	assert.Contains(t, diff, `%!s(int=456) != %!s(int=123)`)
	assert.Contains(t, diff, `false != %!s(bool=true)`)

}

func Test_Arguments_Diff_DifferentNumberOfArgs(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	var diff string
	var count int
	diff, count = args.Diff([]interface{}{"string", 456, "false", "extra"})

	assert.Equal(t, 3, count)
	assert.Contains(t, diff, `extra != (Missing)`)

}

func Test_Arguments_Diff_WithAnythingArgument(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	var count int
	_, count = args.Diff([]interface{}{"string", Anything, true})

	assert.Equal(t, 0, count)

}

func Test_Arguments_Diff_WithAnythingArgument_InActualToo(t *testing.T) {

	var args Arguments = []interface{}{"string", Anything, true}
	var count int
	_, count = args.Diff([]interface{}{"string", 123, true})

	assert.Equal(t, 0, count)

}

func Test_Arguments_Diff_WithAnythingOfTypeArgument(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	var count int
	_, count = args.Diff([]interface{}{"string", AnythingOfType("int"), true})

	assert.Equal(t, 0, count)

}

func Test_Arguments_Diff_WithAnythingOfTypeArgument_Failing(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	var count int
	var diff string
	diff, count = args.Diff([]interface{}{"string", AnythingOfType("string"), true})

	assert.Equal(t, 1, count)
	assert.Contains(t, diff, `string != type int - %!s(int=123)`)

}

func Test_Arguments_Assert(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}

	assert.True(t, args.Assert(t, "string", 123, true))

}

func Test_Arguments_String_Representation(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	assert.Equal(t, `string,int,bool`, args.String())

}

func Test_Arguments_String(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	assert.Equal(t, "string", args.String(0))

}

func Test_Arguments_Error(t *testing.T) {

	var err error = errors.New("An Error")
	var args Arguments = []interface{}{"string", 123, true, err}
	assert.Equal(t, err, args.Error(3))

}

func Test_Arguments_Error_Nil(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true, nil}
	assert.Equal(t, nil, args.Error(3))

}

func Test_Arguments_Int(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	assert.Equal(t, 123, args.Int(1))

}

func Test_Arguments_Bool(t *testing.T) {

	var args Arguments = []interface{}{"string", 123, true}
	assert.Equal(t, true, args.Bool(2))

}
