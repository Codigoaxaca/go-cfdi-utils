package cleaner

import (
	"os"
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/cadenaorigen"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

const dirty = `<?xml version="1.0" encoding="UTF-8"?>
<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:foo="http://example.com/foo" xmlns:bar="http://example.com/bar" xsi:schemaLocation="http://www.sat.gob.mx/cfd/4 http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd http://example.com/foo http://example.com/foo.xsd http://example.com/incompleto" Version="4.0">
  <cfdi:Emisor Rfc="EKU9003173C9"/>
  <cfdi:Complemento>
    <tfd:TimbreFiscalDigital xmlns:tfd="http://www.sat.gob.mx/TimbreFiscalDigital" xsi:schemaLocation="http://www.sat.gob.mx/TimbreFiscalDigital http://www.sat.gob.mx/sitio_internet/cfd/timbrefiscaldigital/TIMBREFISCALDIGITALV11.XSD" Version="1.1"/>
    <foo:Algo/>
  </cfdi:Complemento>
  <cfdi:Complemento>
    <leyendasFisc:LeyendasFiscales xmlns:leyendasFisc="http://www.sat.gob.mx/leyendasFiscales" version="1.0"/>
  </cfdi:Complemento>
  <cfdi:Addenda>
    <bar:Datos/>
  </cfdi:Addenda>
</cfdi:Comprobante>`

func TestClean(t *testing.T) {
	out, err := Clean([]byte(dirty))
	if err != nil {
		t.Fatal(err)
	}
	root, err := nodes.ParseXML(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, unwanted := range []string{"Addenda", "foo:", "bar:", "example.com", "incompleto"} {
		if strings.Contains(s, unwanted) {
			t.Errorf("el XML limpio no debería contener %q:\n%s", unwanted, s)
		}
	}
	if len(root.ChildrenNamed("cfdi:Complemento")) != 1 {
		t.Fatal("los complementos no se colapsaron")
	}
	complemento := root.FirstChild("cfdi:Complemento")
	if complemento.Len() != 2 || complemento.Child(0).Name() != "tfd:TimbreFiscalDigital" || complemento.Child(1).Name() != "leyendasFisc:LeyendasFiscales" {
		t.Fatalf("orden de complementos inesperado:\n%s", s)
	}
	if got := complemento.Child(0).Attr("xsi:schemaLocation"); got != "http://www.sat.gob.mx/TimbreFiscalDigital http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigitalv11.xsd" {
		t.Fatalf("ubicación no corregida: %s", got)
	}
}

func TestBeforeLoad(t *testing.T) {
	in := `<cfdi:Comprobante xmlns="http://www.sat.gob.mx/cfd/3" xmlns:cfdi="http://www.sat.gob.mx/cfd/3" xmlns:schemaLocation="a b"/>`
	out := string(BeforeLoad([]byte(in)))
	if strings.Contains(out, `xmlns="`) || !strings.Contains(out, `xsi:schemaLocation="a b"`) {
		t.Fatalf("limpieza previa inesperada: %s", out)
	}
}

func TestCleanKeepsCadenaOrigen(t *testing.T) {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	before, err := cadenaorigen.Build(data)
	if err != nil {
		t.Fatal(err)
	}
	cleaned, err := Clean(data)
	if err != nil {
		t.Fatal(err)
	}
	after, err := cadenaorigen.Build(cleaned)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("la limpieza alteró la cadena de origen")
	}
}
