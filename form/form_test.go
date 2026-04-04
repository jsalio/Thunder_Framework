package form

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type LoginForm struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Remember bool   `form:"remember"`
}

type ItemForm struct {
	Name     string  `form:"name"`
	Quantity int     `form:"quantity"`
	Price    float64 `form:"price"`
	Active   bool    `form:"active"`
}

type NoTagForm struct {
	FirstName string
	Age       int
}

type SkipForm struct {
	Visible string `form:"visible"`
	Hidden  string `form:"-"`
}

func newFormRequest(values url.Values) *http.Request {
	body := values.Encode()
	r, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestDecodeStringFields(t *testing.T) {
	r := newFormRequest(url.Values{
		"email":    {"user@example.com"},
		"password": {"secret"},
	})
	got, err := Decode[LoginForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "user@example.com")
	}
	if got.Password != "secret" {
		t.Errorf("Password = %q, want %q", got.Password, "secret")
	}
}

func TestDecodeBoolField(t *testing.T) {
	r := newFormRequest(url.Values{
		"email":    {"a@b.com"},
		"password": {"x"},
		"remember": {"true"},
	})
	got, err := Decode[LoginForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Remember {
		t.Error("Remember = false, want true")
	}
}

func TestDecodeIntAndFloat(t *testing.T) {
	r := newFormRequest(url.Values{
		"name":     {"Widget"},
		"quantity": {"5"},
		"price":    {"19.99"},
		"active":   {"true"},
	})
	got, err := Decode[ItemForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Widget" {
		t.Errorf("Name = %q, want %q", got.Name, "Widget")
	}
	if got.Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", got.Quantity)
	}
	if got.Price != 19.99 {
		t.Errorf("Price = %f, want 19.99", got.Price)
	}
	if !got.Active {
		t.Error("Active = false, want true")
	}
}

func TestDecodeNoTag(t *testing.T) {
	r := newFormRequest(url.Values{
		"firstname": {"Alice"},
		"age":       {"30"},
	})
	got, err := Decode[NoTagForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstName != "Alice" {
		t.Errorf("FirstName = %q, want %q", got.FirstName, "Alice")
	}
	if got.Age != 30 {
		t.Errorf("Age = %d, want 30", got.Age)
	}
}

func TestDecodeSkipTag(t *testing.T) {
	r := newFormRequest(url.Values{
		"visible": {"yes"},
		"hidden":  {"should-be-ignored"},
	})
	got, err := Decode[SkipForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Visible != "yes" {
		t.Errorf("Visible = %q, want %q", got.Visible, "yes")
	}
	if got.Hidden != "" {
		t.Errorf("Hidden = %q, want empty (skipped)", got.Hidden)
	}
}

func TestDecodeInvalidInt(t *testing.T) {
	r := newFormRequest(url.Values{
		"name":     {"X"},
		"quantity": {"not-a-number"},
	})
	_, err := Decode[ItemForm](r)
	if err == nil {
		t.Error("expected error for invalid int, got nil")
	}
}

func TestDecodeEmptyValues(t *testing.T) {
	r := newFormRequest(url.Values{})
	got, err := Decode[LoginForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "" || got.Password != "" || got.Remember {
		t.Errorf("expected zero value struct, got %+v", got)
	}
}

// Validation tests

type RegisterForm struct {
	Username string `form:"username" validate:"required"`
	Email    string `form:"email" validate:"required"`
	Bio      string `form:"bio"`
}

func TestValidateRequired(t *testing.T) {
	r := newFormRequest(url.Values{
		"username": {"alice"},
		"email":    {"alice@example.com"},
		"bio":      {"hello"},
	})
	got, err := Decode[RegisterForm](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice" {
		t.Errorf("Username = %q, want %q", got.Username, "alice")
	}
}

func TestValidateRequiredMissing(t *testing.T) {
	r := newFormRequest(url.Values{
		"bio": {"hello"},
	})
	_, err := Decode[RegisterForm](r)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	ve, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !ve.HasField("username") {
		t.Error("expected error for field 'username'")
	}
	if !ve.HasField("email") {
		t.Error("expected error for field 'email'")
	}
}

func TestValidateRequiredPartialMissing(t *testing.T) {
	r := newFormRequest(url.Values{
		"username": {"alice"},
	})
	_, err := Decode[RegisterForm](r)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	ve := err.(ValidationError)
	if ve.HasField("username") {
		t.Error("username was provided, should not have error")
	}
	if !ve.HasField("email") {
		t.Error("expected error for missing 'email'")
	}
}

func TestValidationErrorMessage(t *testing.T) {
	r := newFormRequest(url.Values{})
	_, err := Decode[RegisterForm](r)
	ve := err.(ValidationError)
	msg := ve.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestFieldErrorMessage(t *testing.T) {
	fe := FieldError{Field: "email", Message: "required"}
	if fe.Error() != "email: required" {
		t.Errorf("got %q", fe.Error())
	}
}

// Benchmarks

func BenchmarkDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := newFormRequest(url.Values{
			"name":     {"Widget"},
			"quantity": {"5"},
			"price":    {"19.99"},
			"active":   {"true"},
		})
		_, _ = Decode[ItemForm](r)
	}
}

func BenchmarkDecodeWithCache(b *testing.B) {
	// Warm the cache
	r := newFormRequest(url.Values{"name": {"W"}, "quantity": {"1"}, "price": {"1.0"}, "active": {"true"}})
	_, _ = Decode[ItemForm](r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newFormRequest(url.Values{
			"name":     {"Widget"},
			"quantity": {"5"},
			"price":    {"19.99"},
			"active":   {"true"},
		})
		_, _ = Decode[ItemForm](r)
	}
}
