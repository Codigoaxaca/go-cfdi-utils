# Creación de CFDI 4.0

Para crear un CFDI versión 4.0 se usa `cfdi40.Creator`.

`Creator` trabaja directamente con el elemento `cfdi40.Comprobante` y tiene métodos para
establecer el certificado, calcular las sumas, generar el sello, validar y exportar el XML.
Es el "pegamento" de todas las piezas de la librería.

```go
import "github.com/codigoaxaca/go-cfdi-utils/cfdi40"
```

## Ejemplo completo

```go
package main

import (
	"fmt"
	"log"

	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
)

func main() {
	// 1. Abrir el CSD tal como lo entrega el SAT (.cer, .key y contraseña)
	csd, err := credentials.OpenFiles("EKU9003173C9.cer", "EKU9003173C9.key", "12345678a")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Crear el comprobante con sus atributos
	creator := cfdi40.NewCreator(cfdi40.Attrs{
		"Serie":             "XXX",
		"Folio":             "0000123456",
		"Fecha":             "2023-06-18T19:20:21",
		"FormaPago":         "01", // efectivo
		"MetodoPago":        "PUE", // pago en una sola exhibición
		"Moneda":            "MXN",
		"TipoDeComprobante": "I", // ingreso
		"Exportacion":       "01", // no aplica
		"LugarExpedicion":   "52000",
	})

	// 3. Poner NoCertificado, Certificado y (opcional) Rfc y Nombre del emisor.
	//    En personas morales el nombre se escribe sin el régimen de capital
	//    ("ESCUELA KEMPER URGATE SA DE CV" => "ESCUELA KEMPER URGATE").
	creator.PutCertificado(csd.Certificate(), true)

	comprobante := creator.Comprobante()

	// No agrego Rfc ni Nombre porque PutCertificado ya los estableció
	comprobante.AddEmisor(cfdi40.Attrs{
		"RegimenFiscal": "601", // General de Ley Personas Morales
	})

	comprobante.AddReceptor(cfdi40.Attrs{
		"Rfc":                     "COSC8001137NA",
		"Nombre":                  "CARLOS CORTES SOTO",
		"DomicilioFiscalReceptor": "52000",
		"RegimenFiscalReceptor":   "612",
		"UsoCFDI":                 "G01",
	})

	// 4. Agregar conceptos con sus impuestos
	comprobante.AddConcepto(cfdi40.Attrs{
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
	}).AddTraslado(cfdi40.Attrs{
		"Base":       "1500",
		"Impuesto":   "002", // IVA
		"TipoFactor": "Tasa",
		"TasaOCuota": "0.160000",
		"Importe":    "240",
	})

	// 5. Calcular SubTotal, Descuento, Total y el nodo Impuestos del comprobante
	creator.AddSumasConceptos(nil, 2)

	// 6. Sellar: genera la cadena original y la firma con la llave privada
	if err := creator.AddSello(csd.PrivateKey()); err != nil {
		log.Fatal(err)
	}

	// 7. (opcional) mover las declaraciones xmlns y xsi:schemaLocation al nodo raíz
	creator.MoveSatDefinitionsToComprobante()

	// 8. Validar
	asserts := creator.Validate()
	if asserts.HasErrors() {
		for _, e := range asserts.Errors() {
			fmt.Printf("%s %s: %s\n", e.Code, e.Title, e.Explanation)
		}
		return
	}

	// 9. Obtener o guardar el XML
	fmt.Println(creator.AsXML())
	if err := creator.SaveXML("cfdi.xml"); err != nil {
		log.Fatal(err)
	}
}
```

> El resultado es un *precfdi* sellado, listo para enviarse a tu PAC para su timbrado.

## Métodos de `Creator`

