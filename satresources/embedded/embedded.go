// Package embedded incluye en el binario un respaldo de los recursos del SAT
// (XSLT de cadena original y XSD de CFDI 4.0 y complementos) para equipos
// sin acceso a internet.
//
// Basta con importarlo; el respaldo se registra con
// satresources.RegisterFallback:
//
//	import _ "github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
//
// El respaldo solo se usa cuando no hay recursos guardados en disco y no se
// pueden descargar del SAT (o el almacén está en modo Offline). La versión
// que se copia del respaldo se marca como no revisada, de modo que en cuanto
// haya conexión se revisa con el SAT y se reemplaza.
//
// El respaldo agrega unos megabytes al binario (el catálogo catCFDI.xsd es el
// archivo más grande). Se regenera con:
//
//	go generate ./satresources/embedded
package embedded

import (
	"embed"
	"io/fs"

	"github.com/codigoaxaca/go-cfdi-utils/satresources"
)

//go:generate sh ../../scripts/update-embedded.sh

//go:embed all:data
var data embed.FS

// FS devuelve el respaldo: manifest.json y files/<host>/<ruta>.
func FS() fs.FS {
	sub, err := fs.Sub(data, "data")
	if err != nil {
		panic(err)
	}
	return sub
}

func init() { satresources.RegisterFallback(FS()) }
