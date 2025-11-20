package safetool

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"text/template"
	"time"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

// Varchar is a string type that can be easily converted to/from JSON.
type Varchar string

func (s Varchar) Bytes() []byte {
	return []byte(s)
}

func (s Varchar) String() string {
	return string(s)
}

// MarshalJSON implements the json.Marshaler interface for Varchar.
// It marshals the underlying byte slice.
func (s *Varchar) MarshalJSON() ([]byte, error) {
	if s == nil || len(s.Bytes()) == 0 {
		return []byte(`""`), nil
	}
	return json.Marshal(string(*s))
}

// RandInt returns a random number in the specified range [min, max).
// It returns an error if random number generation fails.
func RandInt[num constraints.Signed](min, max num) (num, error) {
	if min >= max {
		return 0, fmt.Errorf("min (%v) must be less than max (%v)", min, max)
	}
	valRange := big.NewInt(int64(max - min))
	n, err := rand.Int(rand.Reader, valRange)
	if err != nil {
		return 0, fmt.Errorf("failed to generate random number: %w", err)
	}
	result := n.Add(n, big.NewInt(int64(min)))
	return num(result.Int64()), nil
}

// Ptr returns a pointer for any passed object.
func Ptr[T any](n T) *T {
	return &n
}

// Val returns the value pointed to by ptr, or the zero value of T if ptr is nil.
func Val[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}

// NilPtr returns a pointer to n if n is not zero, otherwise returns nil.
func NilPtr[T comparable](n T) *T {
	var zero T
	if n == zero {
		return nil
	}
	return &n
}

// ZeroVal returns the zero value of type T, regardless of the input value.
func ZeroVal[T any](_ T) T {
	var zero T
	return zero
}

// In checks if an element is present in a slice.
func In[T comparable](needle T, haystack ...T) bool {
	return slices.Contains(haystack, needle)
}

// RetryFunc re-runs the provided function f up to 'attempts' times if it returns an error.
// It waits for 'sleep' duration between retries.
// It returns nil if f succeeds, otherwise the last error from f.
func RetryFunc[num constraints.Signed](attempts num, sleep time.Duration, f func() error) error {
	var lastErr error
	for i := num(0); i <= attempts || attempts < 0; i++ { // attempts < 0 means infinite
		lastErr = f()
		if lastErr == nil {
			return nil
		}
		if i == attempts && attempts >= 0 {
			break
		}
		time.Sleep(sleep)
	}
	return lastErr
}

// Jsonify serializes the given value 's' into a JSON Varchar.
// It returns the JSON Varchar and an error if marshaling fails.
func Jsonify(s any) (Varchar, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return Varchar(b), nil
}