| Método | Descripción |
|--------|-------------|
| `NewCreator(attrs, opts...)` | Crea el `Comprobante` con los atributos fijos (`xmlns:cfdi`, `xmlns:xsi`, `xsi:schemaLocation`, `Version="4.0"`) más los atributos dados. |
| `NewCreatorFromXML(xml, opts...)` / `NewCreatorFromNode(node, opts...)` | Crea el `Creator` a partir de un comprobante existente, útil para modificar un precfdi y volver a sellarlo. |
| `Comprobante() *Comprobante` | Devuelve el nodo raíz. Todos los demás métodos trabajan sobre él. |
| `PutCertificado(cert, putEmisorRfcNombre)` | Establece `NoCertificado` y `Certificado`. Si `putEmisorRfcNombre` es verdadero también `Rfc` y `Nombre` del `Emisor` (sin régimen de capital para personas morales). |
| `BuildSumasConceptos(precision) *sumasconceptos.SumasConceptos` | Calcula las sumas de los conceptos. |
| `AddSumasConceptos(sumas, precision)` | Escribe `SubTotal`, `Descuento`, `Total`, el nodo `Impuestos` (traslados agrupados con su `Base`, exentos y retenciones) y los totales del complemento de impuestos locales. Si `sumas` es `nil` las calcula. |
| `BuildCadenaDeOrigen() (string, error)` | Genera la cadena original del comprobante en su estado actual. |
| `AddSello(key *credentials.PrivateKey) error` | Genera la cadena original, la firma con RSA-SHA256 y la guarda en base64 en `Sello`. Si hay certificado verifica que la llave le pertenezca (`ErrKeyDoesNotBelongToCertificate`). |
| `Sellar(csd *credentials.Credential, putEmisorRfcNombre) error` | Atajo de `PutCertificado` + `AddSello`. |
| `MoveSatDefinitionsToComprobante()` | Mueve las declaraciones `xmlns:*` y `xsi:schemaLocation` del SAT al nodo raíz. |
| `Validate() *validate.Asserts` / `ValidateContext(ctx)` | Valida el comprobante (ver [validación](../validar/validacion-cfdi-40.md)). |
| `AsXML() string` | Devuelve el XML (con declaración y formato). |
| `SaveXML(path) error` | Guarda el XML en un archivo. |

### Opciones

`NewCreator` (y `NewValidator`) aceptan opciones:

```go
creator := cfdi40.NewCreator(attrs,
	cfdi40.WithResolver(xmlresolver.NewWithLocalPath("/var/cache/sat")), // caché de recursos del SAT
	cfdi40.WithSchemaValidator(nil),                                    // desactivar validación XSD
	cfdi40.WithCadenaOrigenBuilder(miBuilder),                           // generador de cadena propio
)
```

| Opción | Predeterminado |
|--------|----------------|
| `WithCadenaOrigenBuilder(b)` | `cadenaorigen.NewBuilder()`: genera la cadena original en Go puro con los XSLT del SAT incluidos en la librería. |
| `WithResolver(r)` | `xmlresolver.New()`: descarga recursos del SAT (certificados, XSD) y los guarda en `~/Library/Caches/go-cfdi-utils/sat` (macOS) o `~/.cache/go-cfdi-utils/sat` (Linux). `nil` desactiva las validaciones que requieren internet. |
| `WithSchemaValidator(v)` | `xmlschema.NewXmllint(...)` si `xmllint` está instalado; `nil` desactiva la validación XSD. |

## Agregar nodos

Los elementos tienen dos familias de métodos (ver [elementos de CFDI 4.0](elementos-cfdi-40.md)):

- `Emisor()`, `Receptor()`, `Impuestos()`, `Complemento()`… obtienen el nodo **único** (lo crean si no existe).
- `AddEmisor(attrs)` escribe atributos en el nodo único y lo devuelve.
- `AddConcepto(attrs)`, `AddTraslado(attrs)`… agregan un nodo **nuevo** (para nodos que se pueden repetir) y lo devuelven.
- `MultiConcepto(attrs...)`, `MultiTraslado(attrs...)`… agregan varios nodos y devuelven el padre, para encadenar.

```go
concepto := comprobante.AddConcepto(cfdi40.Attrs{ /* ... */ })
concepto.AddTraslado(cfdi40.Attrs{ /* IVA */ })
concepto.AddRetencion(cfdi40.Attrs{ /* ISR */ })
concepto.MultiInformacionAduanera(
	cfdi40.Attrs{"NumeroPedimento": "17  24  3420  7010987"},
	cfdi40.Attrs{"NumeroPedimento": "17  24  3420  7010123"},
)
```

Los atributos se escriben con `cfdi40.Attrs` (un `map[string]string`) y también se pueden
leer o modificar directamente porque todo elemento es un [nodo](../componentes/nodos.md):

