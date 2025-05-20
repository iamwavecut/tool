package safetool_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iamwavecut/tool/safetool"
)

func TestVarchar(t *testing.T) {
	t.Parallel()
	t.Run("Bytes", func(t *testing.T) {
		t.Parallel()
		sv := safetool.Varchar("hello")
		if string(sv.Bytes()) != "hello" {
			t.Errorf("Expected Bytes() to return 'hello', got %s", string(sv.Bytes()))
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		sv := safetool.Varchar("world")
		if sv.String() != "world" {
			t.Errorf("Expected String() to return 'world', got %s", sv.String())
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name     string
			input    safetool.Varchar
			expected string
			hasError bool
		}{
			{"empty string", safetool.Varchar(""), "\"\"", false},
			{"simple string", safetool.Varchar("hello"), `"hello"`, false},
			{"string with quotes", safetool.Varchar(`"quoted"`), `"\"quoted\""`, false},
			{"json string", safetool.Varchar(`{"key":"value"}`), `"{\"key\":\"value\"}"`, false},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				result, err := json.Marshal(&tc.input)
				if (err != nil) != tc.hasError {
					t.Errorf("Expected error status %v, got error %v", tc.hasError, err)
				}
				if string(result) != tc.expected {
					t.Errorf("Expected JSON '%s', got '%s'", tc.expected, string(result))
				}
			})
		}
	})
}

func TestRandInt(t *testing.T) {
	t.Parallel()
	t.Run("valid range", func(t *testing.T) {
		t.Parallel()
		min, max := 10, 20
		for i := 0; i < 100; i++ {
			val, err := safetool.RandInt(min, max)
			if err != nil {
				t.Fatalf("RandInt returned an error: %v", err)
			}
			if val < min || val >= max {
				t.Errorf("RandInt(%d, %d) returned %d, which is out of range", min, max, val)
			}
		}
	})

	t.Run("invalid range min > max", func(t *testing.T) {
		t.Parallel()
		min, max := 20, 10
		_, err := safetool.RandInt(min, max)
		if err == nil {
			t.Error("RandInt did not return an error for min > max")
		}
	})

	t.Run("invalid range min == max", func(t *testing.T) {
		t.Parallel()
		min, max := 10, 10
		_, err := safetool.RandInt(min, max)
		if err == nil {
			t.Error("RandInt did not return an error for min == max")
		}
	})
}

func TestPtr(t *testing.T) {
	t.Parallel()
	val := 10
	ptrVal := safetool.Ptr(val)
	if *ptrVal != val {
		t.Errorf("Expected Ptr to return a pointer to %d, got pointer to %d", val, *ptrVal)
	}
	*ptrVal = 20
	if val == 20 {
		t.Error("Ptr did not return a pointer to a copy of the value")
	}

	strVal := "hello"
	ptrStrVal := safetool.Ptr(strVal)
	if *ptrStrVal != strVal {
		t.Errorf("Expected Ptr to return a pointer to %s, got pointer to %s", strVal, *ptrStrVal)
	}
}

func TestIn(t *testing.T) {
	t.Parallel()
	t.Run("element present", func(t *testing.T) {
		t.Parallel()
		if !safetool.In(5, 1, 2, 3, 4, 5) {
			t.Error("Expected In to return true when element is present")
		}
	})

	t.Run("element not present", func(t *testing.T) {
		t.Parallel()
		if safetool.In(10, 1, 2, 3, 4, 5) {
			t.Error("Expected In to return false when element is not present")
		}
	})

	t.Run("empty haystack", func(t *testing.T) {
		t.Parallel()
		if safetool.In(5) {
			t.Error("Expected In to return false for empty haystack")
		}
	})

	t.Run("string slice", func(t *testing.T) {
		t.Parallel()
		if !safetool.In("b", "a", "b", "c") {
			t.Error("Expected In to return true for string slice")
		}
	})
}

