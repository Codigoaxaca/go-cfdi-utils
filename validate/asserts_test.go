package validate

import (
	"context"
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

func TestAsserts(t *testing.T) {
	as := NewAsserts()
	as.Put("A01", "Primero")
	as.PutStatus("A01", StatusOk)
	as.Put("A02", "Segundo").SetStatus(StatusError, "falló")
	as.PutStatus("A03", StatusWarning)
	if as.Len() != 3 || !as.HasErrors() || !as.HasWarnings() {
		t.Fatal("estado inesperado")
	}
	if as.Get("A01").Title != "Primero" || as.Get("A01").Status != StatusOk {
		t.Fatal("A01 inesperado")
	}
	if len(as.Errors()) != 1 || as.FirstError().Code != "A02" || as.FirstError().Explanation != "falló" {
		t.Fatal("errores inesperados")
	}
	as.Put("A01", "Primero renombrado")
	if as.Len() != 3 || as.Get("A01").Title != "Primero renombrado" {
		t.Fatal("no debe duplicar códigos")
	}
	as.Remove("A02")
	if as.HasErrors() || as.Exists("A02") {
		t.Fatal("no se eliminó")
	}
	if s := as.Get("A01").String(); s != "OK: A01 - Primero renombrado" {
		t.Fatalf("String inesperado %q", s)
	}
	if !strings.Contains(as.String(), "WARN  A03") {
		t.Fatalf("reporte inesperado:\n%s", as.String())
	}
	other := NewAsserts()
	other.PutStatus("A01", StatusError, "importado")
	other.SetMustStop(true)
	as.Import(other)
	if as.Get("A01").Status != StatusError || !as.MustStop() {
		t.Fatal("import inesperado")
	}
}

func TestStatus(t *testing.T) {
	if StatusWhen(true) != StatusOk || StatusWhen(false) != StatusError || StatusWhenOrWarning(false) != StatusWarning {
		t.Fatal("StatusWhen")
	}
	for s, want := range map[Status]string{StatusNone: "NONE", StatusOk: "OK", StatusWarning: "WARN", StatusError: "ERROR"} {
		if s.String() != want {
			t.Fatalf("%d => %s", s, s.String())
		}
	}
}

func TestMultiValidatorMustStop(t *testing.T) {
	calls := 0
	stopper := ValidatorFunc(func(_ context.Context, _ *nodes.Node, _ *Environment, as *Asserts) {
		calls++
		as.PutStatus("STOP", StatusError)
		as.SetMustStop(true)
	})
	never := ValidatorFunc(func(_ context.Context, _ *nodes.Node, _ *Environment, as *Asserts) {
		calls++
	})
	as := NewAsserts()
	NewMultiValidator("4.0", stopper, never).Validate(context.Background(), nodes.New("x", nil), nil, as)
	if calls != 1 || !as.HasErrors() {
		t.Fatalf("calls=%d", calls)
	}
}
