# Recursos del SAT (XSLT y XSD)

Para generar la cadena original se necesitan las hojas de estilo XSLT que publica el SAT, y para
validar un CFDI contra sus esquemas se necesitan los XSD. El SAT los cambia de vez en cuando
(por ejemplo, en abril de 2026 agregó el complemento de hidrocarburos y petrolíferos a
`cadenaoriginal_4_0.xslt`). Si una aplicación sigue usando una versión anterior, la cadena
original no coincide con la que calcula el PAC y el CFDI se rechaza.

El paquete `satresources` se encarga de esto:

- Los recursos **no van dentro del binario**: se descargan del SAT la primera vez que se
  necesitan y se guardan en disco.
- **Cada mes** (configurable) se revisa en segundo plano si el SAT publicó cambios. La revisión
  usa solicitudes condicionales (`ETag`/`If-None-Match`): si nada cambió, el SAT responde
  `304` y no se descarga nada.
- Una versión nueva **nunca se activa sin verificarla**. Primero se descarga completa en un
  directorio temporal y se revisa:
  - se compilan los XSLT con el mismo intérprete que genera la cadena original;
  - se generan cadenas de prueba;
  - se revisa que los XSD estén completos y, si `xmllint` está instalado, que compilen.

  Solo si todo pasa se cambia la versión activa, de una sola vez. Si algo falla, se sigue
  usando la versión anterior y el error queda registrado.
- Si el SAT no responde, se sigue usando la copia guardada.
- Hay métodos para **forzar la actualización** y para **borrar la caché**, y un comando para cron,
  deploys y `Dockerfile`.

En la mayoría de las aplicaciones no hay que hacer nada: `cadenaorigen`, `cfdi40` y `xmlschema`
usan `satresources.Default()` automáticamente.

## Configuración

```go
import "github.com/codigoaxaca/go-cfdi-utils/satresources"

store := satresources.New(satresources.Config{
	Dir:    "/var/cache/mi-app/sat", // por omisión satresources.DefaultDir()
	MaxAge: 30 * 24 * time.Hour,     // cada cuánto revisar cambios (por omisión 30 días)
})
satresources.SetDefault(store) // toda la librería usa este almacén
```

Llama a `SetDefault` al arrancar la aplicación, antes de crear o validar CFDI. También puedes
pasar el almacén solo a un creador o validador con `cfdi40.WithResources(store)`, o a
`cadenaorigen.NewBuilder(cadenaorigen.WithStore(store))` y `xmlschema.NewXmllint(store)`.

| Campo | Por omisión | Descripción |
|-------|-------------|-------------|
| `Dir` | `DefaultDir()` | Directorio donde se guardan los recursos. |
| `MaxAge` | 30 días | Cada cuánto se revisa si el SAT publicó cambios. Un valor negativo desactiva la revisión automática (solo se actualiza con `Refresh` o con el comando). |
| `RetryInterval` | 1 hora | Espera antes de reintentar una revisión automática que falló. |
| `Offline` | `false` | No descargar nunca: solo usar `Dir` o el respaldo incluido. |
| `HTTPClient` | 60 s por solicitud | Cliente HTTP para descargar. |
| `Roots` | | Recursos adicionales que se deben mantener siempre (por ejemplo `nomina12.xsd`). |
| `Logger` | `slog.Default()` | Recibe los avisos de actualizaciones y errores. `slog.New(slog.DiscardHandler)` para silenciarlo. |
| `AllowHost` | `sat.gob.mx` | Servidores de los que se puede descargar. |
| `Fallback` | el de `satresources/embedded` | Respaldo para cuando no hay nada en disco y no se puede descargar. |

`DefaultDir()` es `<directorio de caché del usuario>/go-cfdi-utils/satresources`:

- macOS: `~/Library/Caches/go-cfdi-utils/satresources`
- Linux: `$XDG_CACHE_HOME/go-cfdi-utils/satresources` o `~/.cache/go-cfdi-utils/satresources`
- Windows: `%LocalAppData%\go-cfdi-utils\satresources`

## Qué recursos se mantienen

