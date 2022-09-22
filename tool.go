// Package tool Useful general purpose tool
package tool

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"math/big"
	"runtime"
	"strings"
	"time"

	"golang.org/x/exp/constraints"
)

type (
	StdLogger interface {
		Println(...any)
		Panicln(...any)
		Printf(string, ...any)
		Print(...any)
	}

	LogRus interface {
		StdLogger
		WithError(error) LogRus
		Errorln(...any)
	}

	logger struct {
		l StdLogger
	}

	Varchar string
)

// tooloLog Package level logger, defaults to log.Default()
var tooloLog *logger = &logger{l: stdlog.Default()}

// Console Prints %+v of arguments, great to debug stuff
func Console(obj ...any) {
	tooloLog.l.Print("> ")
	tooloLog.LogDeep(obj...)
}

// SetLogger Sets tool package logger, pass nil to disable logging
func SetLogger(l StdLogger) {
	tooloLog = &logger{l: l}
}

// Try Probes the error and returns bool, optionally logs the message.
func Try(err error, verbose ...bool) bool {
	if err != nil {
		if len(verbose) > 0 && verbose[0] {
			tooloLog.LogError(err)
		}
		return true
	}
	return false
}

// Must Tolerates no errors.
func Must(err error) {
	if nil != err {
		tooloLog.PanicOnError(err, "failed")
	}
}

// RandInt Return a random number in specified range.
func RandInt[num constraints.Signed](min, max num) num {
	bInt, err := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	Must(err)
	bInt = bInt.Add(bInt, big.NewInt(int64(min)))
	return num(bInt.Int64())
}

// Ptr Return a pointer for any passed object
func Ptr[T any](n T) *T {
	return &n
}

// In Checks if element is in a slice
func In[T comparable](needle T, haystack []T) bool {
	for _, spica := range haystack {
		if spica == needle {
			return true
		}
	}
	return false
}

// RetryFunc Re-runs function if error returned
func RetryFunc[num constraints.Signed](attempts num, sleep time.Duration, f func() error) error {
	var retryErr error
	for {
		retryErr = f()

		if !Try(retryErr) {
			return nil
		}
		if 0 == attempts {
			break
		}
		attempts--
		time.Sleep(sleep)
		tooloLog.LogError(retryErr, "retrying after error")
	}
	return retryErr
}

// Recoverer Recovers job from panic, if maxPanics<0 then infinitely
func Recoverer[num constraints.Integer](maxPanics num, f func(), jobID ...string) error {
	var recovErr error
	defer func() {
		if err := recover(); err != nil {
			msg := fmt.Sprintf(`Job "%s" panics with message: %s, %s`, strings.Join(jobID, " "), err, identifyPanic())
			tooloLog.LogError(errors.New(msg))

			if maxPanics == 0 {
				recovErr = errors.New("maximum recoveries exceeded")
			} else {
				recovErr = Recoverer(maxPanics-1, f, jobID...)
			}
			return
		}
	}()
	f()
	return recovErr
}

// Jsonify Returns Varchar implementation of the serialized value, returns empty on error
func Jsonify(s any) Varchar {
	b, err := json.Marshal(s)
	if Try(err, true) {
		return ""
	}
	return Varchar(b)
}

// Objectify Unmarshalls value to the target pointer value
func Objectify[T ~[]byte | ~string](in T, target any) bool {
	str := string(in)
	return !Try(json.Unmarshal([]byte(str), target), true)
}

// Strtr Replaces all old string occurrences with new string in subject
func Strtr(subject string, oldToNew map[string]string) string {
	if len(oldToNew) == 0 || len(subject) == 0 {
		return subject
	}
	for old, news := range oldToNew {
		subject = strings.ReplaceAll(subject, old, news)
	}
	return subject
}

// identifyPanic Helper function to get user-friendly call stack message.
func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

// Bytes Return Varchar as Bytes slice
func (s Varchar) Bytes() []byte {
	return []byte(s)
}

// String Return Varchar as string
func (s Varchar) String() string {
	return string(s)
}

func (s *Varchar) MarshalJSON() ([]byte, error) {
	return Jsonify(s.String()).Bytes(), nil
}

// Log Logs anything
func (l *logger) Log(msgs ...any) {
	if l.l == nil {
		return
	}
	l.l.Println(msgs)
}

// LogDeep Printf version to log objects deeply
func (l *logger) LogDeep(obj ...any) {
	if l.l == nil {
		return
	}
	lastIndex := len(obj) - 1
	for i, subj := range obj {
		l.l.Printf("%+v", subj)
		if lastIndex != i {
			l.l.Print(" ")
		}
	}
	l.l.Print("\n")
}

// LogError Loose function to log error
func (l *logger) LogError(err error, msgs ...string) {
	if l.l == nil {
		return
	}
	msg := strings.Join(msgs, ": ")
	if isrus, ok := l.l.(LogRus); ok {
		isrus.WithError(err).Errorln(msg)
		return
	}
	if len(msgs) > 0 {
		msgs = append(msgs, "") // add final colon
	}
	l.l.Println(errors.New(msg + err.Error()))
}

// PanicOnError Loose function to die with error
func (l *logger) PanicOnError(err error, msgs ...string) {
	if l.l == nil {
		return
	}
	l.LogError(err, msgs...)
	l.l.Panicln("PANIC reason ^^^")
}
