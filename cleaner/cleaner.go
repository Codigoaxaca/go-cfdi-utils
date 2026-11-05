// Package cleaner limpia CFDI recibidos que, después de ser sellados y
// timbrados, fueron alterados con información que no forma parte de la
// cadena original (por ejemplo addendas con espacios de nombres sin XSD) y
// que provocarían errores al validarlos.
//
// La limpieza no altera la cadena original ni el sello:
//
//	limpio, err := cleaner.Clean(xmlSucio)
package cleaner

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

const xsiNamespace = "http://www.w3.org/2001/XMLSchema-instance"

// Clean realiza todos los pasos de limpieza y devuelve el XML limpio:
//
//  1. Cambia la definición incorrecta xmlns:schemaLocation por xsi:schemaLocation.
//  2. Remueve la definición de CFDI 3 sin prefijo si existe la definición con prefijo cfdi.
//  3. Remueve el nodo cfdi:Addenda.
//  4. Remueve de xsi:schemaLocation los espacios de nombres sin una ubicación que termine en .xsd.
//  5. Remueve los nodos que no pertenecen a espacios de nombres del SAT.
//  6. Remueve de xsi:schemaLocation los pares que no pertenecen al SAT.
//  7. Remueve las declaraciones de espacios de nombres que no son del SAT y ya no se usan.
//  8. Colapsa los nodos cfdi:Complemento en uno solo, conservando el orden.
//  9. Corrige las ubicaciones de XSD conocidas del SAT escritas con otras mayúsculas.
func Clean(xml []byte) ([]byte, error) {
	c, err := New(xml)
	if err != nil {
		return nil, err
	}
	c.CleanAll()
	return []byte(c.Root().XML()), nil
}

// Cleaner permite ejecutar los pasos de limpieza de forma individual.
type Cleaner struct {
	root *nodes.Node
}

// New realiza las limpiezas previas a la carga (pasos 1 y 2) e interpreta el XML.
func New(xml []byte) (*Cleaner, error) {
	xml = BeforeLoad(xml)
	root, err := nodes.ParseXML(xml)
	if err != nil {
		return nil, err
	}
	return &Cleaner{root: root}, nil
}

// Root devuelve el nodo raíz.
func (c *Cleaner) Root() *nodes.Node { return c.root }

// CleanAll ejecuta los pasos 3 al 9.
func (c *Cleaner) CleanAll() {
	c.RemoveAddenda()
	c.RemoveIncompleteSchemaLocations()
	c.RemoveNonSatNSNodes()
	c.RemoveNonSatNSSchemaLocations()
	c.RemoveUnusedNamespaces()
	c.CollapseComprobanteComplemento()
	c.FixKnownSchemaLocationsXsdUrls()
}

var (
	xmlnsSchemaLocationRe = regexp.MustCompile(`\sxmlns:schemaLocation\s*=`)
	cfdi3DefaultNsRe      = regexp.MustCompile(`\sxmlns\s*=\s*["']http://www\.sat\.gob\.mx/cfd/3["']`)
)

// BeforeLoad realiza las limpiezas sobre el texto antes de interpretarlo como XML.
func BeforeLoad(xml []byte) []byte {
	if xmlnsSchemaLocationRe.Match(xml) && !bytes.Contains(xml, []byte("xsi:schemaLocation")) {
		xml = xmlnsSchemaLocationRe.ReplaceAll(xml, []byte(" xsi:schemaLocation="))
	}
	if bytes.Contains(xml, []byte(`xmlns:cfdi="http://www.sat.gob.mx/cfd/3"`)) {
		xml = cfdi3DefaultNsRe.ReplaceAll(xml, nil)
	}
	return xml
}

// IsSatNamespace informa si el espacio de nombres pertenece al SAT o es
// necesario para el CFDI (XMLSchema-instance).
func IsSatNamespace(ns string) bool {
	return strings.HasPrefix(ns, "http://www.sat.gob.mx/") || ns == xsiNamespace
}

// RemoveAddenda elimina los nodos Addenda hijos del comprobante.
func (c *Cleaner) RemoveAddenda() {
	for _, child := range c.root.Children() {
		if child.LocalName() == "Addenda" && c.namespaceOf(c.root, child) != "" &&
			strings.HasPrefix(c.namespaceOf(c.root, child), "http://www.sat.gob.mx/cfd/") {
			c.root.RemoveChild(child)
		}
	}
}

// RemoveIncompleteSchemaLocations elimina de cada xsi:schemaLocation los
// espacios de nombres que no tienen a continuación una ubicación .xsd.
func (c *Cleaner) RemoveIncompleteSchemaLocations() {
	c.root.Walk(func(n *nodes.Node) bool {
		value, ok := n.LookupAttr("xsi:schemaLocation")
		if !ok {
			return true
		}
		parts := strings.Fields(value)
		var keep []string
		for i := 0; i < len(parts); {
			if i+1 < len(parts) && strings.HasSuffix(strings.ToLower(parts[i+1]), ".xsd") {
				keep = append(keep, parts[i], parts[i+1])
				i += 2
				continue
			}
			i++
		}
		setOrRemove(n, "xsi:schemaLocation", strings.Join(keep, " "))
		return true
	})
}

// RemoveNonSatNSNodes elimina los nodos cuyo espacio de nombres no es del SAT.
func (c *Cleaner) RemoveNonSatNSNodes() {
	var visit func(n *nodes.Node, scope map[string]string)
	visit = func(n *nodes.Node, scope map[string]string) {
		scope = withDeclarations(scope, n)
		for _, child := range n.Children() {
			childScope := withDeclarations(scope, child)
			if ns := childScope[child.Prefix()]; ns != "" && !IsSatNamespace(ns) {
				n.RemoveChild(child)
				continue
			}
			visit(child, scope)
		}
	}
	visit(c.root, map[string]string{})
}

