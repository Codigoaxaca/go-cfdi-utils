# Validación de CFDI 4.0

La librería valida tanto los CFDI que creas como los que recibes. Al validar no solo se reportan
los errores: también lo que fue correcto, las advertencias y lo que no se pudo comprobar.
Todos los mensajes de validación están en español.

La lista completa de validaciones está en [validaciones de CFDI 4.0](validaciones-40.md).

## Validación de documentos creados

Si creas el CFDI con `cfdi40.Creator` usa `Validate()`:

```go
asserts := creator.Validate()
if asserts.HasErrors() {
	for _, a := range asserts.Errors() {
		fmt.Printf("%-10s %-6s %s => %s\n", a.Code, a.Status, a.Title, a.Explanation)
	}
}
```

## Validación de documentos recibidos

Para un CFDI recibido usa `cfdi40.Validator`:

```go
data, err := os.ReadFile("cfdi-recibido.xml")
if err != nil { /* ... */ }

validator := cfdi40.NewValidator()
asserts, err := validator.ValidateXML(data)
if err != nil {
	// el contenido no es XML válido
}
if asserts.HasErrors() {
	for _, e := range asserts.Errors() {
		fmt.Printf("%s %s: %s\n", e.Code, e.Title, e.Explanation)
	}
}
```

También existen `validator.ValidateXMLContext(ctx, data)`, `validator.ValidateFile(path)` y el
atajo `cfdi40.ValidateXML(data)`.

Las validaciones trabajan con la información tal como fue recibida. Es frecuente que los CFDI
recibidos tengan addendas sin esquema u otros "agregados" que provocan errores de validación
XSD aunque el CFDI sea válido; en ese caso límpialo antes con el paquete
[`cleaner`](../leer/limpieza-cfdi.md):

```go
limpio, err := cleaner.Clean(data)
asserts, err := cfdi40.NewValidator().ValidateXML(limpio)
```

## Recursos que usan las validaciones

| Validación | Necesita | Opción para configurarlo |
|------------|----------|---------------------------|
| `XSD01` esquemas XSD | `xmllint` instalado y descargar los XSD del SAT | `cfdi40.WithSchemaValidator(...)` |
| `SELLO08` sello del comprobante | generar la cadena original (incluido, sin internet) | `cfdi40.WithCadenaOrigenBuilder(...)` |
| `TFDSELLO01` sello del SAT | descargar el certificado del SAT de `rdc.sat.gob.mx` | `cfdi40.WithResolver(...)` |

Los recursos descargados se guardan en caché (ver [xmlresolver](../componentes/xmlresolver.md)),
por lo que solo se descargan la primera vez. Si alguno no está disponible, la validación
correspondiente queda con estado `NONE` y una explicación.

```go
// sin acceso a internet ni xmllint
validator := cfdi40.NewValidator(cfdi40.WithResolver(nil), cfdi40.WithSchemaValidator(nil))
```

## Status

Un estado puede tener uno de cuatro valores:

| Constante | `String()` | Significado |
|-----------|------------|-------------|
| `validate.StatusError` | `ERROR` | Existe un fallo; el CFDI se debe considerar inválido. |
| `validate.StatusWarning` | `WARN` | Existe un fallo pero se desconoce si es correcto o incorrecto. |
| `validate.StatusOk` | `OK` | Se realizó la prueba y no se encontró fallo. |
| `validate.StatusNone` | `NONE` | La prueba no se realizó (no aplica o faltó algún recurso). |

## Assert

Un `Assert` es un *aseguramiento*: un enunciado afirmativo (no un mensaje de error), por ejemplo
*"El sello del comprobante está en base 64"*. Tiene los campos `Code`, `Title`, `Status` y
`Explanation`.

```go
for _, a := range asserts.All() {
	fmt.Println(a) // OK: SELLO07 - El sello del comprobante está en base 64
}
```

## Asserts

`*validate.Asserts` es la colección ordenada de `Assert`. No puede haber dos con el mismo código;
escribir un código existente actualiza el previo.

| Método | Descripción |
|--------|-------------|
| `All()` | Todos, en orden. |
| `Errors()`, `Warnings()`, `Oks()`, `Nones()` | Filtrados por estado. |
| `HasErrors()`, `HasWarnings()`, `HasStatus(s)` | Informan si existe alguno con ese estado. |
| `Get(code)`, `Exists(code)` | Acceso por código. |
| `FirstError()` | Primer error o `nil`. |
| `MustStop()` | Si un validador pidió detener el resto (lo hace `XSD01` al fallar). |
| `String()` | Reporte de texto, una línea por `Assert`. |

```go
fmt.Print(asserts) // reporte completo
```

## Validadores propios

Un validador implementa `validate.Validator`:

```go
type Validator interface {
	Validate(ctx context.Context, comprobante *nodes.Node, env *validate.Environment, asserts *validate.Asserts)
	CanValidateCfdiVersion(version string) bool
}
```

Para casos simples se puede usar `validate.ValidatorFunc`. Por ejemplo, exigir que todos los CFDI
tengan `Serie`:

```go
validator := cfdi40.NewValidator()
validator.MultiValidator().Add(validate.ValidatorFunc(
	func(ctx context.Context, c *nodes.Node, env *validate.Environment, asserts *validate.Asserts) {
		asserts.Put("MISERIE01", "El comprobante tiene serie").
			SetStatus(validate.StatusWhen(c.Attr("Serie") != ""))
	},
))
asserts, err := validator.ValidateXML(data)
```

`validate.Environment` contiene el XML original, el generador de cadena original, el resolvedor
de recursos y el validador de esquemas, para que los validadores puedan usarlos.

Los validadores de CFDI 4.0 están exportados en el paquete
`validate/cfdi40validators` (`XmlDefinition`, `SelloDigitalCertificado`, `TimbreFiscalDigitalSello`,
etc.) por si quieres armar un `validate.MultiValidator` a la medida:

```go
multi := validate.NewMultiValidator("4.0",
	cfdi40validators.XmlDefinition{},
	cfdi40validators.SelloDigitalCertificado{},
)
```
