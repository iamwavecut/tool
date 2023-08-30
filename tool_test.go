package tool

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"reflect"
	"strconv"
	"testing"
)

type (
	ToolTestSuite struct {
		suite.Suite
		StdLogger
	}
	testLogger struct {
		buf string
	}
)

func (t *testLogger) Println(a ...any) {
	t.buf += fmt.Sprintln(a...)
}

func (t *testLogger) Panicln(a ...any) {
	panic(a)
}

func (t *testLogger) Printf(s string, a ...any) {
	t.buf += fmt.Sprintf(s, a...)
}

func (t *testLogger) Print(a ...any) {
	t.buf += fmt.Sprint(a...)
}

var testLog = &testLogger{}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ToolTestSuite))
}

func (s *ToolTestSuite) SetupSuite() {
	SetLogger(testLog)
}

func (s *ToolTestSuite) SetupTest() {
	testLog.buf = ""
}

func (s *ToolTestSuite) TestIn() {
	s.Run("string", func() {
		s.True(In("hi", "oh", "hi", "there"))
		s.False(In("hi", "hello", "beautiful"))
	})
	s.Run("byte", func() {
		s.True(In(byte(2), []byte{1, 2, 3}...))
		s.Equal(false, In(byte(255), []byte{1, 2, 3}...))
	})
}

func (s *ToolTestSuite) TestConsole() {
	s.Run("1", func() {
		Console("123", "456", "789")
		s.Equal("[tool_test.go:65]> 123 456 789\n", testLog.buf)
	})
	s.Run("2", func() {
		testLog.buf = ""
		Console(struct{ int }{123})
		s.Equal("[tool_test.go:70]> {int:123}\n", testLog.buf)
	})
	s.Run("3", func() {
		testLog.buf = ""
		Console(nil)
		s.Equal("[tool_test.go:75]> <nil>\n", testLog.buf)
	})
}

func (s *ToolTestSuite) TestNonZero() {
	s.Run("string", func() {
		s.Equal("hi", NonZero("hi", "there"))
		s.Equal("there", NonZero("", "there"))
	})
	s.Run("int", func() {
		s.Equal(1, NonZero(1, 2))
		s.Equal(2, NonZero(0, 2))
	})
	type testStruct struct {
		i int
	}
	s.Run("struct", func() {
		s.Equal(testStruct{i: 2}, NonZero(testStruct{}, testStruct{i: 2}))
	})
}

func (s *ToolTestSuite) TestJsonify() {
	s.Run("string", func() {
		res := Jsonify([]string{"oh", "hi", "there"})
		s.NotEmpty(res.String())
		s.Equal(`["oh","hi","there"]`, res.String())
	})
	s.Run("bytes", func() {
		res := Jsonify([]string{"oh", "hi", "there"})
		s.NotEmpty(res.Bytes())
		s.Equal([]byte(`["oh","hi","there"]`), res.Bytes())
	})
	s.Run("invalid", func() {
		res := Jsonify(func() {})
		s.Empty(res)
	})
}

func (s *ToolTestSuite) TestObjectify() {
	s.Run("string", func() {
		out := map[string]string{}
		in := `{"key":"value"}`

		res := Objectify(in, &out)
		s.True(res)

		s.Equal(map[string]string{"key": "value"}, out)
	})
	s.Run("bytestring", func() {
		out := map[string]string{}
		in := []byte(`{"key":"value"}`)

		res := Objectify(in, &out)
		s.True(res)

		s.Equal(map[string]string{"key": "value"}, out)
	})
}

func (s *ToolTestSuite) TestRetryFunc() {
	s.Run("failure", func() {
		times := 5
		errorNum := 7
		res := RetryFunc(times, 0, func() error {
			if errorNum > 0 {
				return errors.New(strconv.Itoa(errorNum))
			}
			return nil
		})
		s.Error(res)
	})
	s.Run("success", func() {
		times := 5
		errorNum := 3
		res := RetryFunc(times, 0, func() error {
			if errorNum > 0 {
				errorNum--
				return errors.New(strconv.Itoa(errorNum))
			}
			return nil
		})
		s.NoError(res)
	})
}

func (s *ToolTestSuite) TestTry() {
	s.Run("failure", func() {
		s.False(Try(nil))
	})
	s.Run("success", func() {
		s.True(Try(fmt.Errorf("error")))
	})
	s.Run("failure verbose", func() {
		s.False(Try(nil, true))
		s.Empty(testLog.buf)
	})
	s.Run("success verbose", func() {
		s.True(Try(fmt.Errorf("verbose error"), true))
		s.Equal("verbose error\n", testLog.buf)
	})
}

