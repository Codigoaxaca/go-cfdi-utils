// Package cadenaorigen genera cadenas originales del SAT sin herramientas externas.
// Interpreta únicamente el subconjunto XSLT utilizado por los recursos incluidos.
package cadenaorigen

import (
	"bytes"
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	XsltCfdi40        = "http://www.sat.gob.mx/sitio_internet/cfd/4/cadenaoriginal_4_0/cadenaoriginal_4_0.xslt"
	XsltCfdi33        = "http://www.sat.gob.mx/sitio_internet/cfd/3/cadenaoriginal_3_3/cadenaoriginal_3_3.xslt"
	XsltTfd11         = "http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/cadenaoriginal_TFD_1_1.xslt"
	XsltRetenciones20 = "http://www.sat.gob.mx/esquemas/retencionpago/2/retenciones.xslt"
	xslNS             = "http://www.w3.org/1999/XSL/Transform"
)

// ErrUnsupportedDocument indica un documento o versión no reconocido.
var ErrUnsupportedDocument = errors.New("cadenaorigen: documento no soportado")

// Builder transforma XML mediante una ubicación XSLT.
type Builder interface {
	Build(xml []byte, xsltLocation string) (string, error)
}

//go:embed resources
var resources embed.FS

// XsltBuilder mantiene una caché segura para uso concurrente. Su valor cero es utilizable.
// No debe copiarse después de su primer uso.
type XsltBuilder struct {
	mu      sync.Mutex
	cache   map[string]*stylesheet
	sources map[string][]byte
	fsys    fs.FS
	client  *http.Client
}

// Option configura un constructor antes de su uso.
type Option func(*XsltBuilder)

// WithFS consulta archivos host/ruta antes de los recursos incluidos.
func WithFS(fsys fs.FS) Option { return func(b *XsltBuilder) { b.fsys = fsys } }

// WithHTTPClient permite descargar recursos ausentes y guardarlos en memoria.
func WithHTTPClient(c *http.Client) Option { return func(b *XsltBuilder) { b.client = c } }

// NewBuilder crea un constructor con recursos SAT incluidos.
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
	doc, err := parse(data)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: XML: %w", err)
	}
	n := doc.children[0]
	v := n.attr("Version")
	if v == "" {
		v = n.attr("version")
	}
	switch {
	case n.name == (xml.Name{Space: "http://www.sat.gob.mx/cfd/4", Local: "Comprobante"}) && v == "4.0":
		return XsltCfdi40, nil
	case n.name == (xml.Name{Space: "http://www.sat.gob.mx/cfd/3", Local: "Comprobante"}) && v == "3.3":
		return XsltCfdi33, nil
	case n.name == (xml.Name{Space: "http://www.sat.gob.mx/TimbreFiscalDigital", Local: "TimbreFiscalDigital"}) && v == "1.1":
		return XsltTfd11, nil
	case n.name == (xml.Name{Space: "http://www.sat.gob.mx/esquemas/retencionpago/2", Local: "Retenciones"}) && v == "2.0":
		return XsltRetenciones20, nil
	}
	return "", fmt.Errorf("%w: {%s}%s versión %q", ErrUnsupportedDocument, n.name.Space, n.name.Local, v)
}

type node struct {
	name     xml.Name
	attrs    []xml.Attr
	text     string
	isText   bool
	children []*node
	parent   *node
	ns       map[string]string
}

func (n *node) attr(k string) string {
	for _, a := range n.attrs {
		if a.Name.Space == "" && a.Name.Local == k {
			return a.Value
		}
	}
	return ""
}
func parse(data []byte) (*node, error) {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	d := xml.NewDecoder(bytes.NewReader(data))
	doc := &node{}
	cur := doc
	roots := 0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := t.(type) {
		case xml.StartElement:
			if cur == doc {
				roots++
			}
			ns := map[string]string{"xml": "http://www.w3.org/XML/1998/namespace"}
			for k, v := range cur.ns {
				ns[k] = v
			}
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" {
					ns[a.Name.Local] = a.Value
				} else if a.Name.Local == "xmlns" && a.Name.Space == "" {
					ns[""] = a.Value
				}
			}
			n := &node{name: t.Name, attrs: t.Attr, parent: cur, ns: ns}
			cur.children = append(cur.children, n)
			cur = n
		case xml.EndElement:
			cur = cur.parent
		case xml.CharData:
			if cur == doc {
				if strings.TrimSpace(string(t)) != "" {
					return nil, errors.New("texto fuera de la raíz")
				}
				continue
			}
			s := string(t)
			if len(cur.children) > 0 && cur.children[len(cur.children)-1].isText {
				cur.children[len(cur.children)-1].text += s
			} else {
				cur.children = append(cur.children, &node{isText: true, text: s, parent: cur})
			}
		}
	}
	if roots != 1 || cur != doc {
		return nil, errors.New("se requiere exactamente una raíz XML")
	}
	return doc, nil
}

