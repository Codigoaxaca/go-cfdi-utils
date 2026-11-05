# Certificados del SAT (`xmlresolver`)

Para validar el sello del Timbre Fiscal Digital (validación `TFDSELLO01`) se necesita el
certificado con el que el SAT o el PAC selló el timbre. El SAT lo publica en
`https://rdc.sat.gob.mx/rccf/...`.

`xmlresolver.Resolver` descarga esos certificados y los guarda en una caché local para no volver
a descargarlos. Un certificado no cambia nunca, así que esta caché no caduca.

> Los recursos que sí pueden cambiar, los XSLT de la cadena original y los esquemas XSD, los
> administra [`satresources`](recursos-sat.md), que revisa cada mes si el SAT publicó versiones
> nuevas.

```go
import "github.com/codigoaxaca/go-cfdi-utils/xmlresolver"

r := xmlresolver.New() // caché en xmlresolver.DefaultLocalPath()

// certificado del SAT por número (atributo NoCertificadoSAT del timbre)
cert, err := r.SatCertificate(ctx, "00001000000505211329")

// cualquier archivo que no cambie
data, err := r.Get(ctx, "https://rdc.sat.gob.mx/rccf/000010/000005/05/21/13/00001000000505211329.cer")
```

## Ubicación de la caché

`xmlresolver.DefaultLocalPath()` devuelve `<directorio de caché del usuario>/go-cfdi-utils/sat`:

- macOS: `~/Library/Caches/go-cfdi-utils/sat`
- Linux: `$XDG_CACHE_HOME/go-cfdi-utils/sat` o `~/.cache/go-cfdi-utils/sat`
- Windows: `%LocalAppData%\go-cfdi-utils\sat`

Los archivos se guardan con la estructura `<caché>/<host>/<ruta de la URL>`.

```go
r := xmlresolver.NewWithLocalPath("/var/cache/mi-app/sat") // caché propia
r := &xmlresolver.Resolver{}                               // sin caché en disco
r := &xmlresolver.Resolver{LocalPath: "...", Client: miClienteHTTP}
```

Para usarlo en la creación o validación de CFDI:

```go
validator := cfdi40.NewValidator(cfdi40.WithResolver(r))
```

Con `cfdi40.WithResolver(nil)` no se descarga nada y `TFDSELLO01` queda con estado `NONE`.

## Servidores sin acceso a internet

Puedes llenar la caché en otro equipo, por ejemplo validando los CFDI timbrados por tus PAC, y
copiar el directorio.
