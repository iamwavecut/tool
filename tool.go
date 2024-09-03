// Package tool Useful general purpose tool
package tool

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"math/big"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"text/template"
	"time"

	"golang.org/x/exp/slices"

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

	catchableError struct {
		error
	}
)

// Unwrap Returns the wrapped error
func (e catchableError) Unwrap() error { return e.error }

// tooloLog Package level logger, defaults to log.Default()
var tooloLog = &logger{l: stdlog.Default()}

func getRelativePath(filePath string) string {
	relPath, err := filepath.Rel(filepath.Dir(findRootCaller()), filePath)
	if err != nil {
		return filePath // return the original file path if error
	}
	return relPath
}

// Console Prints %+v of arguments, great to debug stuff
func Console(obj ...interface{}) {
	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		tooloLog.LogError(errors.New("unable to get caller information"))
		return
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		tooloLog.LogError(errors.New("unable to get function information"))
		return
	}
	pkg := strings.Split(fn.Name(), "/")
	pkgName := strings.Join(pkg[0:len(pkg)-1], "/") + "/"
	pkgName += strings.Split(pkg[len(pkg)-1:][0], ".")[0]

	prefix := fmt.Sprintf("[%s:%d]>", pkgName, line)
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

	if iamError, ok := e.(*catchableError); ok {
		fn(iamError.Unwrap())
		return
	}
	panic(e)
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
func In[T comparable](needle T, haystack ...T) bool {
	return slices.Contains(haystack, needle)
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
func Jsonify(s any) Varchar {
	b, err := json.Marshal(s)
	if Try(err, true) {
		return ""
	}
	return Varchar(b)
}

// Objectify Unmarshalls value to the target pointer value
func Objectify[T ~[]byte | ~string](in T, target any) bool {
	return !Try(json.Unmarshal([]byte(in), target), true)
}

// Strtr Replaces all old string occurrences with new string in subject
func Strtr(subject string, oldToNew map[string]string) string {
	if len(oldToNew) == 0 || len(subject) == 0 {
		return subject
	}
	for old, news := range oldToNew {
		if old == "" || old == news {
			continue
		}
		subject = strings.ReplaceAll(subject, old, news)
	}
	return subject
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
	if len(s.Bytes()) == 0 {
		return s.Bytes(), nil
	}
	if res := Jsonify(s.Bytes()).Bytes(); len(res) > 0 {
		return res, nil
	}
	return nil, fmt.Errorf("failed to marshal varchar")
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
		msgs = append(msgs, "") // add final colon
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
	tpl, err := template.New("ez").Parse(templateText)
	tpl.Option("missingkey=zero")
	if Try(err) {
		return ""
	}
	var buf strings.Builder
	err = tpl.Execute(&buf, templateVars)
	if Try(err) {
		return ""
	}
	return buf.String()
}

// ConvertSlice Return a new slice as `[]dstTypedValue.(type)` cast from the `srcSlice`
func ConvertSlice[T any, Y any](srcSlice []T, destTypedValue Y) []Y {
	srcReflectType := reflect.TypeOf(srcSlice)
	if srcReflectType.Kind() != reflect.Slice {
		panic("srcSlice is not a slice")
	}
	if srcSlice == nil {
		return nil
	} else if len(srcSlice) == 0 {
		return []Y{}
	}
	destType := reflect.TypeOf(destTypedValue)
	destSlice := reflect.MakeSlice(reflect.SliceOf(destType), len(srcSlice), len(srcSlice))
	for i := range srcSlice {
		srcVal := reflect.Indirect(reflect.ValueOf(srcSlice[i]))
		destVal := reflect.New(destType).Elem()
		switch {
		case srcVal.Type().ConvertibleTo(destType):
			destVal = srcVal.Convert(destType)
		case srcVal.Type().AssignableTo(destType):
			destVal = srcVal
		default:
			for j := 0; j < srcVal.NumField(); j++ {
				srcField := srcVal.Type().Field(j)
				destField := destVal.FieldByName(srcField.Name)
				if destField.IsValid() && srcField.Type.AssignableTo(destField.Type()) {
					destField.Set(srcVal.Field(j))
				}
			}
		}
		destSlice.Index(i).Set(destVal)
	}
	return destSlice.Interface().([]Y)
}

// findRootCaller Finds the root caller filepath of the application
func findRootCaller() string {
	const MaxDepth = 32

	for i := 0; i < MaxDepth; i++ {
		_, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if i == MaxDepth-1 || !strings.Contains(file, "runtime/") {
			return file
		}
	}
	return ""
}
