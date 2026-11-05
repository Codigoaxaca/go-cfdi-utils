package cfdi40_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
	"github.com/codigoaxaca/go-cfdi-utils/xmlresolver"
)

const certsDir = "../testdata/certs/"

// sameNodes compara dos estructuras sin importar el orden de los atributos.
func sameNodes(t *testing.T, path string, expected, actual *nodes.Node) {
	t.Helper()
	if expected.Name() != actual.Name() {
		t.Fatalf("%s: se esperaba el nodo %s y se obtuvo %s", path, expected.Name(), actual.Name())
	}
	path = path + "/" + expected.Name()
	ea, aa := expected.AttrsMap(), actual.AttrsMap()
	keys := map[string]bool{}
	for k := range ea {
		keys[k] = true
	}
	for k := range aa {
		keys[k] = true
	}
	var names []string
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if ea[k] != aa[k] {
			t.Errorf("%s@%s: se esperaba %q y se obtuvo %q", path, k, ea[k], aa[k])
		}
	}
	if expected.Len() != actual.Len() {
		t.Fatalf("%s: se esperaban %d hijos y se obtuvieron %d", path, expected.Len(), actual.Len())
	}
	for i := 0; i < expected.Len(); i++ {
		sameNodes(t, path, expected.Child(i), actual.Child(i))
	}
}

func buildCreator(t *testing.T, opts ...cfdi40.Option) (*cfdi40.Creator, *credentials.Credential) {
	t.Helper()
	csd, err := credentials.OpenFiles(certsDir+"EKU9003173C9.cer", certsDir+"EKU9003173C9.key", "12345678a")
	if err != nil {
		t.Fatal(err)
	}
	creator := cfdi40.NewCreator(cfdi40.Attrs{
		"Serie":             "XXX",
		"Folio":             "0000123456",
		"Fecha":             "2023-06-18T19:20:21",
		"FormaPago":         "01",
		"Moneda":            "USD",
		"TipoCambio":        "18.9008",
		"TipoDeComprobante": "I",
		"Exportacion":       "01",
		"LugarExpedicion":   "52000",
	}, opts...)
	creator.PutCertificado(csd.Certificate(), true)

	comprobante := creator.Comprobante()
	comprobante.SetAttr("MetodoPago", "PUE")
	comprobante.AddEmisor(cfdi40.Attrs{"RegimenFiscal": "601"})
	comprobante.AddReceptor(cfdi40.Attrs{
		"Rfc":                     "COSC8001137NA",
		"Nombre":                  "Carlos Cortés Soto",
		"UsoCFDI":                 "G01",
		"RegimenFiscalReceptor":   "612",
		"DomicilioFiscalReceptor": "52000",
	})

	concepto := comprobante.AddConcepto(cfdi40.Attrs{
		"ClaveProdServ":    "52161557",
		"NoIdentificacion": "GAMEPAD007",
		"Cantidad":         "4",
		"ClaveUnidad":      "H87",
		"Unidad":           "PIEZA",
		"Descripcion":      "Portable tetris gamepad pro++",
		"ValorUnitario":    "500",
		"Importe":          "2000",
		"Descuento":        "500",
		"ObjetoImp":        "02",
	})
	concepto.AddTraslado(cfdi40.Attrs{
		"Base": "1500", "Impuesto": "002", "TipoFactor": "Tasa", "TasaOCuota": "0.160000", "Importe": "240",
	})
	concepto.MultiInformacionAduanera(
		cfdi40.Attrs{"NumeroPedimento": "17  24  3420  7010987"},
		cfdi40.Attrs{"NumeroPedimento": "17  24  3420  7010123"},
	)

	comprobante.AddConcepto(cfdi40.Attrs{
		"ClaveProdServ":    "43211914",
		"NoIdentificacion": "SCREEN5004",
		"Cantidad":         "1",
		"ClaveUnidad":      "H87",
		"Unidad":           "PIEZA",
		"Descripcion":      `Pantalla led 3x4" con entrada HDMI`,
		"ValorUnitario":    "1000",
		"Importe":          "1000",
		"ObjetoImp":        "02",
	}).AddTraslado(cfdi40.Attrs{
		"Base": "1000", "Impuesto": "002", "TipoFactor": "Tasa", "TasaOCuota": "0.160000", "Importe": "160",
	})

	comprobante.AddConcepto(cfdi40.Attrs{
		"ClaveProdServ":    "78121603",
		"NoIdentificacion": "FLETE-MX",
		"Cantidad":         "1",
		"ClaveUnidad":      "E48",
		"Unidad":           "SERVICIO",
		"Descripcion":      "Servicio de envío de mercancías",
		"ValorUnitario":    "300",
		"Importe":          "300",
		"ObjetoImp":        "02",
	}).AddTraslado(cfdi40.Attrs{
		"Base": "300", "Impuesto": "002", "TipoFactor": "Tasa", "TasaOCuota": "0.160000", "Importe": "48",
	})

	creator.AddSumasConceptos(nil, 2)
	return creator, csd
}