type stylesheet struct {
	matches map[xml.Name]*node
	named   map[string]*node
	root    *node
}

func key(location string) (string, error) {
	u, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("URL XSLT inválida: %q", location)
	}
	p := u.Host + u.Path
	if !fs.ValidPath(p) {
		return "", fmt.Errorf("ruta XSLT inválida: %q", p)
	}
	return p, nil
}
func (b *XsltBuilder) source(loc string) ([]byte, error) {
	k, err := key(loc)
	if err != nil {
		return nil, err
	}
	if b.sources == nil {
		b.sources = map[string][]byte{}
	}
	if data, ok := b.sources[k]; ok {
		return data, nil
	}
	if b.fsys != nil {
		data, e := fs.ReadFile(b.fsys, k)
		if e == nil {
			b.sources[k] = data
			return data, nil
		}
		if !errors.Is(e, fs.ErrNotExist) {
			return nil, e
		}
	}
	data, err := resources.ReadFile("resources/" + k)
	if err == nil {
		b.sources[k] = data
		return data, nil
	}
	if b.client == nil {
		return nil, fmt.Errorf("recurso XSLT %s: %w", loc, err)
	}
	resp, err := b.client.Get(loc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", loc, resp.Status)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err == nil {
		b.sources[k] = data
	}
	return data, err
}
func expanded(q string, ns map[string]string) (xml.Name, error) {
	p, l, ok := strings.Cut(q, ":")
	if !ok {
		return xml.Name{Local: q}, nil
	}
	uri, ok := ns[p]
	if !ok {
		return xml.Name{}, fmt.Errorf("prefijo desconocido %q", p)
	}
	return xml.Name{Space: uri, Local: l}, nil
}
func (b *XsltBuilder) compile(loc string) (*stylesheet, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	k, err := key(loc)
	if err != nil {
		return nil, err
	}
	if s := b.cache[k]; s != nil {
		return s, nil
	}
	s := &stylesheet{matches: map[xml.Name]*node{}, named: map[string]*node{}}
	seen := map[string]bool{}
	var load func(string) error
	load = func(loc string) error {
		k, err := key(loc)
		if err != nil {
			return err
		}
		if seen[k] {
			return nil
		}
		seen[k] = true
		data, err := b.source(loc)
		if err != nil {
			return err
		}
		doc, err := parse(data)
		if err != nil {
			return err
		}
		root := doc.children[0]
		if root.name.Space != xslNS || (root.name.Local != "stylesheet" && root.name.Local != "transform") {
			return fmt.Errorf("%s no es XSLT", loc)
		}
		for _, n := range root.children {
			if n.isText {
				continue
			}
			switch n.name.Local {
			case "include":
				base, _ := url.Parse(loc)
				ref, e := url.Parse(n.attr("href"))
				if e != nil {
					return e
				}
				if e = load(base.ResolveReference(ref).String()); e != nil {
					return e
				}
			case "template":
				if name := n.attr("name"); name != "" {
					s.named[name] = n
				}
				if m := n.attr("match"); m == "/" {
					s.root = n
				} else if m != "" {
					q, e := expanded(m, n.ns)
					if e != nil {
						return e
					}
					s.matches[q] = n
				}
			case "output":
			default:
				return fmt.Errorf("instrucción de nivel superior no soportada: %s", n.name.Local)
			}
		}
		return nil
	}
	if err = load(loc); err != nil {
		return nil, err
	}
	if b.cache == nil {
		b.cache = map[string]*stylesheet{}
	}
	b.cache[k] = s
	return s, nil
}

// Build transforma XML con el XSLT indicado; no realiza solicitudes de red por defecto.
func (b *XsltBuilder) Build(data []byte, loc string) (string, error) {
	doc, err := parse(data)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: XML: %w", err)
	}
	s, err := b.compile(loc)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: XSLT %s: %w", loc, err)
	}
	r := runner{s: s}
	err = r.apply(doc, 0)
	if err != nil {
		return "", fmt.Errorf("cadenaorigen: transformar: %w", err)
	}
	return r.out.String(), nil
}

type value []*node
type env map[string]value

