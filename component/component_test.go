package component

import (
	"net/http/httptest"
	"testing"
	"thunder/state"
)

func TestCtx(t *testing.T) {
	s := state.New()
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	params := map[string]string{"id": "123"}

	ctx := &Ctx{
		State:   s,
		Request: req,
		Params:  params,
		Writer:  rr,
	}

	if ctx.State != s {
		t.Errorf("expected state mismatch")
	}
	if ctx.Request != req {
		t.Errorf("expected request mismatch")
	}
	if ctx.Params["id"] != "123" {
		t.Errorf("expected param mismatch")
	}
}

func TestComponent(t *testing.T) {
	comp := &Component{
		TemplatePath: "test.html",
		LayoutPath:   "layout.html",
		Handler: func(ctx *Ctx) any {
			return "data"
		},
	}

	if comp.TemplatePath != "test.html" {
		t.Errorf("expected template path mismatch")
	}
	if comp.Handler == nil {
		t.Errorf("expected handler mismatch")
	}
}