func TestCreateComprobanteLikePhp(t *testing.T) {
	creator, csd := buildCreator(t, cfdi40.WithResolver(xmlresolver.New()))
	if err := creator.AddSello(csd.PrivateKey()); err != nil {
		t.Fatal(err)
	}

	asserts := creator.Validate()
	for _, a := range asserts.All() {
		t.Logf("%-5s %-14s %s %s", a.Status, a.Code, a.Title, a.Explanation)
	}
	if asserts.HasErrors() || asserts.HasWarnings() {
		t.Fatalf("el comprobante tiene errores o advertencias")
	}
	for _, code := range []string{"XML01", "XML02", "XML03", "SELLO01", "SELLO02", "SELLO03", "SELLO04", "SELLO05", "SELLO06", "SELLO07", "SELLO08"} {
		if a := asserts.Get(code); a == nil || a.Status != validate.StatusOk {
			t.Errorf("%s debería estar OK: %v", code, a)
		}
	}

	xml := creator.AsXML()
	if !strings.HasPrefix(xml, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Fatal("el XML debe iniciar con la declaración")
	}
	expected, err := nodes.ParseXMLFile("../testdata/cfdi/created-with-discounts-40.xml")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := nodes.ParseXMLString(xml)
	if err != nil {
		t.Fatal(err)
	}
	sameNodes(t, "", expected, actual)
}

func TestCadenaDeOrigenAndSello(t *testing.T) {
	creator, csd := buildCreator(t, cfdi40.WithResolver(nil), cfdi40.WithSchemaValidator(nil))
	if err := creator.AddSello(csd.PrivateKey()); err != nil {
		t.Fatal(err)
	}
	cadena, err := creator.BuildCadenaDeOrigen()
	if err != nil {
		t.Fatal(err)
	}
	want := "||4.0|XXX|0000123456|2023-06-18T19:20:21|01|30001000000500003416|3300.00|500.00|USD|18.9008|3248.00|I|01|PUE|52000|EKU9003173C9|ESCUELA KEMPER URGATE|601|COSC8001137NA|Carlos Cortés Soto|52000|612|G01|52161557|GAMEPAD007|4|H87|PIEZA|Portable tetris gamepad pro++|500|2000|500|02|1500|002|Tasa|0.160000|240|17 24 3420 7010987|17 24 3420 7010123|43211914|SCREEN5004|1|H87|PIEZA|Pantalla led 3x4\" con entrada HDMI|1000|1000|02|1000|002|Tasa|0.160000|160|78121603|FLETE-MX|1|E48|SERVICIO|Servicio de envío de mercancías|300|300|02|300|002|Tasa|0.160000|48|2800.00|002|Tasa|0.160000|448.00|448.00||"
	if cadena != want {
		t.Fatalf("cadena inesperada:\n%s\n%s", cadena, want)
	}
	sello, err := csd.SignBase64([]byte(cadena))
	if err != nil {
		t.Fatal(err)
	}
	if creator.Comprobante().Attr("Sello") != sello {
		t.Fatal("el sello no corresponde a la cadena de origen")
	}
}

func TestAddSelloWithKeyFromOtherCertificate(t *testing.T) {
	cert, err := credentials.OpenCertificateFile(certsDir + "30001000000500003456.cer")
	if err != nil {
		t.Fatal(err)
	}
	key, err := credentials.OpenPrivateKeyFile(certsDir+"EKU9003173C9.key.pem", "")
	if err != nil {
		t.Fatal(err)
	}
	creator := cfdi40.NewCreator(nil, cfdi40.WithResolver(nil))
	creator.PutCertificado(cert, true)
	if err := creator.AddSello(key); !errors.Is(err, cfdi40.ErrKeyDoesNotBelongToCertificate) {
		t.Fatalf("se esperaba ErrKeyDoesNotBelongToCertificate, se obtuvo %v", err)
	}
}

func TestBuildCadenaDeOrigenWithoutBuilder(t *testing.T) {
	creator := cfdi40.NewCreator(nil, cfdi40.WithCadenaOrigenBuilder(nil), cfdi40.WithResolver(nil))
	if _, err := creator.BuildCadenaDeOrigen(); err == nil {
		t.Fatal("se esperaba un error")
	}
}