Siempre se mantienen los de `satresources.DefaultRoots()`, junto con todo lo que incluyen o
importan:

- **XSLT de cadena original:** CFDI 4.0, CFDI 3.3, Timbre Fiscal Digital 1.1 y Retenciones 2.0.
  Con sus inclusiones son más de 50 archivos, porque cubren todos los complementos.
- **XSD:** CFDI 4.0, Timbre Fiscal Digital 1.1, Pagos 2.0, impuestos locales, leyendas fiscales,
  Carta Porte 3.1 y Retenciones 2.0, con sus catálogos.

Cualquier otro recurso se descarga la primera vez que se pide. Por ejemplo, al validar un CFDI con
complemento de nómina se descarga `nomina12.xsd`. A partir de ese momento también se mantiene
actualizado.

## Uso al arrancar la aplicación

```go
ctx := context.Background()
if err := satresources.Default().Prepare(ctx); err != nil {
	log.Printf("recursos del SAT: %v", err)
}
```

`Prepare` descarga solo lo que falte; si todo está guardado, no hace ninguna solicitud. Así la
primera factura no espera la descarga, que tarda unos segundos y ocupa unos 10 MB, la mayor parte
en catálogos.

## Forzar la actualización y borrar la caché

```go
store := satresources.Default()

res, err := store.Refresh(ctx)                    // revisar cambios ahora
res, err = store.Refresh(ctx, satresources.Force) // descargar todo de nuevo, sin usar la caché
err = store.Clear()                               // borrar todo lo guardado
```

- `Refresh` revisa todos los archivos con el SAT. Si hubo cambios, descarga la versión nueva, la
  verifica y la activa. `res.Changed` indica si se activó una versión nueva, y `res.Updated`,
  `res.Added` y `res.Removed` listan las URL que cambiaron.
- `Refresh(ctx, satresources.Force)` descarga todo de nuevo, sin solicitudes condicionales, y
  activa una versión nueva aunque no haya cambios. Es la forma segura de "borrar la caché y
  actualizar": la versión actual se sigue usando hasta que la nueva pasa la verificación, así que
  si el SAT no responde no te quedas sin recursos.
- `Clear` borra todo lo guardado; el siguiente uso descarga de nuevo. Si hay una actualización en
  curso, espera a que termine.

Si `Refresh` falla, devuelve el error y se sigue usando la versión actual. Si la versión nueva no
pasó la verificación, el error envuelve `satresources.ErrVerification`:

```go
if _, err := store.Refresh(ctx); errors.Is(err, satresources.ErrVerification) {
	// el SAT publicó algo que esta versión de la librería no sabe interpretar:
	// se sigue usando la versión anterior; conviene actualizar go-cfdi-utils
}
```

## Estado

```go
st := satresources.Default().Status()
fmt.Println(st.Version, st.CheckedAt, st.NextCheck, st.LastError)
fmt.Print(st) // en texto legible
```

```text
directorio: /home/app/.cache/go-cfdi-utils/satresources
versión: 20261105T041938Z-3cbf75
archivos: 67
descargada: 2026-11-04 22:19:38 CST
revisada: 2026-11-04 22:19:38 CST
origen: SAT
próxima revisión: 2026-12-04 22:19:38 CST
```

`LastError` contiene el último error de actualización: el SAT no respondió, o una versión nueva
no pasó la verificación. Vuelve a `nil` cuando una actualización termina bien. Los mismos avisos
se registran con `slog`.

## Comando `cfdi-sat-resources`

```shell
go install github.com/codigoaxaca/go-cfdi-utils/cmd/cfdi-sat-resources@latest

cfdi-sat-resources status            # versión activa y cuándo se revisó
cfdi-sat-resources update            # descarga lo que falte y revisa cambios
cfdi-sat-resources update -force     # descarga todo de nuevo
cfdi-sat-resources prepare           # solo descarga lo que falte
cfdi-sat-resources clear             # borra los recursos guardados
cfdi-sat-resources export DESTINO    # copia la versión activa
```

