# Cadena de origen (`cadenaorigen`)

El SAT agrupa en una forma y orden determinados la información de un CFDI que no debe ser
alterada: la *cadena original*. Esa cadena se firma con la llave privada para obtener el sello,
que se verifica con el certificado.

Si algún dato que forma parte de la cadena es modificado, el sello deja de coincidir. Lo que no
forma parte de la cadena (por ejemplo la `Addenda`, el formato del XML o la ubicación de las
declaraciones de espacios de nombres) se puede modificar sin invalidar el sello.

## Cómo se genera

El SAT define la cadena original con archivos XSLT 2.0. Go no tiene un procesador XSLT, así que
el paquete `cadenaorigen` incluye:

- Un intérprete en Go puro del subconjunto de XSLT que usan los archivos del SAT
  (`template`, `apply-templates`, `for-each`, `call-template`, `if`, `value-of`,
  `normalize-space`, reglas predeterminadas, etc.).

Los archivos XSLT del SAT (CFDI 4.0, CFDI 3.3, Timbre Fiscal Digital 1.1, Retenciones 2.0 y todos
sus complementos: Pagos, Carta Porte, Nómina, Comercio Exterior, etc.) **no van dentro del
binario**. Los administra el paquete [`satresources`](recursos-sat.md), que:

- los descarga del SAT la primera vez y los guarda en disco;
- revisa cada mes si el SAT publicó cambios;
- verifica cada versión nueva antes de usarla.

No requiere `xsltproc`, Saxon ni cgo. Las pruebas comparan el resultado contra `xsltproc` y
contra la versión PHP.

```go
import "github.com/codigoaxaca/go-cfdi-utils/cadenaorigen"

cadena, err := cadenaorigen.Build(xmlBytes) // detecta CFDI 4.0, 3.3, TFD 1.1 o Retenciones 2.0
```

## Generar la cadena de un comprobante

Normalmente no necesitas generarla tú: `cfdi40.Creator.AddSello` y las validaciones la generan
automáticamente. Si la necesitas:

```go
// desde un Creator
cadena, err := creator.BuildCadenaDeOrigen()

// desde un XML, indicando el XSLT
builder := cadenaorigen.NewBuilder()
cadena, err := builder.Build(xmlBytes, cadenaorigen.XsltCfdi40)

// saber qué XSLT corresponde
location, err := cadenaorigen.LocationFor(xmlBytes)
```

Constantes con las ubicaciones oficiales:

| Constante | Documento |
|-----------|-----------|
| `cadenaorigen.XsltCfdi40` | CFDI 4.0 |
| `cadenaorigen.XsltCfdi33` | CFDI 3.3 |
| `cadenaorigen.XsltTfd11` | Timbre Fiscal Digital 1.1 |
| `cadenaorigen.XsltRetenciones20` | CFDI de Retenciones e Información de Pagos 2.0 |

Si el documento no es de un tipo soportado se devuelve un error que envuelve
`cadenaorigen.ErrUnsupportedDocument`.

## Cadena de origen del Timbre Fiscal Digital

La cadena del timbre se muestra en la representación impresa del CFDI:

```go
c, err := cfdi40.ReadFile("cfdi.xml")
tfd := c.SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital")
cadenaTfd, err := cadenaorigen.BuildTFD([]byte(cfdi40validators.TfdXML(tfd)))
// ||1.1|UUID|FechaTimbrado|RfcProvCertif|SelloCFD|NoCertificadoSAT||
```

`BuildTFD` requiere que el XML del timbre declare su espacio de nombres (`xmlns:tfd`); la
función `cfdi40validators.TfdXML` lo agrega si el nodo lo hereda del comprobante.

## De dónde salen los XSLT

Por omisión se usan los de `satresources.Default()`. Cuando el almacén activa una versión nueva
de los recursos, la siguiente cadena se genera con ella sin reiniciar la aplicación: el `Builder`
compila cada XSLT una vez por versión y se puede usar de forma concurrente.

```go
// un almacén distinto del predeterminado
builder := cadenaorigen.NewBuilder(cadenaorigen.WithStore(store))

// archivos propios fijos, sin descargas ni actualizaciones
builder = cadenaorigen.NewBuilder(
	cadenaorigen.WithFS(os.DirFS("/srv/sat-xslt")), // <dir>/www.sat.gob.mx/sitio_internet/...
)
creator := cfdi40.NewCreator(attrs, cfdi40.WithCadenaOrigenBuilder(builder))
```

Para forzar la actualización, ver el estado o usar la librería sin internet, consulta
[recursos del SAT](recursos-sat.md).

### Revisión estricta

El intérprete revisa todas las plantillas al compilar. Si una hoja de estilo usa algo que no sabe
interpretar (otra instrucción, un predicado `[...]`, una función de XPath, un modo, etc.),
devuelve un error en lugar de generar en silencio una cadena incorrecta. Por eso una versión
nueva del SAT con algo así no se activa: se sigue usando la anterior hasta actualizar la librería.

## Interfaz `Builder`

```go
type Builder interface {
	Build(xml []byte, xsltLocation string) (string, error)
}
```

Puedes implementar tu propio generador (por ejemplo usando Saxon) y usarlo con
`cfdi40.WithCadenaOrigenBuilder`.
