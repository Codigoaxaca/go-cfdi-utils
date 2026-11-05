package cfdi40_test

import (
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
)

func TestReadFile(t *testing.T) {
	c, err := cfdi40.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	if c.SearchAttribute("cfdi:Emisor", "Rfc") != "ISD950921HE5" {
		t.Fatal("rfc emisor inesperado")
	}
	if len(c.Conceptos().Items()) == 0 {
		t.Fatal("sin conceptos")
	}
	if _, err := cfdi40.ReadFile("../testdata/cfdi/cfdi33-real.xml"); err == nil {
		t.Fatal("se esperaba error con CFDI 3.3")
	}
}
