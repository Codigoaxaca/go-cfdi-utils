package nodes

import (
	"encoding/base64"
	"errors"
	"testing"
	"unicode/utf16"
)

// utf7 codifica todos los caracteres en UTF-7, de modo que el resultado no
// contiene ningún "<" que un lector de UTF-8 pueda reconocer.
func utf7(s string) string {
	var raw []byte
	for _, u := range utf16.Encode([]rune(s)) {
		raw = append(raw, byte(u>>8), byte(u))
	}
	return "+" + base64.RawStdEncoding.EncodeToString(raw) + "-"
}

const dtdAttack = `<!DOCTYPE a [<!ENTITY % x SYSTEM "file:///etc/passwd"> %x;]>`

func TestParseRejectsDTD(t *testing.T) {
	for name, src := range map[string]string{
		"sin subconjunto":        `<!DOCTYPE a><a/>`,
		"entidad general":        `<!DOCTYPE a [<!ENTITY x "y">]><a/>`,
		"entidad de parámetro":   dtdAttack + `<a/>`,
		"dtd externo":            `<!DOCTYPE a SYSTEM "file:///etc/passwd"><a/>`,
		"con declaración":        `<?xml version="1.0" encoding="UTF-8"?>` + dtdAttack + `<a/>`,
		"con BOM":                "\xef\xbb\xbf" + dtdAttack + `<a/>`,
		"después de comentario":  `<!-- c -->` + dtdAttack + `<a/>`,
		"después de instrucción": `<?p q?>` + dtdAttack + `<a/>`,
		"en minúsculas":          `<!doctype a><a/>`,
		"después de la raíz":     `<a/><!DOCTYPE a>`,
		"dentro de la raíz":      `<a><!ENTITY x "y"></a>`,
		"en latin1":              `<?xml version="1.0" encoding="ISO-8859-1"?>` + dtdAttack + `<a/>`,
	} {
		if _, err := ParseXMLString(src); !errors.Is(err, ErrDTDNotAllowed) {
			t.Errorf("%s: se esperaba ErrDTDNotAllowed, se obtuvo %v", name, err)
		}
	}
}

// encoding/xml solo reconoce la codificación escrita como encoding="...", sin
// espacios, y libxml2 acepta también encoding = "...". Si el documento declara
// UTF-7 con espacios, encoding/xml lo lee como UTF-8 y no ve el DTD, pero
// xmllint sí. Ninguna declaración que los dos puedan leer distinto se acepta.
func TestParseRejectsDeclarationsReadDifferentlyByLibxml2(t *testing.T) {
	hidden := utf7(dtdAttack)
	for name, src := range map[string]string{
		"utf7 sin espacios":     `<?xml version="1.0" encoding="UTF-7"?>` + hidden + `<a/>`,
		"utf7 con espacios":     `<?xml version="1.0" encoding = "UTF-7"?>` + hidden + `<a/>`,
		"utf7 espacio antes":    `<?xml version="1.0" encoding ="UTF-7"?>` + hidden + `<a/>`,
		"utf7 espacio después":  `<?xml version="1.0" encoding= "UTF-7"?>` + hidden + `<a/>`,
		"utf7 con tabulador":    "<?xml version=\"1.0\" encoding\t=\t'UTF-7'?>" + hidden + `<a/>`,
		"utf7 en comentario":    `<?xml version="1.0" encoding = "UTF-7"?><!-- x ` + utf7(`-->`) + hidden + utf7(`<!-- y`) + ` --><a/>`,
		"utf7 en instrucción":   `<?xml version="1.0" encoding = "UTF-7"?><?p ` + utf7(`?>`) + hidden + utf7(`<?q`) + ` ?><a/>`,
		"utf7 con señuelo":      `<?xml version="1.0" encoding = "UTF-7" x="encoding=utf-8"?>` + hidden + `<a/>`,
		"señuelo antes":         `<?xml version="1.0" x="encoding=utf-8" encoding="UTF-7"?>` + hidden + `<a/>`,
		"latin1 con espacios":   `<?xml version="1.0" encoding = "ISO-8859-1"?><a/>`,
		"versión con espacios":  `<?xml version = "1.1"?><a/>`,
		"versión 1.1":           `<?xml version="1.1"?><a/>`,
		"sin versión":           `<?xml encoding="UTF-8"?><a/>`,
		"vacía":                 `<?xml?><a/>`,
		"orden invertido":       `<?xml encoding="UTF-8" version="1.0"?><a/>`,
		"comillas distintas":    `<?xml version="1.0" encoding="UTF-8'?><a/>`,
		"sin separación":        `<?xml version="1.0"encoding="UTF-8"?><a/>`,
		"mayúsculas":            `<?xml version="1.0" Encoding="UTF-7"?>` + hidden + `<a/>`,
		"codificación vacía":    `<?xml version="1.0" encoding=""?><a/>`,
		"utf16 con BOM":         "\xff\xfe<\x00a\x00/\x00>\x00",
		"utf16 sin BOM":         "<\x00a\x00/\x00>\x00",
		"utf16be sin BOM":       "\x00<\x00a\x00/\x00>",
		"ebcdic":                "\x4c\x6f\xa7\x94\x93",
		"decl fuera del inicio": `<a><?xml version="1.0" encoding = "UTF-7"?></a>`,
	} {
		if _, err := ParseXMLString(src); err == nil {
			t.Errorf("%s: se esperaba un error", name)
		}
	}
}

