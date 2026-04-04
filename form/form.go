// Package form provides generic HTTP form decoding using reflection.
// It maps form values to struct fields using the "form" struct tag,
// falling back to the lowercase field name. Supported types: string, int, float64, bool.
package form

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// fieldInfo holds cached metadata for a single struct field.
type fieldInfo struct {
	index    int
	kind     reflect.Kind
	required bool
	name     string // form field name, used in error messages
}

// FieldError represents a validation error for a single form field.
type FieldError struct {
	Field   string
	Message string
}

func (e FieldError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationError holds one or more field-level validation errors.
type ValidationError struct {
	Errors []FieldError
}

func (e ValidationError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	msg := "validation failed:"
	for _, fe := range e.Errors {
		msg += " " + fe.Error() + ";"
	}
	return msg
}

// HasField returns true if the given field name has a validation error.
func (e ValidationError) HasField(name string) bool {
	for _, fe := range e.Errors {
		if fe.Field == name {
			return true
		}
	}
	return false
}

// typeCache stores parsed struct field metadata keyed by reflect.Type.
var typeCache sync.Map // map[reflect.Type]map[string]fieldInfo

// Decode parses an HTTP request's form data into a new instance of T.
// Struct fields are matched by the "form" tag or lowercase field name.
//
//	type Login struct {
//	    Email    string `form:"email"`
//	    Remember bool   `form:"remember"`
//	}
//	data, err := form.Decode[Login](r)
func Decode[T any](r *http.Request) (T, error) {
	var zero T
	if err := r.ParseForm(); err != nil {
		return zero, err
	}

	result := new(T)
	rv := reflect.ValueOf(result).Elem()
	fields := cachedFields(rv.Type())

	for name, fi := range fields {
		raw := r.FormValue(name)
		if raw == "" {
			continue
		}
		field := rv.Field(fi.index)
		if err := setField(field, fi.kind, raw); err != nil {
			return zero, err
		}
	}

	if err := validate(rv, fields); err != nil {
		return zero, err
	}

	return *result, nil
}

// validate checks required fields and returns a ValidationError if any are missing.
func validate(rv reflect.Value, fields map[string]fieldInfo) error {
	var errs []FieldError
	for _, fi := range fields {
		if !fi.required {
			continue
		}
		field := rv.Field(fi.index)
		if field.IsZero() {
			errs = append(errs, FieldError{Field: fi.name, Message: "required"})
		}
	}
	if len(errs) > 0 {
		return ValidationError{Errors: errs}
	}
	return nil
}

// cachedFields returns the field mapping for t, building and caching it on first access.
func cachedFields(t reflect.Type) map[string]fieldInfo {
	if cached, ok := typeCache.Load(t); ok {
		return cached.(map[string]fieldInfo)
	}

	fields := buildFields(t)
	actual, _ := typeCache.LoadOrStore(t, fields)
	return actual.(map[string]fieldInfo)
}

// buildFields inspects a struct type and returns a map from form field name to fieldInfo.
func buildFields(t reflect.Type) map[string]fieldInfo {
	fields := make(map[string]fieldInfo, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		name := sf.Tag.Get("form")
		if name == "" || name == "-" {
			if name == "-" {
				continue
			}
			name = toLower(sf.Name)
		}
		required := false
		if validate := sf.Tag.Get("validate"); validate != "" {
			for _, rule := range strings.Split(validate, ",") {
				if strings.TrimSpace(rule) == "required" {
					required = true
					break
				}
			}
		}
		fields[name] = fieldInfo{index: i, kind: sf.Type.Kind(), required: required, name: name}
	}
	return fields
}

// setField assigns a string value to a reflect.Value based on its kind.
func setField(field reflect.Value, kind reflect.Kind, raw string) error {
	switch kind {
	case reflect.String:
		field.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(b)
	}
	return nil
}

// toLower returns a simple lowercase version of name (ASCII only).
func toLower(name string) string {
	b := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
