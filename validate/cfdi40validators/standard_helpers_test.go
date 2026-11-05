package cfdi40validators

import (
	"context"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// newComprobante construye un cfdi:Comprobante 4.0 con los atributos dados
// y el contenido XML interno (sin declarar el prefijo cfdi).
func newComprobante(t *testing.T, attrs string, inner string) *nodes.Node {
	t.Helper()
	xml := `<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" Version="4.0" ` + attrs + `>` + inner + `</cfdi:Comprobante>`
	c, err := nodes.ParseXMLString(xml)
	if err != nil {
		t.Fatalf("no se pudo interpretar el XML de prueba: %v\n%s", err, xml)
	}
	return c
}

func runValidator(v validate.Validator, c *nodes.Node) *validate.Asserts {
	asserts := validate.NewAsserts()
	v.Validate(context.Background(), c, &validate.Environment{}, asserts)
	return asserts
}

// expectStatuses comprueba el estado de los códigos dados.
func expectStatuses(t *testing.T, asserts *validate.Asserts, expected map[string]validate.Status) {
	t.Helper()
	for code, status := range expected {
		a := asserts.Get(code)
		if a == nil {
			t.Errorf("no existe el assert %s\n%s", code, asserts)
			continue
		}
		if a.Status != status {
			t.Errorf("%s: se esperaba %s y se obtuvo %s (%s)\n%s", code, status, a.Status, a.Explanation, asserts)
		}
	}
}

// expectCodes comprueba que existan exactamente los códigos dados, en orden,
// todos con título.
func expectCodes(t *testing.T, asserts *validate.Asserts, codes ...string) {
	t.Helper()
	all := asserts.All()
	if len(all) != len(codes) {
		t.Fatalf("se esperaban %d asserts y se obtuvieron %d\n%s", len(codes), len(all), asserts)
	}
	for i, a := range all {
		if a.Code != codes[i] {
			t.Errorf("assert #%d: se esperaba %s y se obtuvo %s", i, codes[i], a.Code)
		}
		if a.Title == "" {
			t.Errorf("assert %s sin título", a.Code)
		}
	}
}

const (
	stNone  = validate.StatusNone
	stOk    = validate.StatusOk
	stError = validate.StatusError
)
