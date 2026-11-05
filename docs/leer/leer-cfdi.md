# Leer un CFDI

Un CFDI es un documento XML. Para leerlo se convierte en una estructura de
[nodos](../componentes/nodos.md) y, si es CFDI 4.0, se puede envolver en los
[elementos](../crear/elementos-cfdi-40.md) de `cfdi40`.

## CFDI 4.0

```go
comprobante, err := cfdi40.ReadFile("cfdi.xml") // o cfdi40.ReadXML(data)
if err != nil {
	// no es XML válido o no es un CFDI 4.0
	log.Fatal(err)
}

fmt.Println(comprobante.Attr("Serie"), comprobante.Attr("Folio"), comprobante.Attr("Total"))
fmt.Println(comprobante.SearchAttribute("cfdi:Emisor", "Rfc"))
fmt.Println(comprobante.SearchAttribute("cfdi:Receptor", "Nombre"))

for _, concepto := range comprobante.Conceptos().Items() {
	fmt.Println(concepto.Attr("Descripcion"), concepto.Attr("Importe"))
	for _, traslado := range concepto.SearchNodes("cfdi:Impuestos", "cfdi:Traslados", "cfdi:Traslado") {
		fmt.Println("  ", traslado.Attr("Impuesto"), traslado.Attr("Importe"))
	}
}
```

> Los métodos como `Emisor()`, `Impuestos()` o `Complemento()` **crean** el nodo si no existe.
> Para solo leer sin modificar el documento usa `SearchNode`, `SearchNodes` y `SearchAttribute`,
> que devuelven `nil`, una lista vacía o `""` cuando no encuentran el nodo.

## Timbre Fiscal Digital

```go
import "github.com/codigoaxaca/go-cfdi-utils/complementos/tfd11"

if tfd, ok := tfd11.FromComprobante(comprobante); ok {
	fmt.Println(tfd.UUID())
	fmt.Println(tfd.FechaTimbrado())
	fmt.Println(tfd.RfcProvCertif())
	fmt.Println(tfd.NoCertificadoSAT())
}
```

O directamente con nodos:

```go
uuid := comprobante.SearchAttribute("cfdi:Complemento", "tfd:TimbreFiscalDigital", "UUID")
```

## Complementos

```go
pagos := comprobante.SearchNode("cfdi:Complemento", "pago20:Pagos")
if pagos != nil {
	for _, pago := range pagos.ChildrenNamed("pago20:Pago") {
		fmt.Println(pago.Attr("FechaPago"), pago.Attr("Monto"))
	}
}
```

## Certificado del emisor

```go
cert, err := credentials.ParseCertificate([]byte(comprobante.Attr("Certificado")))
fmt.Println(cert.Rfc(), cert.Name(), cert.NoCertificado(), cert.ValidTo())
```

## Cualquier versión o tipo de documento

Para CFDI 3.3, 3.2, retenciones o cualquier XML, usa los nodos directamente:

```go
root, err := nodes.ParseXMLFile("cfdi33.xml")
fmt.Println(root.Name(), root.Attr("Version")) // cfdi:Comprobante 3.3
fmt.Println(root.SearchAttribute("cfdi:Emisor", "Rfc"))
```

`cadenaorigen.Build` genera la cadena original de CFDI 4.0, 3.3, Timbre Fiscal Digital 1.1 y
Retenciones 2.0, y `estadosat` consulta el estado de CFDI 3.2, 3.3 y 4.0.

## Antes de validar

Si el CFDI contiene addendas u otros agregados posteriores al timbrado, conviene
[limpiarlo](limpieza-cfdi.md) antes de validarlo.
