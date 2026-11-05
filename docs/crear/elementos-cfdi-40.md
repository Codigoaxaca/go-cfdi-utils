# Elementos de CFDI 4.0

El paquete `cfdi40` contiene los elementos de un CFDI 4.0. Un elemento es un
[nodo](../componentes/nodos.md) (`*nodes.Node` incrustado) con métodos de ayuda que conocen
sus hijos, el orden en que el SAT los exige y el orden de sus atributos.

```go
comprobante := cfdi40.NewComprobante(cfdi40.Attrs{"Serie": "A"})
comprobante.AddEmisor(cfdi40.Attrs{"Rfc": "EKU9003173C9"})
fmt.Println(comprobante.Emisor().Attr("Rfc")) // EKU9003173C9
```

Como todos los elementos incrustan `*nodes.Node`, todos los métodos de nodos están disponibles
(`Attr`, `SetAttr`, `SetAttrs`, `RemoveAttr`, `Children`, `SearchNode`, `XML`, ...) y todos
cumplen con la interfaz `nodes.Element`.

## Nomenclatura

| Forma | Comportamiento |
|-------|----------------|
| `Hijo() *Hijo` | Obtiene el nodo **único** `Hijo`; si no existe lo crea. |
| `AddHijo(attrs) *Hijo` (nodo único) | Obtiene (o crea) el nodo único, le escribe los atributos y lo devuelve. |
| `AddHijo(attrs) *Hijo` (nodo repetible) | Crea un nodo nuevo con los atributos, lo agrega y lo devuelve. |
| `MultiHijo(attrs...) *Padre` | Agrega un nodo nuevo por cada mapa de atributos y devuelve el padre. |
| `AddComplemento(e)`, `AddAddenda(e)`, `AddComplementoConcepto(e)` | Reciben un elemento (no atributos) y lo agregan dentro del nodo único contenedor. |

## `Comprobante`

Nodo raíz `cfdi:Comprobante`. `NewComprobante(attrs)` establece los atributos fijos
`xmlns:cfdi`, `xmlns:xsi`, `xsi:schemaLocation` y `Version="4.0"`.
`WrapComprobante(node)` usa un nodo existente (por ejemplo leído de un XML).

- `InformacionGlobal() *InformacionGlobal` / `AddInformacionGlobal(attrs)`: nodo único `InformacionGlobal`.
- `AddCfdiRelacionados(attrs) *CfdiRelacionados`: agrega un nuevo `CfdiRelacionados`.
- `MultiCfdiRelacionados(attrs...) *Comprobante`.
- `Emisor() *Emisor` / `AddEmisor(attrs)`: nodo único `Emisor`.
- `Receptor() *Receptor` / `AddReceptor(attrs)`: nodo único `Receptor`.
- `Conceptos() *Conceptos`: nodo único `Conceptos`.
- `AddConcepto(attrs) *Concepto`: agrega un `Concepto` dentro de `Conceptos`.
- `MultiConcepto(attrs...) *Comprobante`.
- `Impuestos() *Impuestos` / `AddImpuestos(attrs)`: nodo único `Impuestos`.
- `AddTraslado(attrs) *Traslado` / `MultiTraslado(attrs...)`: en `Impuestos/Traslados`.
- `AddRetencion(attrs) *Retencion` / `MultiRetencion(attrs...)`: en `Impuestos/Retenciones`.
- `Complemento() *Complemento` / `AddComplemento(e nodes.Element) *Comprobante`.
- `Addenda() *Addenda` / `AddAddenda(e nodes.Element) *Comprobante`.

Orden de hijos: `InformacionGlobal`, `CfdiRelacionados`, `Emisor`, `Receptor`, `Conceptos`,
`Impuestos`, `Complemento`, `Addenda`.

## `InformacionGlobal`

Nodo `cfdi:InformacionGlobal` (atributos `Periodicidad`, `Meses`, `Año`), se usa en facturas globales.

```go
comprobante.AddInformacionGlobal(cfdi40.Attrs{"Periodicidad": "04", "Meses": "01", "Año": "2024"})
```