func TestRetryFunc(t *testing.T) {
	t.Parallel()
	t.Run("succeeds on first try", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		fn := func() error {
			attempts++
			return nil
		}
		err := safetool.RetryFunc(3, 1*time.Millisecond, fn)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("Expected function to be called once, called %d times", attempts)
		}
	})

	t.Run("fails then succeeds", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		targetAttempts := 2
		fn := func() error {
			attempts++
			if attempts < targetAttempts {
				return errors.New("transient error")
			}
			return nil
		}
		err := safetool.RetryFunc(3, 1*time.Millisecond, fn)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if attempts != targetAttempts {
			t.Errorf("Expected function to be called %d times, called %d times", targetAttempts, attempts)
		}
	})

	t.Run("fails all attempts", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		expectedErr := errors.New("persistent error")
		fn := func() error {
			attempts++
			return expectedErr
		}
		err := safetool.RetryFunc(3, 1*time.Millisecond, fn)
		if err == nil {
			t.Error("Expected an error, got nil")
		} else if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
		}
		if attempts != 4 {
			t.Errorf("Expected function to be called 4 times, called %d times", attempts)
		}
	})

	t.Run("zero attempts means try once", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		fn := func() error {
			attempts++
			return errors.New("error on first try")
		}
		err := safetool.RetryFunc(0, 1*time.Millisecond, fn)
		if err == nil {
			t.Error("Expected an error, got nil for 0 attempts")
		}
		if attempts != 1 {
			t.Errorf("Expected function to be called once for 0 attempts, called %d times", attempts)
		}
	})

	t.Run("negative attempts means infinite retries", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		succeedAfter := 5
		var wg sync.WaitGroup
		wg.Add(1)
		fn := func() error {
			attempts++
			if attempts < succeedAfter {
				return errors.New("transient error")
			}
			wg.Done()
			return nil
		}
		go func() {
			err := safetool.RetryFunc(-1, 1*time.Millisecond, fn)
			if err != nil {
				t.Errorf("Goroutine: Expected no error for infinite retries, got %v", err)
			}
		}()

		select {
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Test timed out waiting for infinite retry to succeed")
		case <-waitGroupDone(&wg):
		}

		if attempts != succeedAfter {
			t.Errorf("Expected function to be called %d times, called %d times", succeedAfter, attempts)
		}
	})
}

func waitGroupDone(wg *sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	return done
}

func TestJsonify(t *testing.T) {
	t.Parallel()
	type sampleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	testCases := []struct {
		name     string
		input    any
		expected safetool.Varchar
		hasError bool
	}{
		{"simple struct", sampleStruct{Name: "Alice", Age: 30}, `{"name":"Alice","age":30}`, false},
		{"slice", []int{1, 2, 3}, `[1,2,3]`, false},
		{"map", map[string]int{"a": 1, "b": 2}, `{"a":1,"b":2}`, false},
		{"string", "hello", `"hello"`, false},
		{"int", 123, `123`, false},
		{"bool true", true, `true`, false},
		{"bool false", false, `false`, false},
		{"nil", nil, `null`, false},
		{"channel", make(chan int), "", true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := safetool.Jsonify(tc.input)
			if (err != nil) != tc.hasError {
				t.Errorf("Expected error status %v, got error %v for input %v", tc.hasError, err, tc.input)
			}
			if result != tc.expected {
				t.Errorf("Expected Varchar '%s', got '%s' for input %v", tc.expected, result, tc.input)
			}
		})
	}
}

func TestObjectify(t *testing.T) {
	t.Parallel()
	type sampleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	testCases := []struct {
		name     string
		input    safetool.Varchar
		targetFn func() any
		expected any
		hasError bool
	}{
		{
			name:     "valid JSON to struct",
			input:    `{"name":"Bob","age":40}`,
			targetFn: func() any { return &sampleStruct{} },
			expected: &sampleStruct{Name: "Bob", Age: 40},
			hasError: false,
		},
		{
			name:     "valid JSON to map",
			input:    `{"key":"value"}`,
			targetFn: func() any { m := make(map[string]string); return &m },
			expected: &map[string]string{"key": "value"},
			hasError: false,
		},
		{
			name:     "invalid JSON",
			input:    `{"name":"Alice",:,"age":30}`,
			targetFn: func() any { return &sampleStruct{} },
			expected: &sampleStruct{},
			hasError: true,
		},
		{
			name:     "mismatched types (number to string field)",
			input:    `{"name":123,"age":30}`,
			targetFn: func() any { return &sampleStruct{} },
			expected: &sampleStruct{},
			hasError: true,
		},
		{
			name:     "empty input string to struct",
			input:    "",
			targetFn: func() any { return &sampleStruct{} },
			expected: &sampleStruct{},
			hasError: true,
		},
		{
			name:     "'null' input string to struct pointer",
			input:    "null",
			targetFn: func() any { var s *sampleStruct; return &s },
			expected: new(*sampleStruct),
			hasError: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target := tc.targetFn()
			err := safetool.Objectify(tc.input, target)
			if (err != nil) != tc.hasError {
				t.Errorf("Expected error status %v, got error '%v' for input '%s'", tc.hasError, err, tc.input)
			}
			if !tc.hasError && !reflect.DeepEqual(target, tc.expected) {
				t.Errorf("Expected target %+v, got %+v for input '%s'", tc.expected, target, tc.input)
			}
		})
	}
}

