// Package xslt interpreta el subconjunto de XSLT que usan las hojas de estilo
// del SAT para generar cadenas originales. No es un procesador XSLT general.
//
// Compile revisa de forma estática todas las plantillas: cualquier instrucción,
// atributo o expresión XPath fuera del subconjunto soportado produce un error
// al compilar, en lugar de generar en silencio una cadena incorrecta. Esto
// permite verificar una versión nueva de los XSLT del SAT antes de usarla.
package xslt

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"strings"
	"unicode"
)

// NS es el espacio de nombres de XSLT.
const NS = "http://www.w3.org/1999/XSL/Transform"

// Key devuelve la llave host/ruta de una URL http o https. Las URL http y
// https de un mismo recurso comparten la llave.
func Key(location string) (string, error) {
	u, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("URL inválida: %q", location)
	}
	p := strings.ToLower(u.Host) + u.Path
	if !fs.ValidPath(p) {
		return "", fmt.Errorf("ruta inválida: %q", p)
	}
	return p, nil
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

// Document es un documento XML interpretado.
type Document struct{ doc *node }

// RootName devuelve el nombre expandido del elemento raíz.
func (d *Document) RootName() xml.Name { return d.doc.children[0].name }

// RootAttr devuelve un atributo sin espacio de nombres del elemento raíz.
func (d *Document) RootAttr(name string) string { return d.doc.children[0].attr(name) }

// Parse interpreta un documento XML con exactamente un elemento raíz.
func Parse(data []byte) (*Document, error) {
	doc, err := parse(data)
	if err != nil {
		return nil, err
	}
	return &Document{doc: doc}, nil
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

// Loader obtiene el contenido de una hoja de estilo por su URL.
type Loader func(location string) ([]byte, error)

// Stylesheet es una hoja de estilo compilada. Es inmutable y se puede usar de
// forma concurrente.
type Stylesheet struct {
	matches map[xml.Name]*node
	named   map[string]*node
	root    *node
}

// Compile carga la hoja de estilo en location, junto con sus xsl:include,
// y verifica que solo use el subconjunto soportado.
func Compile(location string, load Loader) (*Stylesheet, error) {
	s := &Stylesheet{matches: map[xml.Name]*node{}, named: map[string]*node{}}
	seen := map[string]bool{}
	loading := map[string]bool{}
	var templates []*node
	var loadOne func(string) error
	loadOne = func(loc string) error {
		k, err := Key(loc)
		if err != nil {
			return err
		}
		if loading[k] {
			return fmt.Errorf("%s: inclusión circular", loc)
		}
		if seen[k] {
			return nil
		}
		seen[k] = true
		loading[k] = true
		defer delete(loading, k)
		data, err := load(loc)
		if err != nil {
			return err
		}
		doc, err := parse(data)
		if err != nil {
			return fmt.Errorf("%s: %w", loc, err)
		}
		root := doc.children[0]
		if root.name.Space != NS || (root.name.Local != "stylesheet" && root.name.Local != "transform") {
			return fmt.Errorf("%s no es XSLT", loc)
		}
		for _, n := range root.children {
			if n.isText {
				if strings.TrimSpace(n.text) != "" {
					return fmt.Errorf("%s: texto en el nivel superior", loc)
				}
				continue
			}
			if n.name.Space != NS {
				return fmt.Errorf("%s: elemento de nivel superior no soportado: %s", loc, n.name.Local)
			}
			switch n.name.Local {
			case "include":
				if err := checkAttrs(n, "href"); err != nil {
					return fmt.Errorf("%s: %w", loc, err)
				}
				base, _ := url.Parse(loc)
				ref, e := url.Parse(n.attr("href"))
				if e != nil {
					return e
				}
				if e = loadOne(base.ResolveReference(ref).String()); e != nil {
					return e
				}
			case "template":
				if err := checkAttrs(n, "match", "name"); err != nil {
					return fmt.Errorf("%s: %w", loc, err)
				}
				name, m := n.attr("name"), n.attr("match")
				if name == "" && m == "" {
					return fmt.Errorf("%s: plantilla sin match ni name", loc)
				}
				if name != "" {
					s.named[name] = n
				}
				if m == "/" {
					s.root = n
				} else if m != "" {
					if m == "*" || !isQName(m) {
						return fmt.Errorf("%s: patrón no soportado: %q", loc, m)
					}
					q, e := expanded(m, n.ns)
					if e != nil {
						return fmt.Errorf("%s: %w", loc, e)
					}
					s.matches[q] = n
				}
				templates = append(templates, n)
			case "output":
			default:
				return fmt.Errorf("%s: instrucción de nivel superior no soportada: %s", loc, n.name.Local)
			}
		}
		return nil
	}
	if err := loadOne(location); err != nil {
		return nil, err
	}
	for _, t := range templates {
		if err := s.check(t.children); err != nil {
			return nil, fmt.Errorf("plantilla %q: %w", templateLabel(t), err)
		}
	}
	return s, nil
}

func templateLabel(t *node) string {
	if m := t.attr("match"); m != "" {
		return m
	}
	return t.attr("name")
}

// Transform aplica la hoja de estilo al documento y devuelve el texto generado.
func (s *Stylesheet) Transform(doc *Document) (string, error) {
	r := runner{s: s}
	if err := r.apply(doc.doc, 0); err != nil {
		return "", err
	}
	return r.out.String(), nil
}

// isQName informa si s es un nombre (con prefijo opcional) o "*".
func isQName(s string) bool {
	if s == "*" {
		return true
	}
	if s == "" {
		return false
	}
	prefix, local, hasPrefix := strings.Cut(s, ":")
	if hasPrefix {
		return isNCName(prefix) && isNCName(local)
	}
	return isNCName(s)
}

func isNCName(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		switch {
		case unicode.IsLetter(c) || c == '_':
		case i > 0 && (unicode.IsDigit(c) || c == '-' || c == '.'):
		default:
			return false
		}
	}
	return true
}

