package tfd11

import (
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"testing"
)

func TestTimbre(t *testing.T) {
	stamp := NewTimbreFiscalDigital(Attrs{"UUID": "abc", "FechaTimbrado": "2024-01-01T00:00:00", "SelloCFD": "cfd", "SelloSAT": "sat", "NoCertificadoSAT": "123", "RfcProvCertif": "AAA010101AAA", "Version": "9"})
	expected := `<tfd:TimbreFiscalDigital xmlns:tfd="http://www.sat.gob.mx/TimbreFiscalDigital" xsi:schemaLocation="http://www.sat.gob.mx/TimbreFiscalDigital http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigitalv11.xsd" Version="1.1" UUID="abc" FechaTimbrado="2024-01-01T00:00:00" RfcProvCertif="AAA010101AAA" SelloCFD="cfd" NoCertificadoSAT="123" SelloSAT="sat"/>`
	if stamp.String() != expected {
		t.Fatal(stamp.String())
	}
	root := nodes.New("cfdi:Comprobante", nil)
	if _, ok := FromComprobante(root); ok {
		t.Fatal("timbre inexistente")
	}
	root.AddChild(nodes.New("cfdi:Complemento", nil, stamp.Node))
	found, ok := FromComprobante(root)
	if !ok || found.Node != stamp.Node || found.UUID() != "abc" || found.FechaTimbrado() == "" || found.SelloCFD() != "cfd" || found.SelloSAT() != "sat" || found.NoCertificadoSAT() != "123" || found.RfcProvCertif() != "AAA010101AAA" {
		t.Fatal(found)
	}
}