```go
comprobante.SetAttr("Confirmacion", "ECVH1")
fmt.Println(comprobante.Attr("Total"))
comprobante.RemoveAttr("CondicionesDePago")
```

## Orden de los nodos y de los atributos

El SAT exige en sus XSD un orden específico de los nodos (por ejemplo `Emisor` antes que `Receptor`).
Los elementos de `cfdi40` conocen ese orden, por lo que puedes agregar los nodos en cualquier orden:

```go
comprobante.AddReceptor(...) // se insertará después de Emisor aunque se agregue primero
comprobante.AddEmisor(...)
```

Como Go no conserva el orden de los mapas, los atributos se escriben en el orden en que los define
el Anexo 20 (el mismo de la cadena original) y los desconocidos en orden alfabético. Así el XML
generado siempre es el mismo.

## Complementos

Los complementos se agregan con `AddComplemento` (dentro del único nodo `cfdi:Complemento`):

```go
import "github.com/codigoaxaca/go-cfdi-utils/complementos/leyendasfisc10"

leyendas := leyendasfisc10.NewLeyendasFiscales(nil)
leyendas.AddLeyenda(leyendasfisc10.Attrs{
	"disposicionFiscal": "RESDERAUTH",
	"norma":             "Artículo 2. Fracción IV.",
	"textoLeyenda":      "El software desarrollado se entrega con licencia MIT",
})
comprobante.AddComplemento(leyendas)
```

Consulta [complementos](complementos.md) para Pagos 2.0, impuestos locales, Carta Porte, etc.

## Acerca de las definiciones de espacios de nombres

Si no se llama a `MoveSatDefinitionsToComprobante()` las declaraciones de espacios de nombres
quedan en el nodo que las usa:

```xml
<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" ...>
  <cfdi:Complemento>
    <leyendasFisc:LeyendasFiscales xmlns:leyendasFisc="http://www.sat.gob.mx/leyendasFiscales"
        xsi:schemaLocation="http://www.sat.gob.mx/leyendasFiscales http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd" version="1.0">
```

Al llamarlo se mueven al nodo raíz, como lo indica (como obligatorio) la documentación del SAT:

```xml
<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" xmlns:leyendasFisc="http://www.sat.gob.mx/leyendasFiscales"
    xsi:schemaLocation="http://www.sat.gob.mx/cfd/4 http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd http://www.sat.gob.mx/leyendasFiscales http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd" ...>
  <cfdi:Complemento>
    <leyendasFisc:LeyendasFiscales version="1.0">
```

La ubicación de las declaraciones no forma parte de la cadena original, por lo que moverlas
no altera el sello (se puede hacer antes o después de sellar).

## Texto del código QR

Para la representación impresa se puede obtener el texto del código QR con el paquete
[`estadosat`](../componentes/estado-sat.md):

```go
params, err := estadosat.RequestParametersFromXML([]byte(xmlTimbrado))
fmt.Println(params.Expression())
// https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx?id=...&re=...&rr=...&tt=...&fe=...
```

## Modificar un CFDI existente y volverlo a sellar

```go
creator, err := cfdi40.NewCreatorFromXML(precfdi)
if err != nil { /* ... */ }
creator.Comprobante().SetAttr("Folio", "1001")
err = creator.Sellar(csd, false)
```

## Diferencias con la versión PHP

| PHP (`CfdiUtils\CfdiCreator40`) | Go (`cfdi40.Creator`) |
|---------------------------------|-----------------------|
| `new CfdiCreator40($attrs, $certificado)` | `cfdi40.NewCreator(attrs)` + `PutCertificado(cert, true)` |
| `$creator->comprobante()` | `creator.Comprobante()` |
| `$comprobante->getEmisor()` | `comprobante.Emisor()` |
| `$comprobante->addEmisor([...])` | `comprobante.AddEmisor(cfdi40.Attrs{...})` |
| `$comprobante['Folio'] = '1'` | `comprobante.SetAttr("Folio", "1")` |
| `$creator->addSello('file://key.pem', 'pass')` | `creator.AddSello(privateKey)` (lee directo el `.key` del SAT, sin convertir a PEM) |
| `$creator->asXml()` | `creator.AsXML()` |
| Requiere `ext-xsl` para la cadena original | Cadena original en Go puro, sin dependencias |