func TestSaveXML(t *testing.T) {
	creator := cfdi40.NewCreator(nil, cfdi40.WithResolver(nil))
	file := filepath.Join(t.TempDir(), "cfdi.xml")
	if err := creator.SaveXML(file); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != creator.AsXML() {
		t.Fatal("el archivo guardado no corresponde")
	}
}

func TestNewComprobanteFixedAttributesAndOrder(t *testing.T) {
	c := cfdi40.NewComprobante(cfdi40.Attrs{"Folio": "1", "Serie": "A"})
	c.AddComplemento(nodes.New("otro:Nodo", nil))
	c.AddReceptor(cfdi40.Attrs{"Rfc": "XAXX010101000"})
	c.AddEmisor(cfdi40.Attrs{"Rfc": "EKU9003173C9"})
	c.AddConcepto(nil)
	c.AddInformacionGlobal(cfdi40.Attrs{"Periodicidad": "01"})
	c.AddCfdiRelacionados(cfdi40.Attrs{"TipoRelacion": "04"}).AddCfdiRelacionado(cfdi40.Attrs{"UUID": "X"})
	c.AddTraslado(cfdi40.Attrs{"Impuesto": "002"})
	c.AddRetencion(cfdi40.Attrs{"Impuesto": "001"})
	want := `<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.sat.gob.mx/cfd/4 http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd" Version="4.0" Serie="A" Folio="1"><cfdi:InformacionGlobal Periodicidad="01"/><cfdi:CfdiRelacionados TipoRelacion="04"><cfdi:CfdiRelacionado UUID="X"/></cfdi:CfdiRelacionados><cfdi:Emisor Rfc="EKU9003173C9"/><cfdi:Receptor Rfc="XAXX010101000"/><cfdi:Conceptos><cfdi:Concepto/></cfdi:Conceptos><cfdi:Impuestos><cfdi:Retenciones><cfdi:Retencion Impuesto="001"/></cfdi:Retenciones><cfdi:Traslados><cfdi:Traslado Impuesto="002"/></cfdi:Traslados></cfdi:Impuestos><cfdi:Complemento><otro:Nodo/></cfdi:Complemento></cfdi:Comprobante>`
	if got := c.String(); got != want {
		t.Fatalf("XML inesperado:\n%s", got)
	}
}

func TestConceptoImpuestosOrder(t *testing.T) {
	concepto := cfdi40.NewConcepto(nil)
	concepto.AddRetencion(cfdi40.Attrs{"Impuesto": "001"})
	concepto.AddTraslado(cfdi40.Attrs{"Impuesto": "002"})
	concepto.AddParte(nil)
	concepto.AddCuentaPredial(cfdi40.Attrs{"Numero": "1"})
	concepto.AddACuentaTerceros(cfdi40.Attrs{"RfcACuentaTerceros": "X"})
	want := `<cfdi:Concepto><cfdi:Impuestos><cfdi:Traslados><cfdi:Traslado Impuesto="002"/></cfdi:Traslados><cfdi:Retenciones><cfdi:Retencion Impuesto="001"/></cfdi:Retenciones></cfdi:Impuestos><cfdi:ACuentaTerceros RfcACuentaTerceros="X"/><cfdi:CuentaPredial Numero="1"/><cfdi:Parte/></cfdi:Concepto>`
	if got := concepto.String(); got != want {
		t.Fatalf("XML inesperado:\n%s", got)
	}
}

func TestValidateReceivedRealCfdi(t *testing.T) {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	asserts, err := cfdi40.NewValidator().ValidateXML(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range asserts.All() {
		t.Logf("%-5s %-14s %s %s", a.Status, a.Code, a.Title, a.Explanation)
	}
	if asserts.HasErrors() {
		t.Fatal("el CFDI real no debería tener errores")
	}
	if a := asserts.Get("TFDSELLO01"); a == nil || a.Status != validate.StatusOk {
		t.Fatalf("TFDSELLO01 debería estar OK: %v", a)
	}
}

func TestValidateDetectsAlteredCfdi(t *testing.T) {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	root, err := nodes.ParseXML(data)
	if err != nil {
		t.Fatal(err)
	}
	root.SearchNode("cfdi:Receptor").SetAttr("Nombre", "OTRO NOMBRE")
	asserts, err := cfdi40.NewValidator(cfdi40.WithSchemaValidator(nil)).ValidateXML([]byte(root.XML()))
	if err != nil {
		t.Fatal(err)
	}
	if a := asserts.Get("SELLO08"); a == nil || a.Status != validate.StatusError {
		t.Fatalf("SELLO08 debería estar en ERROR: %v", a)
	}
}
