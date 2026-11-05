package nodes

import "strings"

// SchemaLocation es un par espacio de nombres / ubicación del XSD.
type SchemaLocation struct {
	Namespace string
	Location  string
}

// ParseSchemaLocations interpreta el contenido de un atributo
// xsi:schemaLocation como pares de espacio de nombres y ubicación.
// Si existe un elemento sin pareja se descarta.
func ParseSchemaLocations(value string) []SchemaLocation {
	parts := strings.Fields(value)
	var out []SchemaLocation
	for i := 0; i+1 < len(parts); i += 2 {
		out = append(out, SchemaLocation{Namespace: parts[i], Location: parts[i+1]})
	}
	return out
}

// FormatSchemaLocations genera el valor de un atributo xsi:schemaLocation.
func FormatSchemaLocations(locations []SchemaLocation) string {
	parts := make([]string, 0, len(locations)*2)
	for _, l := range locations {
		parts = append(parts, l.Namespace, l.Location)
	}
	return strings.Join(parts, " ")
}

// IsSatNamespace informa si el espacio de nombres pertenece al SAT
// (inicia con http://www.sat.gob.mx/).
func IsSatNamespace(namespace string) bool {
	return strings.HasPrefix(namespace, "http://www.sat.gob.mx/")
}

// MoveNamespaceDefinitionsToRoot mueve las declaraciones de espacios de
// nombres (xmlns:prefijo) y las ubicaciones de esquemas (xsi:schemaLocation)
// de los nodos descendientes al nodo raíz, siempre que el espacio de nombres
// pase el filtro. Si filter es nil se mueven todos.
//
// El SAT documenta como obligatorio que las declaraciones se encuentren en
// el nodo cfdi:Comprobante. Estas declaraciones no forman parte de la cadena
// original, por lo que moverlas no altera el sello.
func MoveNamespaceDefinitionsToRoot(root *Node, filter func(namespace string) bool) {
	if filter == nil {
		filter = func(string) bool { return true }
	}
	rootLocations := ParseSchemaLocations(root.Attr("xsi:schemaLocation"))
	hasLocation := func(ns string) bool {
		for _, l := range rootLocations {
			if l.Namespace == ns {
				return true
			}
		}
		return false
	}
	var visit func(n *Node)
	visit = func(n *Node) {
		for _, child := range n.children {
			// xmlns:prefijo del propio nodo
			for _, attr := range child.Attributes() {
				if !strings.HasPrefix(attr.Name, "xmlns:") || !filter(attr.Value) {
					continue
				}
				if current, ok := root.LookupAttr(attr.Name); ok && current != attr.Value {
					continue // el prefijo ya está usado en la raíz con otro valor
				}
				root.SetAttr(attr.Name, attr.Value)
				child.RemoveAttr(attr.Name)
			}
			// xmlns:xsi duplicado
			if v, ok := child.LookupAttr("xmlns:xsi"); ok && v == root.Attr("xmlns:xsi") {
				child.RemoveAttr("xmlns:xsi")
			}
			// xsi:schemaLocation
			if v, ok := child.LookupAttr("xsi:schemaLocation"); ok {
				var remaining []SchemaLocation
				for _, l := range ParseSchemaLocations(v) {
					if !filter(l.Namespace) {
						remaining = append(remaining, l)
						continue
					}
					if !hasLocation(l.Namespace) {
						rootLocations = append(rootLocations, l)
					}
				}
				if len(remaining) == 0 {
					child.RemoveAttr("xsi:schemaLocation")
				} else {
					child.SetAttr("xsi:schemaLocation", FormatSchemaLocations(remaining))
				}
			}
			visit(child)
		}
	}
	visit(root)
	if len(rootLocations) > 0 {
		root.SetAttr("xsi:schemaLocation", FormatSchemaLocations(rootLocations))
	}
}

// MoveSatDefinitionsToRoot mueve a la raíz las declaraciones de espacios de
// nombres y ubicaciones de esquemas que pertenecen al SAT.
func MoveSatDefinitionsToRoot(root *Node) {
	MoveNamespaceDefinitionsToRoot(root, IsSatNamespace)
}
