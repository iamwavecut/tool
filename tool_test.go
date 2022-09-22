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
	t.buf += fmt.Sprintln(a...)
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
		Console("123")
		s.Equal("> 123\n", testLog.buf)
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

func TestSuite(t *testing.T) {
	suite.Run(t, new(ToolTestSuite))
}
