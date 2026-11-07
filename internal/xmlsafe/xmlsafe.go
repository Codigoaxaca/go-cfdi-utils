// Package xmlsafe contiene las reglas que debe cumplir un XML que se le va a
// entregar a libxml2 (xmllint).
//
// xmllint procesa el DTD del documento y abre los archivos que este indique,
// aun con --nonet, y no tiene una opción para impedirlo en todas sus versiones.
// Por eso los documentos con DTD se rechazan antes de llamarlo.
//
// Para que ese rechazo sirva, encoding/xml y libxml2 deben leer el mismo
// documento. No siempre ocurre: encoding/xml solo reconoce la codificación
// escrita como encoding="...", sin espacios, y libxml2 acepta también
// encoding = "...". Un documento que declara UTF-7 con espacios se lee como
// UTF-8 en Go, donde su DTD parece texto sin importancia, y como UTF-7 en
// libxml2, donde es un DTD. Por eso también se rechaza cualquier declaración
// XML que los dos puedan interpretar de forma distinta.
package xmlsafe

import (
	"encoding/xml"
	"errors"
	"regexp"
	"strings"
)

// ErrDTD indica que el documento contiene una declaración DOCTYPE o alguna otra
// directiva, como <!ENTITY>.
var ErrDTD = errors.New("no se permiten declaraciones DOCTYPE")

// ErrDeclaration indica que la declaración <?xml ...?> no es válida o que
// encoding/xml y libxml2 podrían interpretarla de forma distinta.
var ErrDeclaration = errors.New("declaración XML no admitida")

// declaration es la producción XMLDecl del estándar XML 1.0 para el contenido
// de <?xml ...?>. El grupo 1 es el signo igual de la codificación con sus
// espacios; los grupos 2 y 3 son la codificación, según el tipo de comillas.
var declaration = regexp.MustCompile(`^version` + eq + `(?:"1\.0"|'1\.0')` +
	`(?:` + space + `+encoding(` + eq + `)(?:"(` + encName + `)"|'(` + encName + `)'))?` +
	`(?:` + space + `+standalone` + eq + `(?:"(?:yes|no)"|'(?:yes|no)'))?` +
	space + `*$`)

const (
	space   = `[ \t\r\n]`
	eq      = space + `*=` + space + `*`
	encName = `[A-Za-z][A-Za-z0-9._-]*`
)

// CheckToken devuelve un error si el token obtenido de un xml.Decoder no se
// debe aceptar. Se debe llamar con todos los tokens del documento.
func CheckToken(tok xml.Token) error {
	switch t := tok.(type) {
	case xml.Directive:
		return ErrDTD
	case xml.ProcInst:
		if t.Target == "xml" {
			return checkDeclaration(t.Inst)
		}
	}
	return nil
}

func checkDeclaration(inst []byte) error {
	m := declaration.FindSubmatch(inst)
	if m == nil {
		return ErrDeclaration
	}
	// encoding/xml solo aplica la codificación escrita exactamente como
	// encoding="...". Con cualquier espacio junto al igual lee el documento
	// como UTF-8, lo cual solo coincide con libxml2 si esa es justamente la
	// codificación declarada.
	spaced := len(m[1]) > 1
	encoding := string(m[2]) + string(m[3])
	if spaced && !strings.EqualFold(encoding, "utf-8") {
		return ErrDeclaration
	}
	return nil
}
