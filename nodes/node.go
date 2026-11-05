// Package nodes contiene una estructura de datos sencilla para representar
// documentos XML como árboles de nodos con atributos ordenados, hijos y
// contenido textual simple.
//
// Es la base sobre la que se construyen los elementos de CFDI (paquete cfdi40)
// y los complementos. Un nodo solo almacena su nombre (con prefijo, por
// ejemplo "cfdi:Comprobante"), sus atributos, sus hijos y un valor textual.
//
//	n := nodes.New("cfdi:Emisor", nodes.Attrs{"Rfc": "EKU9003173C9"})
//	n.SetAttr("RegimenFiscal", "601")
//	fmt.Println(n.Attr("Rfc")) // EKU9003173C9
package nodes

import (
	"slices"
	"sort"
	"strings"
)

// Attrs es un mapa de atributos nombre => valor.
//
// Al aplicar un Attrs a un nodo, los atributos se escriben en el orden
// preferido del nodo (ver SetAttributesOrder) y el resto en orden alfabético,
// de forma que la salida XML siempre es determinista.
type Attrs = map[string]string

// Attribute es un atributo de un nodo.
type Attribute struct {
	Name  string
	Value string
}

// Node es un nodo XML con nombre, atributos ordenados, hijos y valor textual.
//
// El valor cero no es útil, utiliza New para crear nodos.
type Node struct {
	name       string
	attrNames  []string
	attrValues map[string]string
	children   []*Node
	value      string
	childOrder []string
	attrOrder  []string
}

// New crea un nodo con el nombre, atributos e hijos dados.
func New(name string, attrs Attrs, children ...*Node) *Node {
	n := &Node{name: strings.TrimSpace(name), attrValues: map[string]string{}}
	n.SetAttrs(attrs)
	for _, child := range children {
		n.AddChild(child)
	}
	return n
}

// Name devuelve el nombre calificado del nodo, por ejemplo "cfdi:Comprobante".
func (n *Node) Name() string { return n.name }

// LocalName devuelve el nombre sin prefijo, por ejemplo "Comprobante".
func (n *Node) LocalName() string {
	if i := strings.IndexByte(n.name, ':'); i >= 0 {
		return n.name[i+1:]
	}
	return n.name
}

// Prefix devuelve el prefijo del nombre del nodo, por ejemplo "cfdi".
func (n *Node) Prefix() string {
	if i := strings.IndexByte(n.name, ':'); i >= 0 {
		return n.name[:i]
	}
	return ""
}

// Attr devuelve el valor de un atributo; si no existe devuelve una cadena vacía.
func (n *Node) Attr(name string) string { return n.attrValues[name] }

// LookupAttr devuelve el valor de un atributo y si existe.
func (n *Node) LookupAttr(name string) (string, bool) {
	v, ok := n.attrValues[name]
	return v, ok
}

// HasAttr informa si el atributo existe (aunque su valor sea vacío).
func (n *Node) HasAttr(name string) bool {
	_, ok := n.attrValues[name]
	return ok
}

// SetAttr establece el valor de un atributo y devuelve el mismo nodo.
//
// Si el atributo es nuevo se inserta respetando el orden preferido del nodo.
func (n *Node) SetAttr(name, value string) *Node {
	name = strings.TrimSpace(name)
	if name == "" {
		return n
	}
	if n.attrValues == nil {
		n.attrValues = map[string]string{}
	}
	if _, exists := n.attrValues[name]; !exists {
		n.insertAttrName(name)
	}
	n.attrValues[name] = value
	return n
}

// SetAttrs establece varios atributos y devuelve el mismo nodo.
func (n *Node) SetAttrs(attrs Attrs) *Node {
	for _, name := range sortedAttrNames(attrs, n.attrOrder) {
		n.SetAttr(name, attrs[name])
	}
	return n
}

// RemoveAttr elimina un atributo.
func (n *Node) RemoveAttr(name string) *Node {
	if _, ok := n.attrValues[name]; !ok {
		return n
	}
	delete(n.attrValues, name)
	n.attrNames = slices.DeleteFunc(n.attrNames, func(s string) bool { return s == name })
	return n
}

// ClearAttrs elimina todos los atributos.
func (n *Node) ClearAttrs() *Node {
	n.attrNames = nil
	n.attrValues = map[string]string{}
	return n
}

