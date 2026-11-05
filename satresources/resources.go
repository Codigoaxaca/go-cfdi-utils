package satresources

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/codigoaxaca/go-cfdi-utils/internal/xslt"
)

// Ubicaciones de las hojas de estilo de cadena original.
const (
	XsltCfdi40        = "http://www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt"
	XsltCfdi33        = "http://www.sat.gob.mx/sitio_internet/cfd/3/cadenaoriginal_3_3/cadenaoriginal_3_3.xslt"
	XsltTfd11         = "http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/cadenaoriginal_TFD_1_1.xslt"
	XsltRetenciones20 = "http://www.sat.gob.mx/esquemas/retencionpago/2/retenciones.xslt"
)

// DefaultRoots devuelve los recursos que el almacén mantiene siempre (junto
// con todo lo que incluyen o importan): las hojas de estilo de cadena
// original y los esquemas de CFDI 4.0 y de los complementos que esta librería
// sabe crear. Cualquier otro recurso se descarga la primera vez que se pide y
// a partir de ese momento también se mantiene actualizado.
func DefaultRoots() []string {
	return []string{
		XsltCfdi40,
		XsltCfdi33,
		XsltTfd11,
		XsltRetenciones20,
		"http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd",
		"http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigitalv11.xsd",
		"http://www.sat.gob.mx/sitio_internet/cfd/Pagos/Pagos20.xsd",
		"http://www.sat.gob.mx/sitio_internet/cfd/implocal/implocal.xsd",
		"http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd",
		"http://www.sat.gob.mx/sitio_internet/cfd/CartaPorte/CartaPorte31.xsd",
		"http://www.sat.gob.mx/esquemas/retencionpago/2/retencionpagov2.xsd",
	}
}

// Tipos de recurso del manifiesto.
const (
	kindXSLT = "xslt"
	kindXSD  = "xsd"
)

const xsdNS = "http://www.w3.org/2001/XMLSchema"

// Key devuelve la llave de un recurso: host en minúsculas y ruta, sin
// esquema. Las URL http y https de un mismo recurso comparten la llave.
func Key(resource string) (string, error) {
	k, err := xslt.Key(resource)
	if err != nil {
		return "", fmt.Errorf("satresources: %w", err)
	}
	return k, nil
}

// analyze identifica el tipo de recurso, sus dependencias (URL absolutas) y
// el contenido a guardar. En los XSD las referencias a otros esquemas se
// reescriben como rutas relativas para poder usarlos sin red.
func analyze(resource, key string, data []byte) (*fileInfo, []byte, error) {
	kind, err := detectKind(data)
	if err != nil {
		return nil, nil, fmt.Errorf("satresources: %s: %w", resource, err)
	}
	sum := sha256.Sum256(data)
	info := &fileInfo{URL: resource, Kind: kind, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))}
	base, err := url.Parse(resource)
	if err != nil {
		return nil, nil, err
	}
	switch kind {
	case kindXSLT:
		deps, err := xsltDeps(base, data)
		if err != nil {
			return nil, nil, fmt.Errorf("satresources: %s: %w", resource, err)
		}
		info.Deps = deps
		return info, data, nil
	default:
		out, deps, err := rewriteSchema(base, key, data)
		if err != nil {
			return nil, nil, fmt.Errorf("satresources: %s: %w", resource, err)
		}
		info.Deps = deps
		return info, out, nil
	}
}

func newDecoder(data []byte) *xml.Decoder {
	d := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	d.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "utf-8", "utf8", "us-ascii", "ascii":
			return input, nil
		case "iso-8859-1", "latin1", "latin-1":
			raw, err := io.ReadAll(input)
			if err != nil {
				return nil, err
			}
			out := make([]rune, len(raw))
			for i, b := range raw {
				out[i] = rune(b)
			}
			return strings.NewReader(string(out)), nil
		}
		return nil, fmt.Errorf("codificación no soportada %q", charset)
	}
	return d
}

func detectKind(data []byte) (string, error) {
	d := newDecoder(data)
	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return "", errors.New("documento vacío")
			}
			return "", fmt.Errorf("no es XML: %w", err)
		}
		if se, ok := t.(xml.StartElement); ok {
			switch {
			case se.Name.Space == xslt.NS && (se.Name.Local == "stylesheet" || se.Name.Local == "transform"):
				return kindXSLT, checkWellFormed(d)
			case se.Name.Space == xsdNS && se.Name.Local == "schema":
				return kindXSD, checkWellFormed(d)
			}
			return "", fmt.Errorf("no es XSLT ni XSD (raíz {%s}%s)", se.Name.Space, se.Name.Local)
		}
	}
}

func checkWellFormed(d *xml.Decoder) error {
	for {
		_, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("XML mal formado: %w", err)
		}
	}
}

func xsltDeps(base *url.URL, data []byte) ([]string, error) {
	d := newDecoder(data)
	var deps []string
	depth := 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			return deps, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := t.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 && t.Name.Space == xslt.NS && (t.Name.Local == "include" || t.Name.Local == "import") {
				for _, a := range t.Attr {
					if a.Name.Space == "" && a.Name.Local == "href" {
						ref, err := base.Parse(a.Value)
						if err != nil {
							return nil, err
						}
						deps = append(deps, ref.String())
					}
				}
			}
		case xml.EndElement:
			depth--
		}
	}
}

var schemaLocationRe = regexp.MustCompile(`(<(?:[A-Za-z0-9_.-]+:)?(?:import|include|redefine|override)\b[^>]*?\bschemaLocation\s*=\s*)(["'])([^"']+)(["'])`)

func rewriteSchema(base *url.URL, key string, data []byte) ([]byte, []string, error) {
	var deps []string
	var rerr error
	out := schemaLocationRe.ReplaceAllFunc(data, func(m []byte) []byte {
		parts := schemaLocationRe.FindSubmatch(m)
		ref, err := base.Parse(string(parts[3]))
		if err != nil {
			rerr = err
			return m
		}
		depKey, err := Key(ref.String())
		if err != nil {
			rerr = err
			return m
		}
		deps = append(deps, ref.String())
		rel := relPath(key, depKey)
		return []byte(string(parts[1]) + string(parts[2]) + rel + string(parts[4]))
	})
	if rerr != nil {
		return nil, nil, rerr
	}
	return out, deps, nil
}

// relPath devuelve la ruta relativa de toKey vista desde el directorio de fromKey.
func relPath(fromKey, toKey string) string {
	from := strings.Split(path.Dir(fromKey), "/")
	to := strings.Split(toKey, "/")
	i := 0
	for i < len(from) && i < len(to)-1 && from[i] == to[i] {
		i++
	}
	return strings.Repeat("../", len(from)-i) + strings.Join(to[i:], "/")
}