func checkAttrs(n *node, allowed ...string) error {
	for _, a := range n.attrs {
		if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
			continue
		}
		ok := false
		for _, name := range allowed {
			if a.Name.Space == "" && a.Name.Local == name {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("atributo no soportado %q en xsl:%s", a.Name.Local, n.name.Local)
		}
	}
	return nil
}

// checkExpr valida una expresión XPath con la misma gramática que selectNodes.
func checkExpr(expr string, ns map[string]string) error {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "$") {
		if !isNCName(expr[1:]) {
			return fmt.Errorf("XPath no soportado: %q", expr)
		}
		return nil
	}
	if expr == "." {
		return nil
	}
	path := strings.TrimPrefix(expr, "/")
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return fmt.Errorf("XPath no soportado: %q", expr)
	}
	for _, part := range strings.Split(path, "/") {
		name := strings.TrimPrefix(part, "@")
		if !isQName(name) || name == "*" && part != name {
			return fmt.Errorf("XPath no soportado: %q", expr)
		}
		if _, err := expanded(name, ns); err != nil {
			return fmt.Errorf("XPath %q: %w", expr, err)
		}
	}
	return nil
}

func (s *Stylesheet) check(nodes []*node) error {
	for _, n := range nodes {
		if n.isText {
			continue
		}
		if n.name.Space != NS {
			return fmt.Errorf("elemento literal no soportado: %s", n.name.Local)
		}
		var err error
		switch n.name.Local {
		case "param":
			if err = checkAttrs(n, "name", "select"); err == nil && n.attr("select") != "" {
				err = checkExpr(n.attr("select"), n.ns)
			}
			if err == nil && hasContent(n) {
				err = errors.New("xsl:param con contenido no soportado")
			}
		case "text":
			if err = checkAttrs(n); err == nil && hasElements(n) {
				err = errors.New("xsl:text con elementos no soportado")
			}
		case "apply-templates":
			if err = checkAttrs(n, "select"); err == nil && n.attr("select") != "" {
				err = checkExpr(n.attr("select"), n.ns)
			}
			if err == nil && hasContent(n) {
				err = errors.New("xsl:apply-templates con contenido no soportado")
			}
		case "for-each":
			if err = checkAttrs(n, "select"); err == nil {
				err = checkExpr(n.attr("select"), n.ns)
			}
			if err == nil {
				err = s.check(n.children)
			}
		case "call-template":
			if err = checkAttrs(n, "name"); err != nil {
				break
			}
			if s.named[n.attr("name")] == nil {
				err = fmt.Errorf("plantilla desconocida: %s", n.attr("name"))
				break
			}
			for _, p := range n.children {
				if p.isText {
					if strings.TrimSpace(p.text) != "" {
						err = errors.New("xsl:call-template con texto no soportado")
						break
					}
					continue
				}
				if p.name.Space != NS || p.name.Local != "with-param" {
					err = errors.New("se esperaba with-param")
				} else if err = checkAttrs(p, "name", "select"); err == nil {
					err = checkExpr(p.attr("select"), p.ns)
				}
				if err == nil && hasContent(p) {
					err = errors.New("xsl:with-param con contenido no soportado")
				}
				if err != nil {
					break
				}
			}
		case "if":
			if err = checkAttrs(n, "test"); err != nil {
				break
			}
			test := strings.TrimSpace(n.attr("test"))
			left, right, equal := strings.Cut(test, "=")
			if err = checkExpr(left, n.ns); err != nil {
				break
			}
			if equal {
				right = strings.TrimSpace(right)
				if len(right) < 2 || (right[0] != '\'' && right[0] != '"') || right[len(right)-1] != right[0] || strings.ContainsRune(right[1:len(right)-1], rune(right[0])) {
					err = fmt.Errorf("comparación no soportada: %s", test)
					break
				}
			}
			err = s.check(n.children)
		case "value-of":
			if err = checkAttrs(n, "select"); err != nil {
				break
			}
			expr := n.attr("select")
			if strings.HasPrefix(expr, "normalize-space(string(") && strings.HasSuffix(expr, "))") {
				expr = strings.TrimSuffix(strings.TrimPrefix(expr, "normalize-space(string("), "))")
			}
			if err = checkExpr(expr, n.ns); err == nil && hasContent(n) {
				err = errors.New("xsl:value-of con contenido no soportado")
			}
		default:
			err = fmt.Errorf("instrucción XSLT no soportada: %s", n.name.Local)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func hasElements(n *node) bool {
	for _, c := range n.children {
		if !c.isText {
			return true
		}
	}
	return false
}

// hasContent informa si el nodo tiene elementos o texto que no sea espacio:
// contenido que el intérprete ignoraría.
func hasContent(n *node) bool {
	for _, c := range n.children {
		if !c.isText || strings.TrimSpace(c.text) != "" {
			return true
		}
	}
	return false
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
	s   *Stylesheet
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
		if n.name.Space != NS {
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
				if p.name.Space != NS || p.name.Local != "with-param" {
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