// Attributes devuelve una copia de los atributos en orden.
func (n *Node) Attributes() []Attribute {
	out := make([]Attribute, 0, len(n.attrNames))
	for _, name := range n.attrNames {
		out = append(out, Attribute{Name: name, Value: n.attrValues[name]})
	}
	return out
}

// AttrsMap devuelve una copia de los atributos como mapa.
func (n *Node) AttrsMap() Attrs {
	out := make(Attrs, len(n.attrValues))
	for k, v := range n.attrValues {
		out[k] = v
	}
	return out
}

// Value devuelve el contenido textual simple del nodo.
func (n *Node) Value() string { return n.value }

// SetValue establece el contenido textual simple del nodo.
func (n *Node) SetValue(value string) *Node {
	n.value = value
	return n
}

// SetAttributesOrder establece el orden preferido de los atributos.
// Los atributos existentes se reordenan.
func (n *Node) SetAttributesOrder(names ...string) *Node {
	n.attrOrder = slices.Clone(names)
	current := n.attrNames
	n.attrNames = nil
	for _, name := range current {
		n.insertAttrName(name)
	}
	return n
}

// AttributesOrder devuelve el orden preferido de los atributos.
func (n *Node) AttributesOrder() []string { return slices.Clone(n.attrOrder) }

// SetChildrenOrder establece el orden en el que deben aparecer los hijos
// según su nombre. Los hijos existentes se reordenan (de forma estable).
//
// El SAT exige en sus XSD un orden específico de los nodos, por ejemplo
// Emisor antes de Receptor. Los elementos de cfdi40 ya conocen su orden.
func (n *Node) SetChildrenOrder(names ...string) *Node {
	n.childOrder = slices.Clone(names)
	if len(n.childOrder) > 0 {
		sort.SliceStable(n.children, func(i, j int) bool {
			return n.orderOf(n.children[i].name) < n.orderOf(n.children[j].name)
		})
	}
	return n
}

// ChildrenOrder devuelve el orden de hijos establecido.
func (n *Node) ChildrenOrder() []string { return slices.Clone(n.childOrder) }

// Children devuelve una copia de la lista de hijos.
func (n *Node) Children() []*Node { return slices.Clone(n.children) }

// Len devuelve la cantidad de hijos.
func (n *Node) Len() int { return len(n.children) }

// Child devuelve el hijo en la posición index o nil si no existe.
func (n *Node) Child(index int) *Node {
	if index < 0 || index >= len(n.children) {
		return nil
	}
	return n.children[index]
}

// AddChild agrega un hijo respetando el orden de hijos establecido
// y devuelve el hijo agregado. Si el hijo ya existe no se agrega dos veces.
func (n *Node) AddChild(child *Node) *Node {
	if child == nil || n.IndexOf(child) >= 0 {
		return child
	}
	if len(n.childOrder) == 0 {
		n.children = append(n.children, child)
		return child
	}
	pos := n.orderOf(child.name)
	idx := len(n.children)
	for i, c := range n.children {
		if n.orderOf(c.name) > pos {
			idx = i
			break
		}
	}
	n.children = slices.Insert(n.children, idx, child)
	return child
}

// RemoveChild elimina un hijo (por identidad) e informa si fue eliminado.
func (n *Node) RemoveChild(child *Node) bool {
	idx := n.IndexOf(child)
	if idx < 0 {
		return false
	}
	n.children = slices.Delete(n.children, idx, idx+1)
	return true
}

// ClearChildren elimina todos los hijos.
func (n *Node) ClearChildren() *Node {
	n.children = nil
	return n
}

// Clear elimina atributos e hijos.
func (n *Node) Clear() *Node {
	n.ClearAttrs()
	n.ClearChildren()
	return n
}

// IndexOf devuelve la posición de un hijo (por identidad) o -1.
func (n *Node) IndexOf(child *Node) int {
	for i, c := range n.children {
		if c == child {
			return i
		}
	}
	return -1
}

// FirstChild devuelve el primer hijo con el nombre dado o nil.
func (n *Node) FirstChild(name string) *Node {
	for _, c := range n.children {
		if c.name == name {
			return c
		}
	}
	return nil
}

