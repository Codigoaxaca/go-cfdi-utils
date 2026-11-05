# go-cfdi-utils

Utilerías en Go para trabajar con CFDI 4.0 (Comprobante Fiscal Digital por Internet) de México.

Es una implementación en Go inspirada en la librería PHP
[eclipxe/CfdiUtils](https://github.com/eclipxe13/CfdiUtils): la misma forma sencilla de crear un
CFDI agregando nodos y sellarlo, pero en **Go puro**, sin cgo, sin `ext-xsl` ni OpenSSL.

📖 **[Leer la documentación](docs/index.md)**

## Características

- **Crear CFDI 4.0** agregando nodos con métodos de ayuda que conocen el orden que exige el SAT,
  calcular sumas e impuestos y **sellar** ([crear CFDI 4.0](docs/crear/crear-cfdi-40.md)).
- **Certificados y llaves privadas**: abre directamente los `.cer` y `.key` del SAT (CSD y
  e.firma) con su contraseña, firma, verifica, números de serie, RFC, nombre, vigencia
  ([certificados](docs/componentes/certificados.md)).
- **Cadena original** en Go puro con los XSLT oficiales del SAT incluidos
  ([cadena de origen](docs/componentes/cadena-de-origen.md)).
- **Consulta del estado de un CFDI en el SAT** (vigente, cancelado, cancelable, EFOS)
  ([estado SAT](docs/componentes/estado-sat.md)).
- **Validación** de CFDI creados y recibidos: sello, certificado, timbre fiscal digital, esquemas
  XSD, sumas, impuestos y más ([validación](docs/validar/validacion-cfdi-40.md)).
- **Complementos**: Pagos 2.0 (con calculadora de totales), impuestos locales, leyendas fiscales,
  Carta Porte 3.1 y Timbre Fiscal Digital ([complementos](docs/crear/complementos.md)).
- **Limpieza** de CFDI recibidos con addendas u otros agregados ([limpieza](docs/leer/limpieza-cfdi.md)).

## Instalación

```shell
go get github.com/codigoaxaca/go-cfdi-utils
```

Requiere Go 1.24 o superior. Opcionalmente `xmllint` para validar contra los esquemas XSD
(ver [instalación](docs/instalacion.md)).

## Ejemplo rápido: crear y sellar un CFDI 4.0

```go
csd, err := credentials.OpenFiles("csd.cer", "csd.key", "contraseña")
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
})
creator.PutCertificado(csd.Certificate(), true) // NoCertificado, Certificado, Rfc y Nombre del emisor

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

creator.AddSumasConceptos(nil, 2)              // SubTotal, Total e Impuestos
if err := creator.AddSello(csd.PrivateKey()); err != nil {
	log.Fatal(err)
}

if asserts := creator.Validate(); asserts.HasErrors() {
	fmt.Print(asserts)
	return
}
fmt.Println(creator.AsXML()) // listo para enviar a timbrar con tu PAC
```

## Validar un CFDI recibido

```go
asserts, err := cfdi40.NewValidator().ValidateXML(xmlBytes)
if err != nil {
	log.Fatal(err) // no es XML
}
for _, e := range asserts.Errors() {
	fmt.Printf("%s %s: %s\n", e.Code, e.Title, e.Explanation)
}
```

## Consultar el estado en el SAT

```go
resp, err := estadosat.NewWebService(nil).RequestXML(ctx, xmlBytes)
if err != nil {
	log.Fatal(err)
}
fmt.Println(resp.Estado, resp.EsCancelable, resp.EstatusCancelacion) // Vigente Cancelable sin aceptación
```

## Paquetes

| Paquete | Descripción |
|---------|-------------|
| [`cfdi40`](docs/crear/crear-cfdi-40.md) | Elementos, creador (`Creator`) y validador (`Validator`) de CFDI 4.0 |
| [`credentials`](docs/componentes/certificados.md) | Certificados `.cer`, llaves `.key`, firma y verificación |
| [`cadenaorigen`](docs/componentes/cadena-de-origen.md) | Cadena original con XSLT del SAT en Go puro |
| [`estadosat`](docs/componentes/estado-sat.md) | Consulta de estado de CFDI y texto del código QR |
| [`validate`](docs/validar/validacion-cfdi-40.md), `validate/cfdi40validators` | Infraestructura y validadores |
| [`complementos/...`](docs/crear/complementos.md) | Pagos 2.0, impuestos locales, leyendas fiscales, Carta Porte 3.1, TFD |
| [`nodes`](docs/componentes/nodos.md) | Árbol de nodos XML (base de todo) |
| [`sumasconceptos`](docs/utilerias/sumas-conceptos.md) | Cálculo de subtotal, impuestos y total |
| [`cleaner`](docs/leer/limpieza-cfdi.md) | Limpieza de CFDI recibidos |
| [`xmlresolver`](docs/componentes/xmlresolver.md) | Descarga y caché de recursos del SAT |
| [`xmlschema`](docs/validar/esquemas-xsd.md) | Validación XSD usando `xmllint` |

## Desarrollo

```shell
go test ./...                                  # pruebas (sin red, salvo descargas en caché del SAT)
CFDI_LIVE_TESTS=1 go test ./...                # incluye consultas reales al SAT
go vet ./...
```

## Licencia

Este proyecto se distribuye bajo la [Mozilla Public License 2.0](LICENSE) (MPL-2.0).

En resumen (el texto que aplica es el de la licencia):

- ✅ Puedes usarla en software **comercial** y **privativo**, sin pagar ni pedir permiso.
- ✅ Puedes **modificarla** y distribuirla.
- 🔁 Si distribuyes una versión **modificada de los archivos de esta librería**, debes publicar
  el código fuente de esas modificaciones bajo la misma licencia MPL-2.0. Así las mejoras
  regresan a la comunidad.
- 🔓 Tu propio código que solo *usa* la librería (importa sus paquetes) **no** tiene que
  publicarse ni cambiar de licencia.
- Debes conservar los avisos de licencia y de derechos de autor (ver [NOTICE](NOTICE)).

## Créditos

Basada en el diseño, la documentación y las pruebas de
[eclipxe/CfdiUtils](https://github.com/eclipxe13/CfdiUtils) de Carlos C Soto (licencia MIT, ver
[NOTICE](NOTICE)).
