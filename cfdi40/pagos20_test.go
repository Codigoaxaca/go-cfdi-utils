package cfdi40_test

import (
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/complementos/pagos20"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

// Reproduce testdata/cfdi/created-cfdi40-pago20-valid.xml (creado con la versión PHP).
func TestCreateComprobanteConPagos20(t *testing.T) {
	csd, err := credentials.OpenFiles(certsDir+"EKU9003173C9.cer", certsDir+"EKU9003173C9.key", "12345678a")
	if err != nil {
		t.Fatal(err)
	}
	creator := cfdi40.NewCreator(cfdi40.Attrs{
		"Fecha":             "2023-06-13T14:15:16",
		"TipoDeComprobante": "P",
		"LugarExpedicion":   "52000",
		"Moneda":            "XXX",
		"Exportacion":       "01",
		"SubTotal":          "0",
		"Total":             "0",
	})
	creator.PutCertificado(csd.Certificate(), true)
	comprobante := creator.Comprobante()
	comprobante.AddEmisor(cfdi40.Attrs{"RegimenFiscal": "601"})
	comprobante.AddReceptor(cfdi40.Attrs{
		"Rfc":                     "COSC8001137NA",
		"Nombre":                  "CARLOS CORTES SOTO",
		"RegimenFiscalReceptor":   "605",
		"UsoCFDI":                 "CP01",
		"DomicilioFiscalReceptor": "52000",
	})
	comprobante.AddConcepto(cfdi40.Attrs{
		"ClaveProdServ": "84111506",
		"Cantidad":      "1",
		"ClaveUnidad":   "ACT",
		"Descripcion":   "Pago",
		"ValorUnitario": "0",
		"Importe":       "0",
		"ObjetoImp":     "01",
	})

	pagos := pagos20.NewPagos(nil)
	pago := pagos.AddPago(pagos20.Attrs{
		"FechaPago":       "2023-06-12T17:18:19",
		"FormaDePagoP":    "03",
		"MonedaP":         "MXN",
		"TipoCambioP":     "1",
		"Monto":           "15000.00",
		"NumOperacion":    "963852",
		"RfcEmisorCtaOrd": "BMI9704113PA",
		"CtaOrdenante":    "0001970000",
		"RfcEmisorCtaBen": "BBA830831LJ2",
		"CtaBeneficiario": "0198005000",
	})
	pago.AddDoctoRelacionado(pagos20.Attrs{
		"IdDocumento":      "00000000-1111-2222-3333-00000000000A",
		"MonedaDR":         "MXN",
		"EquivalenciaDR":   "1",
		"NumParcialidad":   "2",
		"ImpSaldoAnt":      "12000.00",
		"ImpPagado":        "12000.00",
		"ImpSaldoInsoluto": "0",
		"ObjetoImpDR":      "02",
	}).AddImpuestosDR().AddTrasladosDR().AddTrasladoDR(pagos20.Attrs{
		"ImpuestoDR":   "002",
		"TipoFactorDR": "Tasa",
		"TasaOCuotaDR": "0.160000",
		"BaseDR":       "10344.83",
		"ImporteDR":    "1655.17",
	})
	pago.AddDoctoRelacionado(pagos20.Attrs{
		"IdDocumento":      "00000000-1111-2222-3333-00000000000B",
		"MonedaDR":         "MXN",
		"EquivalenciaDR":   "1",
		"NumParcialidad":   "1",
		"ImpSaldoAnt":      "10000.00",
		"ImpPagado":        "3000.00",
		"ImpSaldoInsoluto": "7000.00",
		"ObjetoImpDR":      "02",
	}).AddImpuestosDR().AddTrasladosDR().AddTrasladoDR(pagos20.Attrs{
		"ImpuestoDR":   "002",
		"TipoFactorDR": "Tasa",
		"TasaOCuotaDR": "0.160000",
		"BaseDR":       "2586.21",
		"ImporteDR":    "413.79",
	})
	if _, err := pagos.WriteTotales(); err != nil {
		t.Fatal(err)
	}
	comprobante.AddComplemento(pagos)
	creator.MoveSatDefinitionsToComprobante()
	if err := creator.AddSello(csd.PrivateKey()); err != nil {
		t.Fatal(err)
	}

	asserts := creator.Validate()
	for _, a := range asserts.All() {
		if a.Status.IsError() || a.Status.IsWarning() {
			t.Errorf("%s %s %s: %s", a.Status, a.Code, a.Title, a.Explanation)
		}
	}

	expected, err := nodes.ParseXMLFile("../testdata/cfdi/created-cfdi40-pago20-valid.xml")
	if err != nil {
		t.Fatal(err)
	}
	actual, err := nodes.ParseXMLString(creator.AsXML())
	if err != nil {
		t.Fatal(err)
	}
	sameNodes(t, "", expected, actual)
}
