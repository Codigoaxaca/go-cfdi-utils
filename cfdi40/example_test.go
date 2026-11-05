package cfdi40_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
	"github.com/codigoaxaca/go-cfdi-utils/estadosat"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// Crear y sellar un CFDI 4.0, el mismo ejemplo del README.
func ExampleCreator() {
	csd, err := credentials.OpenFiles("../testdata/certs/EKU9003173C9.cer", "../testdata/certs/EKU9003173C9.key", "12345678a")
	if err != nil {
		log.Fatal(err)
	}

	creator := cfdi40.NewCreator(cfdi40.Attrs{
		"Serie":             "A",
		"Folio":             "1001",
		"Fecha":             "2024-05-10T12:00:00",
		"FormaPago":         "03",
		"MetodoPago":        "PUE",
		"Moneda":            "MXN",
		"TipoDeComprobante": "I",
		"Exportacion":       "01",
		"LugarExpedicion":   "52000",
	}, cfdi40.WithResolver(nil), cfdi40.WithSchemaValidator(nil))
	creator.PutCertificado(csd.Certificate(), true)

	comprobante := creator.Comprobante()
	comprobante.AddEmisor(cfdi40.Attrs{"RegimenFiscal": "601"})
	comprobante.AddReceptor(cfdi40.Attrs{
		"Rfc":                     "COSC8001137NA",
		"Nombre":                  "CARLOS CORTES SOTO",
		"DomicilioFiscalReceptor": "52000",
		"RegimenFiscalReceptor":   "612",
		"UsoCFDI":                 "G03",
	})
	comprobante.AddConcepto(cfdi40.Attrs{
		"ClaveProdServ": "81111500",
		"Cantidad":      "1",
		"ClaveUnidad":   "E48",
		"Descripcion":   "Desarrollo de software",
		"ValorUnitario": "1000.00",
		"Importe":       "1000.00",
		"ObjetoImp":     "02",
	}).AddTraslado(cfdi40.Attrs{
		"Base":       "1000.00",
		"Impuesto":   "002",
		"TipoFactor": "Tasa",
		"TasaOCuota": "0.160000",
		"Importe":    "160.00",
	})

	creator.AddSumasConceptos(nil, 2)
	if err := creator.AddSello(csd.PrivateKey()); err != nil {
		log.Fatal(err)
	}

	asserts := creator.Validate()
	fmt.Println("errores:", asserts.HasErrors())
	fmt.Println("SELLO08:", asserts.Get("SELLO08").Status)
	fmt.Println("Total:", comprobante.Attr("Total"))
	fmt.Println("Emisor:", comprobante.Emisor().Attr("Nombre"))
	// Output:
	// errores: false
	// SELLO08: OK
	// Total: 1160.00
	// Emisor: ESCUELA KEMPER URGATE
}

// Validar un CFDI recibido.
func ExampleValidator_ValidateXML() {
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		log.Fatal(err)
	}
	validator := cfdi40.NewValidator(cfdi40.WithResolver(nil), cfdi40.WithSchemaValidator(nil))
	asserts, err := validator.ValidateXML(data)
	if err != nil {
		log.Fatal(err)
	}
	for _, code := range []string{"XML01", "SELLO08", "TFDSELLO01"} {
		a := asserts.Get(code)
		fmt.Println(a.Code, a.Status)
	}
	// Output:
	// XML01 OK
	// SELLO08 OK
	// TFDSELLO01 NONE
}

// Agregar un validador propio.
func ExampleValidator_MultiValidator() {
	validator := cfdi40.NewValidator(cfdi40.WithResolver(nil), cfdi40.WithSchemaValidator(nil))
	validator.MultiValidator().Add(validate.ValidatorFunc(
		func(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
			asserts.Put("MISERIE01", "El comprobante tiene serie").
				SetStatus(validate.StatusWhen(c.Attr("Serie") != ""))
		},
	))
	data, _ := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	asserts, _ := validator.ValidateXML(data)
	fmt.Println(asserts.Get("MISERIE01"))
	// Output:
	// OK: MISERIE01 - El comprobante tiene serie
}

// Obtener el texto del código QR de un CFDI timbrado.
func ExampleReadFile() {
	c, err := cfdi40.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		log.Fatal(err)
	}
	params, err := estadosat.RequestParametersFromNode(c.Node)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(params.RfcEmisor, params.RfcReceptor)
	// Output:
	// ISD950921HE5 COSC8001137NA
}
