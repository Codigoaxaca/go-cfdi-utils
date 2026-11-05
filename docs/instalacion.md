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

- `ext-xsl` o Saxon para la cadena original: el procesamiento XSLT está implementado en Go y los
  archivos XSLT del SAT vienen incluidos.
- OpenSSL o convertir la llave `.key` a PEM: se lee directamente el archivo del SAT.
- `ext-soap` para consultar el estado del CFDI.

## Caché de recursos

Los recursos descargados del SAT se guardan en el directorio de caché del usuario
(`xmlresolver.DefaultLocalPath()`). Consulta [xmlresolver](componentes/xmlresolver.md) para
cambiar su ubicación.

## Imagen de contenedor

```dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /app ./cmd/miapp

FROM alpine:3.20
RUN apk add --no-cache libxml2-utils ca-certificates
COPY --from=build /app /app
ENTRYPOINT ["/app"]
```