// ChildrenNamed devuelve los hijos con el nombre dado.
func (n *Node) ChildrenNamed(name string) []*Node {
	var out []*Node
	for _, c := range n.children {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}

// SearchNode busca el primer nodo que coincida con la ruta de nombres de hijos.
//
//	tfd := comprobante.SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital")
func (n *Node) SearchNode(path ...string) *Node {
	current := n
	for _, name := range path {
		current = current.FirstChild(name)
		if current == nil {
			return nil
		}
	}
	return current
}

// SearchNodes devuelve todos los nodos que coinciden con la ruta. Los padres
// intermedios se buscan como el primer hijo con el nombre dado y el último
// elemento de la ruta devuelve todos los hijos con ese nombre.
//
//	conceptos := comprobante.SearchNodes("cfdi:Conceptos", "cfdi:Concepto")
func (n *Node) SearchNodes(path ...string) []*Node {
	if len(path) == 0 {
		return nil
	}
	parent := n.SearchNode(path[:len(path)-1]...)
	if parent == nil {
		return nil
	}
	return parent.ChildrenNamed(path[len(path)-1])
}

// SearchAttribute devuelve el valor de un atributo según una ruta. El último
// elemento de la ruta es el nombre del atributo.
//
//	rfc := comprobante.SearchAttribute("cfdi:Emisor", "Rfc")
func (n *Node) SearchAttribute(path ...string) string {
	if len(path) == 0 {
		return ""
	}
	node := n.SearchNode(path[:len(path)-1]...)
	if node == nil {
		return ""
	}
	return node.Attr(path[len(path)-1])
}

// Walk recorre el nodo y sus descendientes en profundidad. Si fn devuelve
// false no se recorren los hijos del nodo actual.
func (n *Node) Walk(fn func(node *Node) bool) {
	if !fn(n) {
		return
	}
	for _, c := range n.children {
		c.Walk(fn)
	}
}

// Clone crea una copia profunda del nodo.
func (n *Node) Clone() *Node {
	c := &Node{
		name:       n.name,
		attrNames:  slices.Clone(n.attrNames),
		attrValues: make(map[string]string, len(n.attrValues)),
		value:      n.value,
		childOrder: slices.Clone(n.childOrder),
		attrOrder:  slices.Clone(n.attrOrder),
	}
	for k, v := range n.attrValues {
		c.attrValues[k] = v
	}
	for _, child := range n.children {
		c.children = append(c.children, child.Clone())
	}
	return c
}

func (n *Node) orderOf(name string) int {
	if i := slices.Index(n.childOrder, name); i >= 0 {
		return i
	}
	return len(n.childOrder)
}

func (n *Node) insertAttrName(name string) {
	if len(n.attrOrder) == 0 {
		n.attrNames = append(n.attrNames, name)
		return
	}
	pos := attrRank(name, n.attrOrder)
	idx := len(n.attrNames)
	for i, existing := range n.attrNames {
		if attrRank(existing, n.attrOrder) > pos {
			idx = i
			break
		}
	}
	n.attrNames = slices.Insert(n.attrNames, idx, name)
}

// attrRank ordena primero las declaraciones de espacios de nombres, después
// los atributos con orden preferido y al final el resto.
func attrRank(name string, order []string) int {
	if name == "xmlns" || strings.HasPrefix(name, "xmlns:") {
		return -2
	}
	if name == "xsi:schemaLocation" {
		return -1
	}
	if i := slices.Index(order, name); i >= 0 {
		return i
	}
	return len(order)
}

func sortedAttrNames(attrs Attrs, order []string) []string {
	names := make([]string, 0, len(attrs))
	for k := range attrs {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool {
		ri, rj := attrRank(names[i], order), attrRank(names[j], order)
		if ri != rj {
			return ri < rj
		}
		return names[i] < names[j]
	})
	return names
}

// Element es cualquier valor que representa un nodo. Los elementos de CFDI
// (cfdi40.Comprobante, pagos20.Pagos, etc.) incrustan *Node, por lo que
// todos cumplen con esta interfaz.
type Element interface {
	AsNode() *Node
}

// AsNode devuelve el propio nodo, permite que *Node cumpla con Element.
func (n *Node) AsNode() *Node { return n }
