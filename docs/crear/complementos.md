# Complementos

Los complementos se construyen con los paquetes de `complementos/` y se agregan al comprobante
con `AddComplemento`. Siguen la misma nomenclatura que los
[elementos de CFDI 4.0](elementos-cfdi-40.md): `Hijo()` obtiene o crea un nodo único,
`AddHijo(attrs)` agrega y `MultiHijo(attrs...)` agrega varios.

En los complementos el parámetro de atributos de `AddHijo` es opcional (`AddImpuestosDR()` o
`AddImpuestosDR(pagos20.Attrs{...})`).

Cada elemento raíz de complemento establece sus atributos fijos: `xmlns:<prefijo>`,
`xsi:schemaLocation` y la versión. Para dejar las declaraciones en el nodo raíz usa
`creator.MoveSatDefinitionsToComprobante()`.

| Paquete | Complemento | Raíz |
|---------|-------------|------|
| `complementos/pagos20` | Recepción de pagos 2.0 | `pago20:Pagos` |
| `complementos/implocal10` | Impuestos locales 1.0 | `implocal:ImpuestosLocales` |
| `complementos/leyendasfisc10` | Leyendas fiscales 1.0 | `leyendasFisc:LeyendasFiscales` |
| `complementos/cartaporte31` | Carta Porte 3.1 | `cartaporte31:CartaPorte` |
| `complementos/tfd11` | Timbre Fiscal Digital 1.1 (lectura) | `tfd:TimbreFiscalDigital` |

Cada paquete exporta las constantes `Namespace`, `XsdLocation` y `Version`.

## Pagos 2.0

Ver [complemento de pagos 2.0 y su calculadora](pagos20.md).

## Impuestos locales

```go
import "github.com/codigoaxaca/go-cfdi-utils/complementos/implocal10"

impLocal := implocal10.NewImpuestosLocales(nil)
impLocal.AddTrasladoLocal(implocal10.Attrs{
	"ImpLocTrasladado": "ISH",
	"TasadeTraslado":   "3.00",
	"Importe":          "30.00",
})
impLocal.AddRetencionLocal(implocal10.Attrs{
	"ImpLocRetenido":  "CEDULAR",
	"TasadeRetencion": "1.00",
	"Importe":         "10.00",
})
comprobante.AddComplemento(impLocal)

// calcula también TotaldeTraslados y TotaldeRetenciones, y los incluye en el Total del comprobante
creator.AddSumasConceptos(nil, 2)
```

## Leyendas fiscales

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

## Carta Porte 3.1

El paquete `cartaporte31` contiene todos los elementos del complemento (`CartaPorte`,
`RegimenesAduaneros`, `Ubicaciones`, `Ubicacion`, `Domicilio`, `Mercancias`, `Mercancia`,
`Autotransporte`, `TransporteMaritimo`, `TransporteAereo`, `TransporteFerroviario`,
`FiguraTransporte`, `TiposFigura`, etc.), con el orden de hijos que exige el XSD.

```go
import "github.com/codigoaxaca/go-cfdi-utils/complementos/cartaporte31"

cartaPorte := cartaporte31.NewCartaPorte(cartaporte31.Attrs{
	"IdCCP":             "CCC4BE01-ADFA-4DEB-8421-ADD60F0BEDAC",
	"TranspInternac":    "No",
	"TotalDistRec":      "100",
})

ubicaciones := cartaPorte.Ubicaciones()
ubicaciones.AddUbicacion(cartaporte31.Attrs{
	"TipoUbicacion":            "Origen",
	"RFCRemitenteDestinatario": "EKU9003173C9",
	"FechaHoraSalidaLlegada":   "2024-05-10T08:00:00",
}).AddDomicilio(cartaporte31.Attrs{"Estado": "OAX", "Pais": "MEX", "CodigoPostal": "68000"})
ubicaciones.AddUbicacion(cartaporte31.Attrs{
	"TipoUbicacion":            "Destino",
	"RFCRemitenteDestinatario": "COSC8001137NA",
	"FechaHoraSalidaLlegada":   "2024-05-10T12:00:00",
	"DistanciaRecorrida":       "100",
}).AddDomicilio(cartaporte31.Attrs{"Estado": "PUE", "Pais": "MEX", "CodigoPostal": "72000"})

mercancias := cartaPorte.AddMercancias(cartaporte31.Attrs{
	"PesoBrutoTotal":   "100",
	"UnidadPeso":       "KGM",
	"NumTotalMercancias": "1",
})
mercancias.AddMercancia(cartaporte31.Attrs{
	"BienesTransp": "10101500",
	"Descripcion":  "Mercancía",
	"Cantidad":     "1",
	"ClaveUnidad":  "H87",
	"PesoEnKg":     "100",
})
mercancias.AddAutotransporte(cartaporte31.Attrs{"PermSCT": "TPAF01", "NumPermisoSCT": "123456"})

cartaPorte.FiguraTransporte().AddTiposFigura(cartaporte31.Attrs{
	"TipoFigura":    "01",
	"RFCFigura":     "COSC8001137NA",
	"NumLicencia":   "A123456",
	"NombreFigura":  "CARLOS CORTES SOTO",
})

comprobante.AddComplemento(cartaPorte)
```

Los atributos anteriores son ilustrativos; consulta la guía de llenado del SAT para los valores
que aplican a tu caso.

## Timbre Fiscal Digital

El timbre lo agrega el PAC, por lo que el paquete `tfd11` es útil principalmente para leerlo:

```go
import "github.com/codigoaxaca/go-cfdi-utils/complementos/tfd11"

if tfd, ok := tfd11.FromComprobante(comprobante); ok {
	fmt.Println(tfd.UUID(), tfd.FechaTimbrado(), tfd.NoCertificadoSAT())
}
```

## Complementos sin paquete propio

Cualquier complemento se puede construir con [nodos](../componentes/nodos.md). La cadena
original funciona igual porque los XSLT de todos los complementos del SAT vienen incluidos
(Nómina, Comercio Exterior, INE, IEDU, Donatarias, Divisas, Vehículo usado, etc.).

Por ejemplo, el complemento de Instituciones Educativas va dentro de `ComplementoConcepto`:

```go
iedu := nodes.New("iedu:instEducativas", nodes.Attrs{
	"xmlns:iedu":         "http://www.sat.gob.mx/iedu",
	"xsi:schemaLocation": "http://www.sat.gob.mx/iedu http://www.sat.gob.mx/sitio_internet/cfd/iedu/iedu.xsd",
	"version":            "1.0",
	"nombreAlumno":       "JUAN PEREZ",
	"CURP":               "PEPJ000101HOCRRN09",
	"nivelEducativo":     "Primaria",
	"autRVOE":            "123456",
})
concepto.AddComplementoConcepto(iedu)
```

Para fijar el orden de los hijos de un nodo propio usa `SetChildrenOrder` y para el orden de
los atributos `SetAttributesOrder`.