Usa `-dir DIR` antes del comando para indicar el directorio; por omisión es el mismo que usa la
librería con el usuario actual. Termina con código 0 si todo salió bien, 1 si hubo un error y 3
si la versión nueva no pasó la verificación.

Por ejemplo, para revisar cada mes desde cron, aunque la aplicación ya lo hace sola:

```cron
0 3 1 * * cfdi-sat-resources -dir /var/cache/mi-app/sat -q update
```

## Docker y servidores sin internet

**Descargar al construir la imagen.** Así el contenedor arranca con los recursos listos, y si
tiene internet los sigue actualizando:

```dockerfile
RUN go install github.com/codigoaxaca/go-cfdi-utils/cmd/cfdi-sat-resources@latest \
 && cfdi-sat-resources -dir /var/cache/sat prepare
```

```go
satresources.SetDefault(satresources.New(satresources.Config{Dir: "/var/cache/sat"}))
```

Si el sistema de archivos del contenedor no es persistente, las actualizaciones se pierden al
reiniciar, pero se vuelven a revisar. Para conservarlas, monta `Dir` como volumen; varios
contenedores o procesos pueden compartir el mismo directorio.

**Sin acceso a internet.** Usa `Offline: true` y llena el directorio con
`cfdi-sat-resources export` desde otro equipo, o importa el respaldo incluido:

```go
import _ "github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
```

El respaldo agrega unos 10 MB al binario y se usa solo cuando no hay nada en disco y no se puede
descargar. La copia que se toma del respaldo queda marcada como "no revisada": en cuanto haya
conexión se revisa con el SAT y se reemplaza. Si no importas el paquete, no se incluye nada en el
binario.

## Cómo se guarda

```text
<Dir>/current                              versión activa
<Dir>/snapshots/<versión>/manifest.json    URL, ETag, SHA-256 y dependencias de cada archivo
<Dir>/snapshots/<versión>/files/www.sat.gob.mx/sitio_internet/...
<Dir>/.lock                                candado entre procesos
```

- **Activación:** cada versión es un directorio completo. Para activarla se reescribe `current`
  de forma atómica, así que un proceso que se interrumpe a la mitad nunca deja una versión mezclada.
- **Versión anterior:** se conserva, para los procesos que todavía la estén leyendo. Las más
  antiguas se borran al activar otra versión, si tienen más de una hora.
- **XSD:** se guardan con sus `schemaLocation` reescritos como rutas relativas, para que `xmllint`
  los use sin red. El SHA-256 del manifiesto es el del archivo original del SAT.
- **Candado:** el archivo `.lock` evita que dos procesos actualicen al mismo tiempo. Es el
  candado del sistema operativo (`flock` en Linux y macOS, apertura exclusiva en Windows): si el
  proceso que lo tenía muere, el sistema lo libera de inmediato.
- **Lectura:** los procesos solo leen y no esperan el candado, salvo cuando tienen que descargar
  algo que falta.

## Qué verifica antes de activar una versión

1. Que cada archivo sea XSLT o XSD bien formado. Un error del servidor que llegue como página
   HTML se rechaza. También se rechaza cualquier archivo con `DOCTYPE`, porque `xmllint` procesaría
   su DTD y podría abrir archivos locales.
2. Que estén todos los archivos que se incluyen o importan.
3. Que los XSLT compilen con el intérprete de la librería. El intérprete revisa todas las
   plantillas y rechaza cualquier instrucción, atributo o expresión XPath que no sepa interpretar,
   en lugar de generar en silencio una cadena incorrecta.
4. Que las hojas de estilo de CFDI 4.0, CFDI 3.3, TFD 1.1 y Retenciones 2.0 generen cadenas de
   prueba con la forma esperada.
5. Si `xmllint` está instalado, que cada XSD compile. Si `xmllint` falla por otra razón (se
   interrumpe o tarda demasiado), la versión tampoco se activa.

Lo mismo aplica a un recurso que se descarga por primera vez bajo demanda: no se usa hasta que
pasa la verificación. Las redirecciones del servidor solo se siguen hacia servidores permitidos.
