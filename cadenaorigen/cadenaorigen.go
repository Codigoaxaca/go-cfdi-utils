// Package cadenaorigen genera cadenas originales del SAT sin herramientas
// externas, con un intérprete en Go del subconjunto de XSLT que usan las hojas
// de estilo del SAT.
//
// Las hojas de estilo no van dentro del binario: se obtienen del almacén
// satresources (por omisión satresources.Default()), que las descarga del SAT,
// las guarda en disco y revisa periódicamente si cambiaron.
package cadenaorigen

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"github.com/codigoaxaca/go-cfdi-utils/internal/xslt"
	"github.com/codigoaxaca/go-cfdi-utils/satresources"
)

// Ubicaciones oficiales de las hojas de estilo de cadena original.
const (
	XsltCfdi40        = satresources.XsltCfdi40
	XsltCfdi33        = satresources.XsltCfdi33
	XsltTfd11         = satresources.XsltTfd11
	XsltRetenciones20 = satresources.XsltRetenciones20
)

// ErrUnsupportedDocument indica un documento o versión no reconocido.
var ErrUnsupportedDocument = errors.New("cadenaorigen: documento no soportado")

// Builder transforma XML mediante una ubicación XSLT.
type Builder interface {
	Build(xml []byte, xsltLocation string) (string, error)
}

// XsltBuilder genera cadenas originales. Compila cada hoja de estilo una vez
// por versión de los recursos: cuando el almacén activa una versión nueva, la
// siguiente cadena se genera con ella. Es seguro para uso concurrente y su
// valor cero es utilizable (usa satresources.Default()).
type XsltBuilder struct {
	store *satresources.Store
	fsys  fs.FS

	mu      sync.Mutex
	version string
	cache   map[string]*xslt.Stylesheet
}

// Option configura un constructor antes de su uso.
type Option func(*XsltBuilder)

// WithStore usa el almacén de recursos indicado en lugar de satresources.Default().
func WithStore(store *satresources.Store) Option {
	return func(b *XsltBuilder) { b.store = store }
}

// WithFS usa únicamente las hojas de estilo del sistema de archivos indicado,
// con la estructura <host>/<ruta> (por ejemplo
// www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt).
// No se descarga ni se actualiza nada.
func WithFS(fsys fs.FS) Option {
	return func(b *XsltBuilder) { b.fsys = fsys }
}

// NewBuilder crea un constructor.
func NewBuilder(opts ...Option) *XsltBuilder {
	b := &XsltBuilder{}
	for _, o := range opts {
		o(b)
	}
	return b
}

var defaultBuilder = NewBuilder()

// Build detecta el documento y genera su cadena original.
func Build(data []byte) (string, error) {
	loc, err := LocationFor(data)
	if err != nil {
		return "", err
	}
	return defaultBuilder.Build(data, loc)
}

// BuildTFD genera la cadena de un TimbreFiscalDigital 1.1 independiente.
func BuildTFD(data []byte) (string, error) {
	loc, err := LocationFor(data)
	if err != nil {
		return "", err
	}
	if loc != XsltTfd11 {
		return "", ErrUnsupportedDocument
	}
	return defaultBuilder.Build(data, loc)
}

// LocationFor identifica el XSLT por nombre expandido de la raíz y versión.
func LocationFor(data []byte) (string, error) {
	doc, err := xslt.Parse(data)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: XML: %w", err)
	}
	name := doc.RootName()
	v := doc.RootAttr("Version")
	if v == "" {
		v = doc.RootAttr("version")
	}
	switch {
	case name == (xml.Name{Space: "http://www.sat.gob.mx/cfd/4", Local: "Comprobante"}) && v == "4.0":
		return XsltCfdi40, nil
	case name == (xml.Name{Space: "http://www.sat.gob.mx/cfd/3", Local: "Comprobante"}) && v == "3.3":
		return XsltCfdi33, nil
	case name == (xml.Name{Space: "http://www.sat.gob.mx/TimbreFiscalDigital", Local: "TimbreFiscalDigital"}) && v == "1.1":
		return XsltTfd11, nil
	case name == (xml.Name{Space: "http://www.sat.gob.mx/esquemas/retencionpago/2", Local: "Retenciones"}) && v == "2.0":
		return XsltRetenciones20, nil
	}
	return "", fmt.Errorf("%w: {%s}%s versión %q", ErrUnsupportedDocument, name.Space, name.Local, v)
}

// Build transforma XML con el XSLT indicado.
func (b *XsltBuilder) Build(data []byte, loc string) (string, error) {
	doc, err := xslt.Parse(data)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: XML: %w", err)
	}
	s, err := b.compile(loc)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: XSLT %s: %w", loc, err)
	}
	out, err := s.Transform(doc)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: transformar: %w", err)
	}
	return out, nil
}

func (b *XsltBuilder) compile(loc string) (*xslt.Stylesheet, error) {
	k, err := xslt.Key(loc)
	if err != nil {
		return nil, err
	}
	if b.fsys != nil {
		return b.cached("fs", k, func() (*xslt.Stylesheet, error) {
			return xslt.Compile(loc, func(l string) ([]byte, error) {
				lk, err := xslt.Key(l)
				if err != nil {
					return nil, err
				}
				return fs.ReadFile(b.fsys, lk)
			})
		})
	}
	store := b.store
	if store == nil {
		store = satresources.Default()
	}
	ctx := context.Background()
	for attempt := 0; ; attempt++ {
		// Path descarga la hoja de estilo y sus inclusiones si faltan.
		if _, err := store.Path(ctx, loc); err != nil {
			return nil, err
		}
		version := store.Version()
		s, err := b.cached(version, k, func() (*xslt.Stylesheet, error) {
			return xslt.Compile(loc, func(l string) ([]byte, error) { return store.ReadFile(ctx, l) })
		})
		if err != nil || store.Version() == version || attempt > 0 {
			return s, err
		}
		// se activó otra versión mientras se compilaba: compilar con la nueva
	}
}

func (b *XsltBuilder) cached(version, key string, compile func() (*xslt.Stylesheet, error)) (*xslt.Stylesheet, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.version != version || b.cache == nil {
		b.version = version
		b.cache = map[string]*xslt.Stylesheet{}
	}
	if s := b.cache[key]; s != nil {
		return s, nil
	}
	s, err := compile()
	if err != nil {
		return nil, err
	}
	b.cache[key] = s
	return s, nil
}
