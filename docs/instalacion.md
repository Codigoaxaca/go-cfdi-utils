# Instalación

```shell
go get github.com/codigoaxaca/go-cfdi-utils
```

Requisitos:

- Go 1.24 o superior.
- Ninguna dependencia externa de Go (solo la biblioteca estándar) y sin cgo.

## Herramientas opcionales

| Herramienta | Para qué | Si no está |
|-------------|----------|------------|
| `xmllint` (libxml2) | Validar contra los esquemas XSD del SAT (`XSD01`) | La validación `XSD01` queda en `NONE` |
| Acceso a internet | Descargar los certificados del SAT (`TFDSELLO01`) y los XSD (`XSD01`) la primera vez | Esas validaciones quedan en `NONE` o `ERROR` |

Instalar `xmllint`:

```shell
# macOS: ya viene instalado
# Debian / Ubuntu
apt-get install libxml2-utils
# Alpine
apk add libxml2-utils
```

Lo que **no** necesitas (a diferencia de la versión PHP):

- `ext-xsl` o Saxon para la cadena original: el procesamiento XSLT está implementado en Go.
- OpenSSL o convertir la llave `.key` a PEM: se lee directamente el archivo del SAT.
- `ext-soap` para consultar el estado del CFDI.

## Recursos del SAT

Los XSLT de la cadena original y los esquemas XSD se descargan del SAT la primera vez que se
necesitan. Se guardan en `satresources.DefaultDir()`, dentro del directorio de caché del usuario,
y cada mes se revisa si el SAT publicó cambios. Los certificados del SAT para validar el timbre se
guardan aparte, en `xmlresolver.DefaultLocalPath()`.

Consulta [recursos del SAT](componentes/recursos-sat.md) para:

- cambiar el directorio;
- forzar la actualización o borrar la caché;
- usar la librería en servidores sin internet.

## Imagen de contenedor

```dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /app ./cmd/miapp \
 && go install github.com/codigoaxaca/go-cfdi-utils/cmd/cfdi-sat-resources@latest

FROM alpine:3.20
RUN apk add --no-cache libxml2-utils ca-certificates
COPY --from=build /app /app
COPY --from=build /go/bin/cfdi-sat-resources /usr/local/bin/
# recursos del SAT descargados al construir la imagen; la aplicación los sigue actualizando
RUN cfdi-sat-resources -dir /var/cache/sat prepare
ENTRYPOINT ["/app"]
```

En la aplicación:

```go
satresources.SetDefault(satresources.New(satresources.Config{Dir: "/var/cache/sat"}))
```