func (s *ToolTestSuite) TestMust() {
	s.Run("failure", func() {
		s.NotPanics(func() {
			Must(nil)
		})
	})
	s.Run("success", func() {
		s.Panics(func() {
			Must(fmt.Errorf("error"))
		})
	})
}

// TestRandInt is non-deterministic and hollow, but it exists for the sake of the coverage
func (s *ToolTestSuite) TestRandInt() {
	s.Contains([]int{1, 2, 3, 4, 5}, RandInt(1, 5))
}

func (s *ToolTestSuite) TestPtr() {
	intPtr := Ptr(1)
	s.IsType(func() *int { i := 0; return &i }(), intPtr)

	strPtr := Ptr("test")
	s.IsType(func() *string { s := ""; return &s }(), strPtr)

	boolPtr := Ptr(true)
	s.IsType(func() *bool { s := true; return &s }(), boolPtr)
}

func (s *ToolTestSuite) TestRecoverer() {
	for _, tc := range []struct {
		name      string
		initial   int
		expected  int
		maxPanics int
		success   bool
	}{
		{name: "valid 0", initial: 0, expected: 1, maxPanics: 0, success: true},
		{name: "valid 1", initial: 0, expected: 1, maxPanics: 1, success: true},
		{name: "panic 0", maxPanics: 0, success: false},
		{name: "panic 10", maxPanics: 10, success: false},
	} {
		s.Run(tc.name, func() {
			recovers := 0
			if tc.success {
				s.NoError(
					Recoverer(tc.maxPanics, func() {
						tc.initial = tc.expected
					}, tc.name),
				)
				s.Equal(tc.expected, tc.initial)
			} else {
				s.Error(
					Recoverer(tc.maxPanics, func() {
						recovers++
						panic("test")
					}, tc.name),
				)
				s.Equal(tc.maxPanics, recovers-1)
			}
		})
	}

	s.NoError(
		Recoverer(0, func() {}),
	)
}

func (s *ToolTestSuite) TestStrtr() {
	in := "abcdef"
	expected := "rstxyz"

	actual := Strtr(in, map[string]string{
		"a":   "r",
		"b":   "s",
		"c":   "t",
		"def": "xyz",
	})
	s.Equal(expected, actual)
	s.Equal(in, Strtr(in, map[string]string{}))
	s.Equal(in, Strtr(in, map[string]string{"": "b"}))
	s.Empty(Strtr("", map[string]string{"a": "b"}))
	s.Empty(Strtr("", map[string]string{"": ""}))
	s.Equal(in, Strtr(in, map[string]string{"abc": "abc"}))
}

func (s *ToolTestSuite) TestIdentifyPanic() {
	s.NotPanics(func() { identifyPanic() })
}

func (s *ToolTestSuite) TestExecTemplate() {
	s.Run("simple", func() {
		s.Equal("hello world", ExecTemplate("hello {{.}}", "world"))
	})
	s.Run("complex", func() {
		s.Equal("hello world", ExecTemplate("hello {{.name}}", map[string]string{"name": "world"}))
	})
	s.Run("no map key (partial render)", func() {
		s.Equal("hello ", ExecTemplate("hello {{.name}}", map[string]string{}))
	})
	s.Run("struct", func() {
		type Name struct {
			Name string
		}
		s.Equal("hello world", ExecTemplate("hello {{.Name}}", Name{Name: "world"}))
	})
	s.Run("struct no field (error)", func() {
		type Name struct {
			Value string
		}
		s.Equal("", ExecTemplate("hello {{.Name}}", Name{Value: "world"}))
	})
	s.Run("empty", func() {
		s.Equal("", ExecTemplate("", "world"))
	})
}

func (s *ToolTestSuite) TestMuteMulti() {
	tests := []struct {
		name string
		in   []any
		want []any
	}{
		{
			name: "trailing error",
			in:   []any{1, 2, 3, errors.New("error")},
			want: []any{1, 2, 3},
		},
		{
			name: "no error",
			in:   []any{1, 2, 3},
			want: []any{1, 2, 3},
		},
		{
			name: "empty",
			in:   []any{},
			want: nil,
		},
		{
			name: "only error",
			in:   []any{errors.New("error")},
			want: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			res := MultiMute(tc.in...)
			s.Equal(tc.want, res)
		})
	}
}