## `CfdiRelacionados` y `CfdiRelacionado`

- `AddCfdiRelacionado(attrs) *CfdiRelacionado`
- `MultiCfdiRelacionado(attrs...) *CfdiRelacionados`

```go
comprobante.AddCfdiRelacionados(cfdi40.Attrs{"TipoRelacion": "04"}).MultiCfdiRelacionado(
	cfdi40.Attrs{"UUID": "9F1B5F0B-0B8A-4E5A-9F3C-111111111111"},
	cfdi40.Attrs{"UUID": "9F1B5F0B-0B8A-4E5A-9F3C-222222222222"},
)
```

## `Emisor` y `Receptor`

Nodos `cfdi:Emisor` (`Rfc`, `Nombre`, `RegimenFiscal`, `FacAtrAdquirente`) y `cfdi:Receptor`
(`Rfc`, `Nombre`, `DomicilioFiscalReceptor`, `ResidenciaFiscal`, `NumRegIdTrib`,
`RegimenFiscalReceptor`, `UsoCFDI`).

## `Conceptos`

- `AddConcepto(attrs) *Concepto`
- `MultiConcepto(attrs...) *Conceptos`
- `Items() []*Concepto`: los conceptos existentes.

## `Concepto`

- `Impuestos() *ConceptoImpuestos`: nodo único `Impuestos` del concepto.
- `AddTraslado(attrs) *Traslado` / `MultiTraslado(attrs...) *Concepto`
- `AddRetencion(attrs) *Retencion` / `MultiRetencion(attrs...) *Concepto`
- `ACuentaTerceros() *ACuentaTerceros` / `AddACuentaTerceros(attrs)`
- `AddInformacionAduanera(attrs) *InformacionAduanera` / `MultiInformacionAduanera(attrs...)`
- `AddCuentaPredial(attrs) *CuentaPredial` / `MultiCuentaPredial(attrs...)`
- `ComplementoConcepto() *ComplementoConcepto` / `AddComplementoConcepto(e nodes.Element) *Concepto`
- `AddParte(attrs) *Parte` / `MultiParte(attrs...)`

Orden de hijos: `Impuestos`, `ACuentaTerceros`, `InformacionAduanera`, `CuentaPredial`,
`ComplementoConcepto`, `Parte`.

### `ConceptoImpuestos` vs `Impuestos`

Dentro de un concepto el primer nodo de `Impuestos` es `Traslados`, mientras que en el
comprobante el primero es `Retenciones`. Por eso existen dos tipos:

```text
Comprobante
├── Conceptos
│   └── Concepto
│       └── Impuestos        (ConceptoImpuestos)
│           ├── Traslados
│           └── Retenciones
└── Impuestos                (Impuestos)
    ├── Retenciones
    └── Traslados
```

Ambos tienen `Traslados() *Traslados` y `Retenciones() *Retenciones`.

## `Parte`

- `AddInformacionAduanera(attrs)` / `MultiInformacionAduanera(attrs...)`

## `Traslados` y `Retenciones`

- `Traslados.AddTraslado(attrs) *Traslado` / `MultiTraslado(attrs...)`
- `Retenciones.AddRetencion(attrs) *Retencion` / `MultiRetencion(attrs...)`

Atributos de `Traslado` y `Retencion`: `Base`, `Impuesto`, `TipoFactor`, `TasaOCuota`, `Importe`.

## `Complemento`, `ComplementoConcepto` y `Addenda`

Contenedores de otros elementos. Se usan a través de `AddComplemento`,
`AddComplementoConcepto` y `AddAddenda`, que reciben cualquier `nodes.Element`:

```go
addenda := nodes.New("miempresa:Pedido", nodes.Attrs{
	"xmlns:miempresa": "http://miempresa.example/addenda",
	"Numero":          "PO-1234",
})
comprobante.AddAddenda(addenda)
```

## Constantes

| Constante | Valor |
|-----------|-------|
| `cfdi40.Namespace` | `http://www.sat.gob.mx/cfd/4` |
| `cfdi40.XsdLocation` | `http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd` |
| `cfdi40.Version` | `4.0` |