func TestParseAcceptsUsualDeclarations(t *testing.T) {
	for _, decl := range []string{
		`<?xml version="1.0"?>`,
		`<?xml version='1.0'?>`,
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<?xml version="1.0" encoding="utf-8"?>`,
		`<?xml version='1.0' encoding='UTF-8'?>`,
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`,
		`<?xml version="1.0" encoding="UTF-8" standalone='no' ?>`,
		`<?xml version="1.0" standalone="yes"?>`,
		"<?xml version=\"1.0\"\n\tencoding=\"UTF-8\"\r\n?>",
		`<?xml version = "1.0" encoding = "UTF-8"?>`,
		`<?xml version = '1.0' encoding = 'utf-8' standalone = 'yes' ?>`,
		`<?xml version="1.0" encoding="ISO-8859-1"?>`,
		`<?xml version="1.0" encoding="windows-1252"?>`,
	} {
		n, err := ParseXMLString(decl + `<a b="1"/>`)
		if err != nil {
			t.Errorf("%s: %v", decl, err)
			continue
		}
		if n.Name() != "a" || n.Attr("b") != "1" {
			t.Errorf("%s: resultado inesperado %s", decl, n)
		}
	}
}

// Mencionar un DOCTYPE como contenido no es declararlo.
func TestParseAcceptsDoctypeAsContent(t *testing.T) {
	const want = `<!DOCTYPE a>`
	for name, src := range map[string]string{
		"comentario": `<!-- <!DOCTYPE a> --><a/>`,
		"cdata":      `<a><![CDATA[<!DOCTYPE a>]]></a>`,
		"texto":      `<a>&lt;!DOCTYPE a&gt;</a>`,
		"atributo":   `<a b="&lt;!DOCTYPE a&gt;"/>`,
	} {
		n, err := ParseXMLString(src)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		switch name {
		case "cdata", "texto":
			if n.Value() != want {
				t.Errorf("%s: valor inesperado %q", name, n.Value())
			}
		case "atributo":
			if n.Attr("b") != want {
				t.Errorf("%s: atributo inesperado %q", name, n.Attr("b"))
			}
		}
	}
	if _, err := ParseXMLString(`<?xml-stylesheet href="a.xsl"?><a/>`); err != nil {
		t.Errorf("otras instrucciones de procesamiento se deben aceptar: %v", err)
	}
}
