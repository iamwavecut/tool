package tool

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type (
	ToolTestSuite struct {
		suite.Suite
		l StdLogger
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

func (s *ToolTestSuite) SetupSuite() {
	SetLogger(testLog)
}

func (s *ToolTestSuite) SetupTest() {
	testLog.buf = ""
}

func (s *ToolTestSuite) TestIn() {
	s.Run("string", func() {
		s.Equal(true, In("hi", []string{"oh", "hi", "there"}))
		s.Equal(false, In("hi", []string{"hello", "beautiful"}))
	})
	s.Run("byte", func() {
		s.Equal(true, In(byte(2), []byte{1, 2, 3}))
		s.Equal(false, In(byte(255), []byte{1, 2, 3}))
	})
}

func (s *ToolTestSuite) TestConsole() {
	s.Run("1", func() {
		Console("123", "456", "789")
		s.Equal("> 123 456 789\n", testLog.buf)
	})
	testLog.buf = ""
	s.Run("2", func() {
		Console(struct{ int }{123})
		s.Equal("> {int:123}\n", testLog.buf)
	})
	testLog.buf = ""
	s.Run("3", func() {
		Console(nil)
		s.Equal("> <nil>\n", testLog.buf)
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

func TestSuite(t *testing.T) {
	suite.Run(t, new(ToolTestSuite))
}
