package norn_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// require provides the few checks of testify's require package that these
// tests use. testify does not work with TinyGo (it panics in reflect while
// initializing), and this package is built with TinyGo for the firmware, so
// its tests have to run there too: tinygo test ./pkg/norn
var require checks

type checks struct{}

// fail reports the failure, with the optional message of the caller, and
// stops the test.
func fail(t *testing.T, failure string, msgAndArgs []any) {
	t.Helper()
	if len(msgAndArgs) > 0 {
		if format, ok := msgAndArgs[0].(string); ok {
			failure += "\n" + fmt.Sprintf(format, msgAndArgs[1:]...)
		} else {
			failure += "\n" + fmt.Sprint(msgAndArgs...)
		}
	}
	t.Fatal(failure)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func (checks) Equal(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		fail(t, fmt.Sprintf("not equal:\nexpected: %#v\nactual:   %#v", expected, actual), msgAndArgs)
	}
}

func (checks) True(t *testing.T, value bool, msgAndArgs ...any) {
	t.Helper()
	if !value {
		fail(t, "should be true", msgAndArgs)
	}
}

func (checks) Nil(t *testing.T, value any, msgAndArgs ...any) {
	t.Helper()
	if !isNil(value) {
		fail(t, fmt.Sprintf("expected nil, got: %#v", value), msgAndArgs)
	}
}

func (checks) NotNil(t *testing.T, value any, msgAndArgs ...any) {
	t.Helper()
	if isNil(value) {
		fail(t, "expected a value, got nil", msgAndArgs)
	}
}

func (checks) NoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		fail(t, fmt.Sprintf("unexpected error: %v", err), msgAndArgs)
	}
}

func (checks) Error(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		fail(t, "expected an error, got nil", msgAndArgs)
	}
}

func (checks) ErrorContains(t *testing.T, err error, contains string, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		fail(t, fmt.Sprintf("expected an error containing %q, got nil", contains), msgAndArgs)
	} else if !strings.Contains(err.Error(), contains) {
		fail(t, fmt.Sprintf("error %q does not contain %q", err.Error(), contains), msgAndArgs)
	}
}

func (checks) Contains(t *testing.T, s, contains string, msgAndArgs ...any) {
	t.Helper()
	if !strings.Contains(s, contains) {
		fail(t, fmt.Sprintf("%q does not contain %q", s, contains), msgAndArgs)
	}
}