// Objectify unmarshals the JSON input 'in' (string or byte slice) into the 'target' pointer.
// It returns an error if unmarshaling fails.
func Objectify[T ~[]byte | ~string](in T, target any) error {
	err := json.Unmarshal([]byte(in), target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// Strtr replaces all occurrences of keys in 'oldToNew' map with their corresponding values in 'subject' string.
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

// NonZero returns the first non-zero value from the provided arguments 'ts'.
// If all values are zero, it returns the zero value for type T.
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

// IsZero checks if the given value 'v' is the zero value for its type.
func IsZero[T comparable](v T) bool {
	var zeroValue T
	return v == zeroValue
}

// Zero returns the zero value for a given type T.
func Zero[T any]() T {
	var zeroValue T
	return zeroValue
}

// ExecTemplate executes a text template with the given variables.
// It returns the executed template as a string and an error if parsing or execution fails.
func ExecTemplate(templateText string, templateVars any) (string, error) {
	tpl, err := template.New("safetool_template").Option("missingkey=zero").Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	var buf strings.Builder
	err = tpl.Execute(&buf, templateVars)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// ConvertSlice converts a slice of type T to a slice of type Y.
// It returns the converted slice and an error if the input is not a slice
// or if conversion of elements is not possible.
func ConvertSlice[T any, Y any](srcSlice []T, destTypedValue Y) ([]Y, error) {
	srcReflectType := reflect.TypeOf(srcSlice)
	if srcReflectType.Kind() != reflect.Slice {
		return nil, errors.New("srcSlice is not a slice")
	}
	if srcSlice == nil {
		return nil, errors.New("srcSlice is nil")
	}
	if len(srcSlice) == 0 {
		return []Y{}, nil
	}

	destType := reflect.TypeOf(destTypedValue)
	destSlice := reflect.MakeSlice(reflect.SliceOf(destType), len(srcSlice), len(srcSlice))

	for i := range srcSlice {
		srcVal := reflect.ValueOf(srcSlice[i])

		if !srcVal.IsValid() || ((srcVal.Kind() == reflect.Ptr || srcVal.Kind() == reflect.Interface) && srcVal.IsNil()) {
			destSlice.Index(i).Set(reflect.Zero(destType))
			continue
		}

		if srcVal.Kind() == reflect.Ptr {
			srcVal = srcVal.Elem()
			if !srcVal.IsValid() || (srcVal.Kind() == reflect.Interface && srcVal.IsNil()) {
				destSlice.Index(i).Set(reflect.Zero(destType))
				continue
			}
		}

		if srcVal.Kind() == reflect.Interface {
			srcVal = srcVal.Elem()
			if !srcVal.IsValid() {
				destSlice.Index(i).Set(reflect.Zero(destType))
				continue
			}
		}

		newDestVal := reflect.New(destType).Elem()

		srcKind := srcVal.Kind()
		destKind := destType.Kind()
		if ((srcKind >= reflect.Int && srcKind <= reflect.Uintptr) ||
			(srcKind >= reflect.Float32 && srcKind <= reflect.Complex128) ||
			srcKind == reflect.Bool) && destKind == reflect.String {
			return nil, fmt.Errorf("cannot convert element at index %d: direct conversion from numeric/bool type %s to string is not supported by ConvertSlice", i, srcVal.Type())
		}

		if srcVal.Type().ConvertibleTo(destType) {
			newDestVal.Set(srcVal.Convert(destType))
		} else if srcVal.Type().AssignableTo(destType) {
			newDestVal.Set(srcVal)
		} else if srcVal.Kind() == reflect.Struct && newDestVal.Kind() == reflect.Struct {
			for j := 0; j < srcVal.NumField(); j++ {
				srcFieldDesc := srcVal.Type().Field(j)
				srcFieldVal := srcVal.Field(j)
				destField := newDestVal.FieldByName(srcFieldDesc.Name)

				if destField.IsValid() && destField.CanSet() {
					if srcFieldVal.Type().AssignableTo(destField.Type()) {
						destField.Set(srcFieldVal)
					} else {
						return nil, fmt.Errorf("cannot convert element at index %d: struct field '%s' type mismatch, source type %s, destination type %s, not assignable",
							i, srcFieldDesc.Name, srcFieldVal.Type(), destField.Type())
					}
				}
			}
		} else {
			return nil, fmt.Errorf("cannot convert element at index %d from type %s to %s: no direct conversion, assignability, or compatible struct field copy strategy found", i, srcVal.Type(), destType)
		}
		destSlice.Index(i).Set(newDestVal)
	}
	return destSlice.Interface().([]Y), nil
}

// FindRootCaller finds the root caller filepath of the application.
// It skips runtime frames.
func FindRootCaller() string {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(0, pcs)
	_ = runtime.CallersFrames(pcs[:n])

	for i := 2; i < maxDepth; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if strings.Contains(file, "/runtime/") || strings.Contains(file, "libexec/src/runtime/") {
			continue
		}
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			_ = fn.Name()
		}
		return file
	}
	return ""
}

// GetRelativePath calculates a relative path from the directory of the root caller to the given filePath.
// It returns the relative path or the original filePath and an error if the calculation fails.
func GetRelativePath(filePath string) (string, error) {
	callerPath := FindRootCaller()
	if callerPath == "" {
		return filePath, errors.New("could not determine caller path")
	}

	callerDir := filepath.Dir(callerPath)
	relPath, err := filepath.Rel(callerDir, filePath)
	if err != nil {
		return filePath, fmt.Errorf("failed to get relative path from %s to %s: %w", callerDir, filePath, err)
	}
	return relPath, nil
}