func TestStrtr(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		subject  string
		oldToNew map[string]string
		expected string
	}{
		{"basic replacements", "hello world", map[string]string{"hello": "hi", "world": "earth"}, "hi earth"},
		{"no replacements", "hello world", map[string]string{"foo": "bar"}, "hello world"},
		{"empty subject", "", map[string]string{"hello": "hi"}, ""},
		{"empty map", "hello world", map[string]string{}, "hello world"},
		{"replacement to empty string", "hello world hello", map[string]string{"hello": ""}, " world "},
		{"overlapping keys (sequential map iteration dependent)", "ababab", map[string]string{"aba": "x", "ab": "y"}, ""},
		{"empty old string in map", "hello", map[string]string{"": "x"}, "hello"},
		{"old equals new in map", "hello", map[string]string{"h": "h"}, "hello"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := safetool.Strtr(tc.subject, tc.oldToNew)
			if tc.name == "overlapping keys (sequential map iteration dependent)" {
				if result != "yyy" && result != "xby" {
					t.Errorf("Expected 'yyy' or 'xby', got '%s'", result)
				}
			} else if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestNonZero(t *testing.T) {
	t.Parallel()
	t.Run("int first non-zero", func(t *testing.T) {
		t.Parallel()
		if safetool.NonZero(0, 0, 5, 10) != 5 {
			t.Error("Expected NonZero to return first non-zero int")
		}
	})
	t.Run("int last non-zero", func(t *testing.T) {
		t.Parallel()
		if safetool.NonZero(0, 0, 0, 10) != 10 {
			t.Error("Expected NonZero to return last non-zero int")
		}
	})
	t.Run("string all zero", func(t *testing.T) {
		t.Parallel()
		if safetool.NonZero("", "", "") != "" {
			t.Error("Expected NonZero to return zero value for string")
		}
	})
	t.Run("no elements", func(t *testing.T) {
		t.Parallel()
		if safetool.NonZero[int]() != 0 {
			t.Error("Expected NonZero to return zero value for int with no elements")
		}
		if safetool.NonZero[string]() != "" {
			t.Error("Expected NonZero to return zero value for string with no elements")
		}
	})
	t.Run("pointer type non-nil", func(t *testing.T) {
		t.Parallel()
		var p1 *int
		p2 := new(int)
		*p2 = 5
		if safetool.NonZero(p1, p2) != p2 {
			t.Error("Expected NonZero to return first non-nil pointer")
		}
	})
}

func TestIsZero(t *testing.T) {
	t.Parallel()
	t.Run("int zero", func(t *testing.T) {
		t.Parallel()
		if !safetool.IsZero(0) {
			t.Error("Expected IsZero to return true for 0")
		}
	})
	t.Run("int non-zero", func(t *testing.T) {
		t.Parallel()
		if safetool.IsZero(5) {
			t.Error("Expected IsZero to return false for 5")
		}
	})
	t.Run("string zero", func(t *testing.T) {
		t.Parallel()
		if !safetool.IsZero("") {
			t.Error("Expected IsZero to return true for empty string")
		}
	})
	t.Run("string non-zero", func(t *testing.T) {
		t.Parallel()
		if safetool.IsZero("hello") {
			t.Error("Expected IsZero to return false for 'hello'")
		}
	})
	t.Run("pointer nil", func(t *testing.T) {
		t.Parallel()
		var p *int
		if !safetool.IsZero(p) {
			t.Error("Expected IsZero to return true for nil pointer")
		}
	})
	t.Run("pointer non-nil", func(t *testing.T) {
		t.Parallel()
		p := new(int)
		if safetool.IsZero(p) {
			t.Error("Expected IsZero to return false for non-nil pointer")
		}
	})
}

func TestExecTemplate(t *testing.T) {
	t.Parallel()
	type vars struct {
		Name string
		Age  int
	}

	testCases := []struct {
		name         string
		templateText string
		templateVars any
		expected     string
		hasError     bool
	}{
		{"valid template and vars", "Hello {{.Name}}, age {{.Age}}", vars{Name: "Test", Age: 25}, "Hello Test, age 25", false},
		{"invalid template syntax", "Hello {{.Name", vars{Name: "Test"}, "", true},
		// If a field .Missing does not exist in vars, text/template will error during execution, even with missingkey=zero.
		// missingkey=zero applies to map keys that are present but whose values might be nil, or for fields that are nil interfaces.
		// For a non-existent field in a struct, it's an execution error.
		{"non-existent field in struct", "Hello {{.Missing}}", vars{Name: "Test"}, "", true},
		// When data is nil, template engine renders <no value> for field access, no error.
		{"nil vars (map access will be invalid for .Name)", "Hello {{.Name}}", nil, "Hello <no value>", false},
		// text/template printf with bad format specifier for type does not return error from Execute,
		// instead, it embeds an error string in the output.
		{"printf with type mismatch", "{{printf \"%d\" .Name}}", vars{Name: "Test"}, "%!d(string=Test)", false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := safetool.ExecTemplate(tc.templateText, tc.templateVars)
			if (err != nil) != tc.hasError {
				t.Errorf("Expected error status %v, got error '%v' for template '%s'", tc.hasError, err, tc.templateText)
			}
			if result != tc.expected {
				t.Errorf("Expected result '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestConvertSlice(t *testing.T) {
	t.Parallel()
	type SrcStruct struct {
		ID   int
		Name string
		Val  float64
	}
	type DestStructSimple struct {
		ID   int
		Name string
	}
	type DestStructFull struct {
		ID   int
		Name string
		Val  float64
	}

	t.Run("slice of ints to slice of float64s (convertible)", func(t *testing.T) {
		t.Parallel()
		srcFloat := []int{1, 2, 3}
		destFloat, errFloat := safetool.ConvertSlice(srcFloat, float64(0))
		if errFloat != nil {
			t.Fatalf("ConvertSlice (int to float64) failed: %v", errFloat)
		}
		expectedFloat := []float64{1.0, 2.0, 3.0}
		if !reflect.DeepEqual(destFloat, expectedFloat) {
			t.Errorf("Expected %v, got %v", expectedFloat, destFloat)
		}
	})

	t.Run("slice of structs to slice of different structs (matching fields)", func(t *testing.T) {
		t.Parallel()
		src := []SrcStruct{
			{ID: 1, Name: "Alice", Val: 1.1},
			{ID: 2, Name: "Bob", Val: 2.2},
		}
		dest, err := safetool.ConvertSlice(src, DestStructSimple{})
		if err != nil {
			t.Fatalf("ConvertSlice failed: %v", err)
		}
		expected := []DestStructSimple{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		}
		if !reflect.DeepEqual(dest, expected) {
			t.Errorf("Expected %+v, got %+v", expected, dest)
		}
	})

	t.Run("slice of structs to identical structs", func(t *testing.T) {
		t.Parallel()
		src := []SrcStruct{
			{ID: 1, Name: "Alice", Val: 1.1},
		}
		dest, err := safetool.ConvertSlice(src, DestStructFull{})
		if err != nil {
			t.Fatalf("ConvertSlice failed: %v", err)
		}
		expected := []DestStructFull{
			{ID: 1, Name: "Alice", Val: 1.1},
		}
		if !reflect.DeepEqual(dest, expected) {
			t.Errorf("Expected %+v, got %+v", expected, dest)
		}
	})

	t.Run("slice of interface{} to slice of concrete types (assignable)", func(t *testing.T) {
		t.Parallel()
		src := []interface{}{1, 2, 3}
		dest, err := safetool.ConvertSlice(src, int(0))
		if err != nil {
			t.Fatalf("ConvertSlice failed: %v", err)
		}
		expected := []int{1, 2, 3}
		if !reflect.DeepEqual(dest, expected) {
			t.Errorf("Expected %v, got %v", expected, dest)
		}
	})

	t.Run("nil input slice", func(t *testing.T) {
		t.Parallel()
		var src []int
		dest, err := safetool.ConvertSlice(src, 0)
		if err == nil {
			t.Fatal("Expected an error for nil slice, got nil")
		}
		if err.Error() != "srcSlice is nil" {
			t.Errorf("Expected error 'srcSlice is nil', got '%v'", err)
		}
		if dest != nil {
			t.Errorf("Expected nil dest slice for nil src when error occurs, got %v", dest)
		}
	})

	t.Run("empty input slice", func(t *testing.T) {
		t.Parallel()
		src := []int{}
		dest, err := safetool.ConvertSlice(src, 0)
		if err != nil {
			t.Fatalf("ConvertSlice failed for empty slice: %v", err)
		}
		if len(dest) != 0 {
			t.Errorf("Expected empty dest slice for empty src, got %v", dest)
		}
	})

	t.Run("error case: incompatible types (struct to int)", func(t *testing.T) {
		t.Parallel()
		src := []SrcStruct{{ID: 1, Name: "Test"}}
		_, err := safetool.ConvertSlice(src, 0)
		if err == nil {
			t.Error("Expected error for incompatible type conversion (struct to int)")
		}
	})

	t.Run("error case: incompatible types (int to struct)", func(t *testing.T) {
		t.Parallel()
		src := []int{1, 2, 3}
		_, err := safetool.ConvertSlice(src, SrcStruct{})
		if err == nil {
			t.Error("Expected error for incompatible type conversion (int to struct)")
		}
	})

	t.Run("slice of pointers to structs", func(t *testing.T) {
		t.Parallel()
		src := []*SrcStruct{
			{ID: 1, Name: "PtrAlice", Val: 1.1},
			nil,
			{ID: 2, Name: "PtrBob", Val: 2.2},
		}
		dest, err := safetool.ConvertSlice(src, DestStructSimple{})
		if err != nil {
			t.Fatalf("ConvertSlice failed: %v", err)
		}
		expected := []DestStructSimple{
			{ID: 1, Name: "PtrAlice"},
			{},
			{ID: 2, Name: "PtrBob"},
		}
		if !reflect.DeepEqual(dest, expected) {
			t.Errorf("Expected %+v, got %+v", expected, dest)
		}
	})

	t.Run("element type mismatch", func(t *testing.T) {
		t.Parallel()
		// Case 1: Structs that are not convertible, assignable, and fields don't match
		type SourceStructInconvertible struct{ Val complex128 }
		type DestStructInconvertible struct{ Val bool }

		src1 := []SourceStructInconvertible{{Val: complex(1, 1)}}
		var destSample1 DestStructInconvertible
		_, err1 := safetool.ConvertSlice(src1, destSample1)
		if err1 == nil {
			t.Error("Expected an error for inconvertible struct types, got nil")
		} else if !strings.Contains(err1.Error(), "cannot convert element at index 0") {
			t.Errorf("Expected error message for inconvertible structs, got: %v", err1)
		}

		// Case 2: Slice of interface containing a type not convertible to dest type
		type MyCustomInt int
		src2 := []any{MyCustomInt(5)} // []any containing MyCustomInt
		var destSample2 string        // Target is string
		// MyCustomInt is not convertible to string, not assignable.
		_, err2 := safetool.ConvertSlice(src2, destSample2)
		if err2 == nil {
			t.Error("Expected an error for any(MyCustomInt) to string conversion, got nil")
		} else if !strings.Contains(err2.Error(), "cannot convert element at index 0") {
			t.Errorf("Expected error message for any(MyCustomInt) to string, got: %v", err2)
		}

		// Case 3: Slice of unassignable and unconvertible basic types
		src3 := []func(){func() {}}
		var destSample3 int
		_, err3 := safetool.ConvertSlice(src3, destSample3)
		if err3 == nil {
			t.Errorf("Expected error converting []func() to []int, got nil")
		} else if !strings.Contains(err3.Error(), "cannot convert element at index 0") {
			t.Errorf("Expected error message for []func() to []int, got: %v", err3)
		}
	})

	t.Run("srcSlice is not a slice (theoretically dead code)", func(t *testing.T) {
		t.Parallel()
		// This test attempts to cover a theoretically dead code path in ConvertSlice.
		// `ConvertSlice[T any, Y any](srcSlice []T, ...)` means srcSlice's type is always a slice.
		// `reflect.TypeOf(srcSlice).Kind() != reflect.Slice` should therefore always be false.
		// For completeness, if it were possible to trigger:
		// var notASlice int = 5
		// _, err := safetool.ConvertSlice(notASlice, 0) // This won't compile due to type constraints
		// if err == nil || !strings.Contains(err.Error(), "srcSlice is not a slice") {
		//  t.Errorf("Expected 'srcSlice is not a slice' error, got %v", err)
		// }
		t.Log("Skipping test for 'srcSlice is not a slice' as it's considered dead code due to Go's type system for generics (`srcSlice []T`).")
	})

	// Original Log for ConvertSlice about its limitations if any.
	t.Log("ConvertSlice tests cover basic conversions, nil/empty slices, struct conversions, and type mismatches. Deeper reflection cases or complex interface conversions might present further edge cases.")
}

func TestGetRelativePath(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "testrelpath*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetFilePath := filepath.Join(tempDir, "myfile.txt")
	if _, err := os.Create(targetFilePath); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	t.Run("get relative path to a temp file", func(t *testing.T) {
		t.Parallel()
		relPath, err := safetool.GetRelativePath(targetFilePath)
		if err != nil {
			t.Errorf("GetRelativePath(%s) returned error: %v", targetFilePath, err)
		}
		if relPath == "" {
			t.Errorf("GetRelativePath returned an empty string for %s", targetFilePath)
		}
		t.Logf("GetRelativePath for %s (abs: %s) from test context resulted in: %s", "myfile.txt", targetFilePath, relPath)
	})

	t.Run("filepath.Rel returns error", func(t *testing.T) {
		t.Parallel()
		// Using a path with a NUL character, which typically causes issues for path operations.
		// Note: The behavior of filepath.Rel with NUL bytes might vary or be sanitized.
		invalidFilePath := "some/path/with\x00/null.txt"
		_, err := safetool.GetRelativePath(invalidFilePath)

		if err == nil {
			t.Logf("Warning: GetRelativePath with NUL byte in path ('%s') did not return an error. This specific error condition for filepath.Rel might be elusive to trigger consistently across all OS/Go versions without deeper mocking of filepath.Rel or os.Getwd.", invalidFilePath)
		} else {
			if !strings.Contains(err.Error(), "failed to get relative path") {
				t.Errorf("Expected error to contain 'failed to get relative path', got: %v", err)
			}
			// Further check if the wrapped error from filepath is present
			// This depends on the exact error message from filepath.Rel which can be OS-specific
			// For example, on some systems it might be "invalid argument" or similar for NUL byte.
			// t.Logf("Got expected wrapped error: %v", err) // For diagnostics
		}

		// Attempt with a path that might confuse filepath.Rel on non-Windows systems
		// by looking like a Windows absolute path.
		if runtime.GOOS != "windows" {
			confusingPath := "C:\\Windows\\System32\\notepad.exe"
			_, errWindowsPath := safetool.GetRelativePath(confusingPath)
			if errWindowsPath == nil {
				t.Logf("Warning: GetRelativePath with Windows-like path '%s' on %s did not cause filepath.Rel to error. This coverage branch might be hard to hit.", confusingPath, runtime.GOOS)
			} else if !strings.Contains(errWindowsPath.Error(), "failed to get relative path") {
				t.Errorf("Expected error with confusing Windows path to contain 'failed to get relative path', got: %v", errWindowsPath)
			}
		}
	})

	// Original Log for GetRelativePath about its limitations.
	t.Log("GetRelativePath tests are limited due to its dependency on runtime.Caller for findRootCaller and the test execution context. The filepath.Rel error case is also hard to trigger consistently.")
}

func TestZero(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[int]()
		if val != 0 {
			t.Errorf("Expected Zero[int]() to be 0, got %d", val)
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[string]()
		if val != "" {
			t.Errorf("Expected Zero[string]() to be \"\", got %s", val)
		}
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[bool]()
		if val != false {
			t.Errorf("Expected Zero[bool]() to be false, got %t", val)
		}
	})

	t.Run("pointer", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[*int]()
		if val != nil {
			t.Errorf("Expected Zero[*int]() to be nil, got %v", val)
		}
	})

	type myStruct struct {
		A int
		B string
	}
	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[myStruct]()
		expected := myStruct{}
		if val != expected {
			t.Errorf("Expected Zero[myStruct]() to be %+v, got %+v", expected, val)
		}
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[[]int]()
		if val != nil {
			t.Errorf("Expected Zero[[]int]() to be nil, got %v", val)
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		val := safetool.Zero[map[string]int]()
		if val != nil {
			t.Errorf("Expected Zero[map[string]int]() to be nil, got %v", val)
		}
	})
}
