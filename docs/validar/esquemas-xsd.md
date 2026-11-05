# Validación contra esquemas XSD (`xmlschema`)

La validación `XSD01` revisa que el XML cumpla con los esquemas XSD que declara en sus atributos
`xsi:schemaLocation` (CFDI, complementos, timbre). Es una de las validaciones más útiles porque
revisa la estructura completa contra lo publicado por el SAT, incluyendo los catálogos
(`c_FormaPago`, `c_RegimenFiscal`, `c_ClaveProdServ`, etc.).

Go no tiene un validador de XML Schema en su biblioteca estándar, por lo que el paquete
`xmlschema` usa `xmllint` (de libxml2):

- macOS: viene instalado.
- Debian/Ubuntu: `apt-get install libxml2-utils`.
- Alpine: `apk add libxml2-utils`.
- Windows: incluido en las distribuciones de libxml2 o en Git for Windows/MSYS2.

Cuando `xmllint` está en el `PATH`, `cfdi40.NewCreator` y `cfdi40.NewValidator` lo usan
automáticamente. Si no está, `XSD01` queda con estado `NONE`.

```go
import "github.com/codigoaxaca/go-cfdi-utils/xmlschema"

v, err := xmlschema.NewXmllint(nil) // nil: usa satresources.Default()
if errors.Is(err, xmlschema.ErrXmllintNotFound) { /* ... */ }

if err := v.ValidateSchemas(ctx, data); err != nil {
	var schemaErr *xmlschema.Error
	if errors.As(err, &schemaErr) {
		for _, m := range schemaErr.Messages {
			fmt.Println(m)
		}
	}
}
```

Cómo funciona:

1. Obtiene todos los pares espacio de nombres / XSD de los atributos `xsi:schemaLocation`.
2. Pide cada XSD al almacén [`satresources`](../componentes/recursos-sat.md). El almacén lo
   descarga junto con lo que importa la primera vez y lo guarda con las referencias reescritas a
   rutas locales. También revisa cada mes si el SAT publicó cambios.
3. Ejecuta `xmllint --noout --nonet --schema` con un esquema que importa todos.

Solo se usan los esquemas publicados por el SAT (`sat.gob.mx`). Las ubicaciones de otros
servidores, como las de algunas addendas, se ignoran: no se descargan ni se validan.

La primera validación tarda más porque descarga los catálogos del SAT (varios MB); después se
usan desde disco. Para no esperar en la primera factura, llama a
`satresources.Default().Prepare(ctx)` al arrancar la aplicación.

Si `XSD01` falla, el resto de validaciones ya no se ejecutan (`MustStop`), porque un XML que no
sigue la estructura puede producir resultados confusos.

## Usar otro validador

Cualquier tipo que implemente `validate.SchemaValidator` se puede usar:

```go
type SchemaValidator interface {
	ValidateSchemas(ctx context.Context, xml []byte) error
}

validator := cfdi40.NewValidator(cfdi40.WithSchemaValidator(miValidador))
```