func (s *ToolTestSuite) TestReturn() {
	// Define multiple scenarios for your test
	tests := []struct {
		name     string
		inputVal int
		inputErr error
	}{
		{
			name:     "error is nil",
			inputVal: 5,
			inputErr: nil,
		},
		{
			name:     "error is not nil",
			inputVal: 7,
			inputErr: errors.New("an error"),
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			result := Return(test.inputVal, test.inputErr)
			s.Equal(test.inputVal, result)
		})
	}
}

func (s *ToolTestSuite) TestMustReturn() {
	tests := []struct {
		name        string
		inputVal    int
		inputErr    error
		shouldPanic bool
	}{
		{
			name:        "When error is nil",
			inputVal:    5,
			inputErr:    nil,
			shouldPanic: false,
		},
		{
			name:        "When error is not nil",
			inputVal:    7,
			inputErr:    errors.New("an error"),
			shouldPanic: true,
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			if test.shouldPanic {
				s.Panics(func() { MustReturn(test.inputVal, test.inputErr) })
			} else {
				s.NotPanics(func() {
					result := MustReturn(test.inputVal, test.inputErr)
					s.Equal(test.inputVal, result)
				})
			}
		})
	}
}

func (s *ToolTestSuite) TestErr() {
	errExpected := errors.New("Some error")
	args := []any{"Hello", errExpected}
	err := Err(args...)
	s.NotNil(err)
	s.Equal(errExpected, err)

	args = []any{"Hello", "World"}
	err = Err(args...)
	s.Nil(err)

	args = []any{}
	err = Err(args...)
	s.Nil(err)
}

func (s *ToolTestSuite) TestCatch() {
	s.Run("catchable error", func() {
		defer func() {
			if r := recover(); r != nil {
				s.Fail("Not expected to pass panic", r)
			}
		}()

		defer Catch(func(caught error) {
			s.EqualError(caught, "catchable error")
		})

		panic(&catchableError{errors.New("catchable error")})
	})

	s.Run("uncatchable error", func() {
		defer func() {
			if r := recover(); r != nil {
				s.EqualError(r.(error), "uncatchable error")
			} else {
				s.Fail("Expected a panic")
			}
		}()

		defer Catch(func(caught error) {
			s.Fail("This should not be called")
		})

		panic(errors.New("uncatchable error"))
	})
}

func (s *ToolTestSuite) TestConvertSlice() {
	type testCase struct {
		Name           string
		Input          []int
		DestTypeValue  float64 // used only for its type
		ExpectedOutput []float64
		ShouldPanic    bool
	}

	testCases := []testCase{
		{
			Name:           "valid slice conversion from int to float64",
			Input:          []int{1, 2, 3},
			DestTypeValue:  float64(0),
			ExpectedOutput: []float64{1.0, 2.0, 3.0},
			ShouldPanic:    false,
		},
		{
			Name:           "empty slice conversion",
			Input:          []int{},
			DestTypeValue:  float64(0),
			ExpectedOutput: []float64{},
			ShouldPanic:    false,
		},
		{
			Name:           "nil slice conversion",
			Input:          nil,
			DestTypeValue:  float64(0),
			ExpectedOutput: nil,
			ShouldPanic:    false,
		},
	}

	for _, test := range testCases {
		s.Run(test.Name, func() {
			var result interface{}
			if test.ShouldPanic {
				s.Panics(func() {
					result = ConvertSlice(test.Input, test.DestTypeValue)
				})
			} else {
				result = ConvertSlice(test.Input, test.DestTypeValue)
				s.Equal(reflect.TypeOf(result).Kind(), reflect.Slice, "result should be a slice")
				if _, ok := result.([]float64); !ok {
					s.Fail("result should be a slice of float64")
				}
				s.Equal(len(result.([]float64)), len(test.Input), "result slice size should match input slice size")
				s.Equal(result, test.ExpectedOutput, "slice conversion not as expected")
			}
		})
	}

	// edge case: srcVal is a pointer
	s.Run("valid slice conversion from *int to float64", func() {
		input := []*int{Ptr(1), Ptr(2), Ptr(3)}
		destTypeValue := float64(0)
		expectedOutput := []float64{1.0, 2.0, 3.0}

		result := ConvertSlice(input, destTypeValue)
		s.Equal(reflect.TypeOf(result).Kind(), reflect.Slice, "result should be a slice")
		var resultInterface any = result
		if _, ok := resultInterface.([]float64); !ok {
			s.Fail("result should be a slice of float64")
		}
		s.Equal(len(resultInterface.([]float64)), len(input), "result slice size should match input slice size")
		s.Equal(result, expectedOutput, "slice conversion not as expected")
	})
}
