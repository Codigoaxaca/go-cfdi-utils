# Consulta del estado de un CFDI en el SAT

El SAT cuenta con un servicio web para consultar el estado de un CFDI:
<https://consultaqr.facturaelectronica.sat.gob.mx/ConsultaCFDIService.svc>

El paquete `estadosat` lo consume con `net/http` (sin dependencias de SOAP).

```go
import "github.com/codigoaxaca/go-cfdi-utils/estadosat"
```

## Consulta a partir de un archivo XML

```go
data, err := os.ReadFile("cfdi.xml")
if err != nil { /* ... */ }

ws := estadosat.NewWebService(nil)
resp, err := ws.RequestXML(context.Background(), data)
if err != nil { /* error de red o respuesta inválida */ }

fmt.Println(resp.CodigoEstatus)      // S - Comprobante obtenido satisfactoriamente.
fmt.Println(resp.Estado)             // Vigente
fmt.Println(resp.EsCancelable)       // Cancelable sin aceptación
fmt.Println(resp.EstatusCancelacion) // (vacío)
fmt.Println(resp.ValidacionEFOS)     // 200
```

## Consulta a partir de datos conocidos

```go
params, err := estadosat.NewRequestParameters(
	"4.0",                                  // versión del CFDI
	"EKU9003173C9",                         // RFC emisor
	"COSC8001137NA",                        // RFC receptor
	"1,234.5678",                           // total (puede contener comas de millares)
	"CEE4BE01-ADFA-4DEB-8421-ADD60F0BEDAC", // UUID
	"... abcfe/1234==",                     // sello (se usan los últimos 8 caracteres)
)
if err != nil { /* versión no soportada */ }

resp, err := estadosat.NewWebService(nil).Request(ctx, params)
```

## Datos que se requieren

El servicio recibe la *expresión impresa*, la misma información del código QR:

- CFDI 3.3 y 4.0: `https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx?id=UUID&re=RFC&rr=RFC&tt=TOTAL&fe=8CARACTERES`
- CFDI 3.2: `?re=RFC&rr=RFC&tt=0000001234.567800&id=UUID`

`params.Expression()` (o su alias `params.QRText()`) genera esta expresión, por lo que también
sirve para generar el código QR de la representación impresa.

### Acerca del total

El Anexo 20 establece que el total se expresa con hasta 6 decimales omitiendo los ceros no
significativos, pero sin dejar vacío ningún grupo:

| Total | Expresión |
|-------|-----------|
| 0.99 | `tt=0.99` |
| 1 | `tt=1.0` |
| 0 | `tt=0.0` |
| 1234.5678 | `tt=1234.5678` |

## Configuración

```go
ws := estadosat.NewWebService(&estadosat.Config{
	Timeout:    20 * time.Second,         // por omisión 10 segundos
	ServiceURL: "https://mi-proxy/...",   // por omisión DefaultServiceURL
	HTTPClient: miClienteHTTP,            // proxy, TLS, transporte propio, etc.
})
```

A diferencia de la versión PHP (que usa `ext-soap`), aquí puedes pasar tu propio
`*http.Client`, por lo que es sencillo usar un proxy, certificados propios o pruebas con
`httptest`. Todas las consultas reciben un `context.Context` para cancelarlas.

## Datos que entrega la consulta

### `CodigoEstatus` (estado de la consulta)

- `S - Comprobante obtenido satisfactoriamente.`
- `N - 601: La expresión impresa proporcionada no es válida.`
- `N - 602: Comprobante no encontrado.`

### `Estado` (estado del CFDI)

- `Vigente`, `Cancelado` o `No Encontrado`.

### `EsCancelable`

- `No cancelable`, `Cancelable sin aceptación` o `Cancelable con aceptación`.

### `EstatusCancelacion`

- vacío (no se ha solicitado), `Cancelado sin aceptación`, `En proceso`, `Plazo vencido`,
  `Cancelado con aceptación` o `Solicitud rechazada`.

### `ValidacionEFOS`

- `200`: el emisor no se encontró en el listado de EFOS.
- `100`: el emisor se encontró en el listado de EFOS.

### Métodos de ayuda

| Método | Devuelve verdadero cuando |
|--------|---------------------------|
| `ResponseWasOk()` | `CodigoEstatus` inicia con `S - ` |
| `IsVigente()` | `Estado == "Vigente"` |
| `IsCancelado()` | `Estado == "Cancelado"` |
| `IsNoEncontrado()` | `Estado == "No Encontrado"` |
| `IsCancelable()` | `EsCancelable` inicia con `Cancelable` |
| `RequiresAcceptance()` | `EsCancelable == "Cancelable con aceptación"` |
| `IsEfosListed()` | `ValidacionEFOS == "100"` |

## Estados mutuamente excluyentes

| `CodigoEstatus` | `Estado`  | `EsCancelable`            | `EstatusCancelacion`     | Explicación                                           |
|-----------------|-----------|---------------------------|--------------------------|-------------------------------------------------------|
| N - ...         | *         | *                         | *                        | El SAT no sabe del CFDI con los datos ofrecidos       |
| S - ...         | Cancelado | *                         | Plazo vencido            | Cancelado por plazo vencido                           |
| S - ...         | Cancelado | *                         | Cancelado con aceptación | Cancelado con aceptación del receptor                 |
| S - ...         | Cancelado | *                         | Cancelado sin aceptación | No fue requerido preguntarle al receptor y se canceló |
| S - ...         | Vigente   | No cancelable             | *                        | No se puede cancelar                                  |
| S - ...         | Vigente   | Cancelable sin aceptación | *                        | Se puede cancelar pero no se ha realizado solicitud   |
| S - ...         | Vigente   | Cancelable con aceptación | (ninguno)                | Se puede cancelar pero no se ha realizado solicitud   |
| S - ...         | Vigente   | Cancelable con aceptación | En proceso               | Se hizo la solicitud y se está en espera              |
| S - ...         | Vigente   | Cancelable con aceptación | Solicitud rechazada      | Se hizo la solicitud y fue rechazada                  |

## Pruebas

Las pruebas del paquete usan un servidor falso (`httptest`). Para consultar el servicio real:

```shell
CFDI_LIVE_TESTS=1 go test ./estadosat/ -run Live -v
```
