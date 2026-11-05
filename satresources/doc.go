// Package satresources administra los recursos XML que publica el SAT: las
// hojas de estilo XSLT para generar la cadena original y los esquemas XSD
// para validar CFDI.
//
// Los recursos no van dentro del binario: se descargan del SAT la primera vez
// que se necesitan y se guardan en disco. Cada cierto tiempo (MaxAge, 30 días
// por omisión) se revisa en segundo plano si el SAT publicó cambios usando
// solicitudes condicionales (ETag), así que una revisión sin cambios casi no
// descarga nada.
//
// Una versión nueva nunca se activa sin verificarla: los archivos se descargan
// completos en un directorio temporal, se compilan los XSLT (con el mismo
// intérprete que genera la cadena original), se generan cadenas de prueba y se
// revisa que los XSD estén completos (y que compilen, si xmllint está
// instalado). Solo si todo pasa se cambia la versión activa, de una sola vez;
// si algo falla se sigue usando la versión anterior y el error queda en
// Status y en el registro (slog).
//
// Uso básico (normalmente no hace falta, los paquetes cadenaorigen y
// xmlschema usan Default):
//
//	store := satresources.New(satresources.Config{
//		Dir:    "/var/cache/mi-app/sat", // por omisión DefaultDir()
//		MaxAge: 30 * 24 * time.Hour,     // revisar cambios cada mes
//	})
//	satresources.SetDefault(store)           // usarlo en toda la librería
//	err := store.Prepare(ctx)                // al arrancar: descarga lo que falte
//	res, err := store.Refresh(ctx)           // revisar cambios ahora
//	res, err = store.Refresh(ctx, satresources.Force) // descargar todo de nuevo
//	err = store.Clear()                      // borrar los recursos guardados
//	fmt.Println(store.Status())
//
// Para equipos sin acceso a internet se puede importar el respaldo incluido,
// que se usa solo cuando no hay recursos en disco y no se pueden descargar:
//
//	import _ "github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
//
// El comando cmd/cfdi-sat-resources permite actualizar, consultar y borrar
// los recursos desde cron, un deploy o un Dockerfile.
package satresources