func stringValue(v value) string {
	if len(v) == 0 {
		return ""
	}
	var out strings.Builder
	var visit func(*node)
	visit = func(n *node) {
		if n.isText {
			out.WriteString(n.text)
		} else {
			for _, c := range n.children {
				visit(c)
			}
		}
	}
	visit(v[0])
	return out.String()
}
func selectNodes(expr string, ctx *node, ns map[string]string, vars env) (value, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "$") {
		return vars[expr[1:]], nil
	}
	if expr == "." {
		return value{ctx}, nil
	}
	cur := value{ctx}
	if strings.HasPrefix(expr, "/") {
		for ctx.parent != nil {
			ctx = ctx.parent
		}
		cur = value{ctx}
		expr = expr[1:]
	}
	expr = strings.TrimPrefix(expr, "./")
	desc := strings.HasPrefix(expr, "/")
	expr = strings.TrimPrefix(expr, "/")
	for i, part := range strings.Split(expr, "/") {
		if part == "" {
			return nil, fmt.Errorf("XPath no soportado: %q", expr)
		}
		var next value
		attr := strings.HasPrefix(part, "@")
		q, err := expanded(strings.TrimPrefix(part, "@"), ns)
		if err != nil {
			return nil, err
		}
		var visit func(*node)
		visit = func(n *node) {
			if attr {
				for _, a := range n.attrs {
					if a.Name == q {
						next = append(next, &node{isText: true, text: a.Value, parent: n})
					}
				}
				return
			}
			for _, c := range n.children {
				if !c.isText && (part == "*" || c.name == q) {
					next = append(next, c)
				}
				if desc && i == 0 && !c.isText {
					visit(c)
				}
			}
		}
		for _, n := range cur {
			visit(n)
		}
		cur = next
	}
	return cur, nil
}

type runner struct {
	s   *stylesheet
	out strings.Builder
}

func (r *runner) apply(n *node, depth int) error {
	if depth > 512 {
		return errors.New("profundidad de transformación excedida")
	}
	if n.isText {
		r.out.WriteString(n.text)
		return nil
	}
	t := r.s.matches[n.name]
	if n.parent == nil {
		t = r.s.root
	}
	if t != nil {
		return r.exec(t.children, n, env{}, depth+1)
	}
	for _, c := range n.children {
		if err := r.apply(c, depth+1); err != nil {
			return err
		}
	}
	return nil
}
func (r *runner) exec(nodes []*node, ctx *node, vars env, depth int) error {
	if depth > 512 {
		return errors.New("profundidad de transformación excedida")
	}
	for _, n := range nodes {
		if n.isText {
			if strings.Trim(n.text, " \r\n\t") != "" {
				r.out.WriteString(n.text)
			}
			continue
		}
		if n.name.Space != xslNS {
			return fmt.Errorf("elemento literal no soportado: %s", n.name.Local)
		}
		sel := func(expr string) (value, error) { return selectNodes(expr, ctx, n.ns, vars) }
		switch n.name.Local {
		case "param":
			if _, ok := vars[n.attr("name")]; !ok {
				v, err := sel(n.attr("select"))
				if n.attr("select") == "" {
					v = nil
					err = nil
				}
				if err != nil {
					return err
				}
				vars[n.attr("name")] = v
			}
		case "text":
			r.out.WriteString(stringValue(value{n}))
		case "apply-templates", "for-each":
			expr := n.attr("select")
			var v value
			var err error
			if expr == "" && n.name.Local == "apply-templates" {
				v = ctx.children
			} else {
				v, err = sel(expr)
			}
			if err != nil {
				return err
			}
			for _, item := range v {
				if n.name.Local == "apply-templates" {
					err = r.apply(item, depth+1)
				} else {
					err = r.exec(n.children, item, vars, depth+1)
				}
				if err != nil {
					return err
				}
			}
		case "call-template":
			t := r.s.named[n.attr("name")]
			if t == nil {
				return fmt.Errorf("plantilla desconocida: %s", n.attr("name"))
			}
			args := env{}
			for _, p := range n.children {
				if p.isText {
					continue
				}
				if p.name.Space != xslNS || p.name.Local != "with-param" {
					return errors.New("se esperaba with-param")
				}
				v, err := sel(p.attr("select"))
				if err != nil {
					return err
				}
				args[p.attr("name")] = v
			}
			if err := r.exec(t.children, ctx, args, depth+1); err != nil {
				return err
			}
		case "if":
			test := strings.TrimSpace(n.attr("test"))
			left, right, equal := strings.Cut(test, "=")
			v, err := sel(strings.TrimSpace(left))
			if err != nil {
				return err
			}
			ok := len(v) > 0
			if equal {
				right = strings.TrimSpace(right)
				if len(right) < 2 || (right[0] != '\'' && right[0] != '"') || right[len(right)-1] != right[0] {
					return fmt.Errorf("comparación no soportada: %s", test)
				}
				ok = false
				for _, item := range v {
					if stringValue(value{item}) == right[1:len(right)-1] {
						ok = true
					}
				}
			}
			if ok {
				if err := r.exec(n.children, ctx, vars, depth+1); err != nil {
					return err
				}
			}
		case "value-of":
			expr := n.attr("select")
			normalize := strings.HasPrefix(expr, "normalize-space(string(") && strings.HasSuffix(expr, "))")
			if normalize {
				expr = strings.TrimSuffix(strings.TrimPrefix(expr, "normalize-space(string("), "))")
			}
			v, err := sel(expr)
			if err != nil {
				return err
			}
			text := stringValue(v)
			if normalize {
				text = strings.Join(strings.FieldsFunc(text, func(c rune) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }), " ")
			}
			r.out.WriteString(text)
		default:
			return fmt.Errorf("instrucción XSLT no soportada: %s", n.name.Local)
		}
	}
	return nil
}
