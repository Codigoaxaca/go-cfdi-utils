package nodes

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// ParseXML convierte un contenido XML en una estructura de nodos.
//
// Los nombres de nodos y atributos se conservan tal como aparecen en el
// documento (con su prefijo). Se ignoran comentarios e instrucciones de
// procesamiento. El texto de un elemento solo se conserva cuando no es
// únicamente espacio en blanco.
func ParseXML(data []byte) (*Node, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // BOM UTF-8
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	dec.CharsetReader = charsetReader

	var root *Node
	var stack []*Node
	var texts []*strings.Builder
	for {
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("nodes: XML inválido: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &Node{name: qualifiedName(t.Name), attrValues: map[string]string{}}
			for _, a := range t.Attr {
				n.SetAttr(qualifiedName(a.Name), a.Value)
			}
			if len(stack) == 0 {
				if root != nil {
					return nil, errors.New("nodes: XML inválido: existe más de un elemento raíz")
				}
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)
			texts = append(texts, &strings.Builder{})
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("nodes: XML inválido: cierre inesperado de %s", qualifiedName(t.Name))
			}
			current := stack[len(stack)-1]
			if current.name != qualifiedName(t.Name) {
				return nil, fmt.Errorf("nodes: XML inválido: se esperaba el cierre de %s y se encontró %s", current.name, qualifiedName(t.Name))
			}
			text := texts[len(texts)-1].String()
			if strings.TrimSpace(text) != "" {
				current.value = text
			}
			stack = stack[:len(stack)-1]
			texts = texts[:len(texts)-1]
		case xml.CharData:
			if len(texts) > 0 {
				texts[len(texts)-1].Write(t)
			}
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("nodes: XML inválido: el elemento %s no fue cerrado", stack[len(stack)-1].name)
	}
	if root == nil {
		return nil, errors.New("nodes: XML inválido: no contiene un elemento raíz")
	}
	return root, nil
}

// ParseXMLString es igual que ParseXML pero recibe una cadena.
func ParseXMLString(s string) (*Node, error) { return ParseXML([]byte(s)) }

// ParseXMLFile lee un archivo y lo convierte en una estructura de nodos.
func ParseXMLFile(path string) (*Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nodes: %w", err)
	}
	return ParseXML(data)
}

// MarshalOptions define cómo se genera el XML.
type MarshalOptions struct {
	// Declaration incluye la declaración <?xml version="1.0" encoding="UTF-8"?>.
	Declaration bool
	// Indent es la cadena usada para sangrar cada nivel, vacía para no formatear.
	Indent string
}

// DefaultMarshalOptions son las opciones usadas por XML(): con declaración y
// sangría de dos espacios.
var DefaultMarshalOptions = MarshalOptions{Declaration: true, Indent: "  "}

// XML genera el contenido XML del nodo con declaración y formato.
func (n *Node) XML() string {
	return n.MarshalWith(DefaultMarshalOptions)
}

// String genera el XML del nodo sin declaración ni formato.
func (n *Node) String() string {
	return n.MarshalWith(MarshalOptions{})
}

// MarshalWith genera el XML del nodo con las opciones indicadas.
func (n *Node) MarshalWith(opts MarshalOptions) string {
	var b strings.Builder
	if opts.Declaration {
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		b.WriteByte('\n')
	}
	writeNode(&b, n, opts.Indent, 0)
	if opts.Declaration || opts.Indent != "" {
		b.WriteByte('\n')
	}
	return b.String()
}

// WriteFile guarda el XML del nodo (con declaración y formato) en un archivo.
func (n *Node) WriteFile(path string) error {
	if err := os.WriteFile(path, []byte(n.XML()), 0o644); err != nil {
		return fmt.Errorf("nodes: %w", err)
	}
	return nil
}

func writeNode(b *strings.Builder, n *Node, indent string, level int) {
	if indent != "" && level > 0 {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(indent, level))
	}
	b.WriteByte('<')
	b.WriteString(n.name)
	for _, name := range n.attrNames {
		b.WriteByte(' ')
		b.WriteString(name)
		b.WriteString(`="`)
		escapeAttr(b, n.attrValues[name])
		b.WriteByte('"')
	}
	if len(n.children) == 0 && n.value == "" {
		b.WriteString("/>")
		return
	}
	b.WriteByte('>')
	escapeText(b, n.value)
	for _, c := range n.children {
		writeNode(b, c, indent, level+1)
	}
	if indent != "" && len(n.children) > 0 {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(indent, level))
	}
	b.WriteString("</")
	b.WriteString(n.name)
	b.WriteByte('>')
}

func escapeAttr(b *strings.Builder, s string) {
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\t':
			b.WriteString("&#9;")
		case '\n':
			b.WriteString("&#10;")
		case '\r':
			b.WriteString("&#13;")
		default:
			b.WriteRune(r)
		}
	}
}

func escapeText(b *strings.Builder, s string) {
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '\r':
			b.WriteString("&#13;")
		default:
			b.WriteRune(r)
		}
	}
}

func qualifiedName(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

// charsetReader permite leer documentos declarados como ISO-8859-1 o
// windows-1252 (se trata como latin1) además de UTF-8.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "utf-8", "utf8", "":
		return input, nil
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1", "windows-1252", "cp1252":
		data, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 0, len(data)*2)
		for _, c := range data {
			buf = utf8.AppendRune(buf, rune(c))
		}
		return bytes.NewReader(buf), nil
	}
	return nil, fmt.Errorf("codificación no soportada: %s", charset)
}
