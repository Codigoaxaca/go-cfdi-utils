# Resolvedor de recursos del SAT (`xmlresolver`)

Algunas tareas necesitan recursos publicados por el SAT:

- Los certificados con los que el SAT (o el PAC) sella el Timbre Fiscal Digital, publicados en
  `https://rdc.sat.gob.mx/rccf/...`, necesarios para la validación `TFDSELLO01`.
- Los esquemas XSD (y los XSD que importan, como los catálogos), necesarios para la validación
  `XSD01`.

`xmlresolver.Resolver` los descarga y los guarda en una caché local para no volver a descargarlos.

> Los archivos XSLT para la cadena original **no** se descargan: vienen incluidos en la librería
> (ver [cadena de origen](cadena-de-origen.md)).

```go
import "github.com/codigoaxaca/go-cfdi-utils/xmlresolver"

r := xmlresolver.New() // caché en xmlresolver.DefaultLocalPath()

// certificado del SAT por número (atributo NoCertificadoSAT del timbre)
cert, err := r.SatCertificate(ctx, "00001000000505211329")

// cualquier recurso
data, err := r.Get(ctx, "http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd")

// XSD con todas sus importaciones descargadas y reescritas a rutas locales
local, err := r.ResolveSchema(ctx, "http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd")
```

## Ubicación de la caché

`xmlresolver.DefaultLocalPath()` devuelve `<directorio de caché del usuario>/go-cfdi-utils/sat`:

- macOS: `~/Library/Caches/go-cfdi-utils/sat`
- Linux: `$XDG_CACHE_HOME/go-cfdi-utils/sat` o `~/.cache/go-cfdi-utils/sat`
- Windows: `%LocalAppData%\go-cfdi-utils\sat`

Los archivos se guardan con la estructura `<caché>/<host>/<ruta de la URL>`, por ejemplo
`<caché>/www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd`.

```go
r := xmlresolver.NewWithLocalPath("/var/cache/mi-app/sat") // caché propia
r := &xmlresolver.Resolver{}                               // sin caché en disco
r := &xmlresolver.Resolver{LocalPath: "...", Client: miClienteHTTP}
```

Para usarlo en la creación o validación de CFDI:

```go
creator := cfdi40.NewCreator(attrs, cfdi40.WithResolver(r))
validator := cfdi40.NewValidator(cfdi40.WithResolver(r))
```

Con `cfdi40.WithResolver(nil)` no se descarga nada y las validaciones que lo necesitan quedan
con estado `NONE`.

## Descargar previamente los recursos

En servidores sin acceso a internet puedes llenar la caché en otro equipo (por ejemplo validando
un CFDI) y copiar el directorio, o descargar los archivos con la misma estructura de carpetas.
