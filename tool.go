// Package tool Useful general purpose tool
package tool

import (
	"errors"
	"fmt"
	stdlog "log"
	"reflect"
	"runtime"
	"strings"
	"time"

	"golang.org/x/exp/constraints"

	"github.com/iamwavecut/tool/safetool"
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

	catchableError struct {
		error
	}
)

// Unwrap Returns the wrapped error
func (e catchableError) Unwrap() error { return e.error }

// tooloLog Package level logger, defaults to log.Default()
var tooloLog = &logger{l: stdlog.Default()}

// Console Prints %+v of arguments, great to debug stuff
func Console(obj ...interface{}) {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		tooloLog.LogError(errors.New("unable to get caller information"))
		return
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		tooloLog.LogError(errors.New("unable to get function information"))
		return
	}

	displayFilePath, err := safetool.GetRelativePath(file)
	if err != nil {
		tooloLog.LogError(err)
		displayFilePath = file
	}

	pkg := strings.Split(fn.Name(), "/")
	pkgName := strings.Join(pkg[0:len(pkg)-1], "/") + "/"
	pkgName += strings.Split(pkg[len(pkg)-1:][0], ".")[0]

	prefix := fmt.Sprintf("[%s:%s:%d]>", pkgName, displayFilePath, line)
	tooloLog.LogDeep(append([]interface{}{prefix}, obj...)...)
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
func Must(err error, verbose ...bool) {
	if err != nil {
		if len(verbose) > 0 && verbose[0] {
			tooloLog.LogError(err)
		}
		panic(catchableError{err})
	}
}

// Return Ignores errors, returns value.
func Return[T any](val T, _ error) T {
	return val
}

// MultiMute Ignores errors, returns slice of results.
func MultiMute[T any](a ...T) []T {
	if len(a) == 0 {
		return nil
	}
	val := reflect.ValueOf(a[len(a)-1])
	lastInterface := val.Interface()
	if reflect.TypeOf(lastInterface).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		a = a[:len(a)-1]
	}
	if len(a) == 0 {
		return nil
	}
	return a
}

// MustReturn Tolerates no errors, returns value.
func MustReturn[T any](val T, err error) T {
	Must(err)
	return val
}

// Err Returns the last argument if it is an error, otherwise nil
func Err(args ...any) error {
	var err error
	if len(args) > 0 {
		err, _ = args[len(args)-1].(error)
	}
	return err
}

// Catch Recovers from panic and callbacks with error
// If error is not catchableError, it will panic again
// May be used as defer, coupled with MustReturn or Must, to override named return values
//
// Usage:
//
//	  func example() (val *http.Request, err error) {
//		defer tool.Catch(func(caught error) {
//			err = caught
//	 	})
//
//		val = tool.MustReturn(funcThatReturnsValAndErr()) // <- this will be caught if err!=nil
//		panic(errors.New("some error")) // <- this will not be caught
//		return
//	}
func Catch(fn func(err error)) {
	e := recover()
	if e == nil {
		return
	}

	if iamError, ok := e.(catchableError); ok {
		fn(iamError.Unwrap())
		return
	}
	panic(e)
}

// RandInt Return a random number in specified range.
func RandInt[num constraints.Signed](min, max num) num {
	val, err := safetool.RandInt(min, max)
	Must(err)
	return val
}

// Ptr Return a pointer for any passed object
func Ptr[T any](n T) *T {
	return safetool.Ptr(n)
}

// In Checks if element is in a slice
// Deprecated: Use slices.Contains instead
func In[T comparable](needle T, haystack ...T) bool {
	return safetool.In(needle, haystack...)
}

// RetryFunc Re-runs function if error returned
func RetryFunc[num constraints.Signed](attempts num, sleep time.Duration, f func() error) error {
	var retryErr error
	for {
		retryErr = f()

		if !Try(retryErr) {
			return nil
		}
		if attempts == 0 {
			break
		}
		attempts--
		time.Sleep(sleep)
		tooloLog.LogError(retryErr, "retrying after error")
	}
	return retryErr
}

// Recoverer Recovers job from panic, if maxPanics<0 then infinitely
func Recoverer[num constraints.Integer](maxPanics num, f func(), jobID ...string) (recovErr error) {
	defer func() {
		if err := recover(); err != nil {
			panicErr := fmt.Errorf(`job %spanics with message: %s, %s`, strings.Join(jobID, " ")+" ", err, identifyPanic())
			tooloLog.LogError(panicErr)

			if maxPanics != 0 {
				recovErr = Recoverer(maxPanics-1, f, jobID...)
			}
			if recovErr == nil {
				recovErr = panicErr
			}
			return
		}
	}()
	f()
	return
}

// Jsonify Returns Varchar implementation of the serialized value, returns empty on error
func Jsonify(s any) safetool.Varchar {
	val, err := safetool.Jsonify(s)
	if Try(err, true) {
		return ""
	}
	return val
}

// Objectify Unmarshalls value to the target pointer value
func Objectify[T ~[]byte | ~string](in T, target any) bool {
	err := safetool.Objectify(in, target)
	return !Try(err, true)
}

// Strtr Replaces all old string occurrences with new string in subject
func Strtr(subject string, oldToNew map[string]string) string {
	return safetool.Strtr(subject, oldToNew)
}

// NonZero Returns first non-zero value or zero value if all values are zero
func NonZero[T comparable](ts ...T) T {
	var zeroValue T
	if len(ts) == 0 {
		return zeroValue
	}
	for _, t := range ts {
		if t == zeroValue {
			continue
		}
		return t
	}
	return zeroValue
}

// IsZero Checks if value is zero
func IsZero[T comparable](v T) bool {
	var zero T
	return v == zero
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
		relPath, err := safetool.GetRelativePath(file)
		if err != nil {
			tooloLog.LogError(err)
			relPath = file
		}
		return fmt.Sprintf("%v:%v", relPath, line)
	}

	return fmt.Sprintf("pc:%x", pc)
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
	var buf strings.Builder
	for _, subj := range obj {
		buf.WriteString(fmt.Sprintf("%+v ", subj))
	}
	str := buf.String()[:buf.Len()-1]
	str = strings.ReplaceAll(strings.ReplaceAll(str, "\r", "\\r"), "\n", "\\n")
	l.l.Println(str)
}

// LogError Loose function to log error
func (l *logger) LogError(err error, msgs ...string) {
	if l.l == nil {
		return
	}
	if isrus, ok := l.l.(LogRus); ok {
		isrus.WithError(err).Errorln(strings.Join(msgs, ": "))
		return
	}
	if len(msgs) > 0 {
		msgs = append(msgs, "")
	}
	l.l.Println(errors.New(strings.Join(msgs, ": ") + err.Error()))
}

// PanicOnError Loose function to panic with error
func (l *logger) PanicOnError(err error, msgs ...string) {
	if l.l == nil {
		return
	}
	l.LogError(err, msgs...)
	panic(err)
}

func ExecTemplate(templateText string, templateVars any) string {
	val, err := safetool.ExecTemplate(templateText, templateVars)
	if Try(err) {
		return ""
	}
	return val
}

// ConvertSlice Return a new slice as `[]dstTypedValue.(type)` cast from the `srcSlice`
func ConvertSlice[T any, Y any](srcSlice []T, destTypedValue Y) []Y {
	val, err := safetool.ConvertSlice(srcSlice, destTypedValue)
	if err != nil {
		panic(fmt.Errorf("ConvertSlice failed: %w", err))
	}
	return val
}
