package cfdi40validators

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

func TestStandardValidatorsOrderAndVersion(t *testing.T) {
	validators := StandardValidators()
	if len(validators) != 13 {
		t.Fatalf("se esperaban 13 validadores, se obtuvieron %d", len(validators))
	}
	for _, v := range validators {
		if !v.CanValidateCfdiVersion("4.0") || v.CanValidateCfdiVersion("3.3") {
			t.Errorf("%T debe aplicar solo a CFDI 4.0", v)
		}
	}
	if _, isFecha := validators[10].(FechaComprobante); !isFecha {
		t.Errorf("el validador #10 debe ser FechaComprobante, es %T", validators[10])
	}
}

func TestStandardValidatorsOnRealFiles(t *testing.T) {
	files := []string{
		"cfdi40-real.xml",
		"cfdi40-valid.xml",
		"created-with-discounts-40.xml",
		"created-cfdi40-pago20-valid.xml",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			c, err := nodes.ParseXMLFile(filepath.Join("..", "..", "testdata", "cfdi", file))
			if err != nil {
				t.Fatal(err)
			}
			asserts := validate.NewAsserts()
			validate.NewMultiValidator("4.0", StandardValidators()...).
				Validate(context.Background(), c, &validate.Environment{}, asserts)
			if asserts.HasErrors() || asserts.HasWarnings() {
				t.Errorf("no se esperaban errores ni advertencias:\n%s", asserts)
			}
			// asserts que siempre aplican
			expectStatuses(t, asserts, map[string]validate.Status{
				"MONDEC01": stOk, "MONDEC03": stOk, "TIPOCAMBIO01": stOk, "TOTAL01": stOk,
				"REGFIS01": stOk, "FECHA01": stOk, "FECHA02": stOk, "RESFISC01": stOk,
				"SUMAS01": stOk, "SUMAS03": stOk, "CONCEPDESC01": stOk,
			})
			if testing.Verbose() {
				t.Logf("\n%s", asserts)
			}
		})
	}
}