// RemoveNonSatNSSchemaLocations elimina de xsi:schemaLocation los pares que no son del SAT.
func (c *Cleaner) RemoveNonSatNSSchemaLocations() {
	c.root.Walk(func(n *nodes.Node) bool {
		value, ok := n.LookupAttr("xsi:schemaLocation")
		if !ok {
			return true
		}
		var keep []nodes.SchemaLocation
		for _, l := range nodes.ParseSchemaLocations(value) {
			if IsSatNamespace(l.Namespace) {
				keep = append(keep, l)
			}
		}
		setOrRemove(n, "xsi:schemaLocation", nodes.FormatSchemaLocations(keep))
		return true
	})
}

// RemoveUnusedNamespaces elimina las declaraciones xmlns:prefijo que no son
// del SAT y cuyo prefijo ya no se usa en ningún nodo o atributo.
func (c *Cleaner) RemoveUnusedNamespaces() {
	used := map[string]bool{}
	c.root.Walk(func(n *nodes.Node) bool {
		used[n.Prefix()] = true
		for _, a := range n.Attributes() {
			if i := strings.IndexByte(a.Name, ':'); i > 0 && a.Name[:i] != "xmlns" {
				used[a.Name[:i]] = true
			}
		}
		return true
	})
	c.root.Walk(func(n *nodes.Node) bool {
		for _, a := range n.Attributes() {
			prefix, isDecl := strings.CutPrefix(a.Name, "xmlns:")
			if !isDecl || used[prefix] || IsSatNamespace(a.Value) {
				continue
			}
			n.RemoveAttr(a.Name)
		}
		if v, ok := n.LookupAttr("xmlns"); ok && !IsSatNamespace(v) && !used[""] {
			n.RemoveAttr("xmlns")
		}
		return true
	})
}

// CollapseComprobanteComplemento junta los hijos de todos los nodos
// Complemento del comprobante en el primero, conservando el orden.
func (c *Cleaner) CollapseComprobanteComplemento() {
	var first *nodes.Node
	for _, child := range c.root.Children() {
		if child.LocalName() != "Complemento" || child.Prefix() != c.root.Prefix() {
			continue
		}
		if first == nil {
			first = child
			continue
		}
		c.root.RemoveChild(child)
		for _, grandChild := range child.Children() {
			first.AddChild(grandChild)
		}
	}
}

var knownXsdUrls = map[string]string{
	"http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd":                                   "http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd",
	"http://www.sat.gob.mx/sitio_internet/cfd/3/cfdv33.xsd":                                   "http://www.sat.gob.mx/sitio_internet/cfd/3/cfdv33.xsd",
	"http://www.sat.gob.mx/sitio_internet/cfd/3/cfdv32.xsd":                                   "http://www.sat.gob.mx/sitio_internet/cfd/3/cfdv32.xsd",
	"http://www.sat.gob.mx/sitio_internet/cfd/3/cfdv3.xsd":                                    "http://www.sat.gob.mx/sitio_internet/cfd/3/cfdv3.xsd",
	"http://www.sat.gob.mx/sitio_internet/cfd/timbrefiscaldigital/timbrefiscaldigitalv11.xsd": "http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigitalv11.xsd",
	"http://www.sat.gob.mx/sitio_internet/cfd/timbrefiscaldigital/timbrefiscaldigital.xsd":    "http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigital.xsd",
	"http://www.sat.gob.mx/sitio_internet/timbrefiscaldigital/timbrefiscaldigital.xsd":        "http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigital.xsd",
}

// FixKnownSchemaLocationsXsdUrls corrige las ubicaciones de los XSD de CFDI
// y TimbreFiscalDigital escritas con mayúsculas/minúsculas incorrectas.
func (c *Cleaner) FixKnownSchemaLocationsXsdUrls() {
	c.root.Walk(func(n *nodes.Node) bool {
		value, ok := n.LookupAttr("xsi:schemaLocation")
		if !ok {
			return true
		}
		locations := nodes.ParseSchemaLocations(value)
		for i, l := range locations {
			if fixed, ok := knownXsdUrls[strings.ToLower(l.Location)]; ok {
				locations[i].Location = fixed
			}
		}
		n.SetAttr("xsi:schemaLocation", nodes.FormatSchemaLocations(locations))
		return true
	})
}

func (c *Cleaner) namespaceOf(parent, child *nodes.Node) string {
	scope := withDeclarations(withDeclarations(map[string]string{}, parent), child)
	return scope[child.Prefix()]
}

func withDeclarations(scope map[string]string, n *nodes.Node) map[string]string {
	var out map[string]string
	for _, a := range n.Attributes() {
		var prefix string
		switch {
		case a.Name == "xmlns":
			prefix = ""
		case strings.HasPrefix(a.Name, "xmlns:"):
			prefix = a.Name[len("xmlns:"):]
		default:
			continue
		}
		if out == nil {
			out = make(map[string]string, len(scope)+1)
			for k, v := range scope {
				out[k] = v
			}
		}
		out[prefix] = a.Value
	}
	if out == nil {
		return scope
	}
	return out
}

func setOrRemove(n *nodes.Node, name, value string) {
	if value == "" {
		n.RemoveAttr(name)
		return
	}
	n.SetAttr(name, value)
}
