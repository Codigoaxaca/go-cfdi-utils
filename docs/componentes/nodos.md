# Estructura de datos `nodes`

El paquete `nodes` permite administrar en memoria un documento XML como un árbol de nodos.
Cada nodo tiene un nombre (con prefijo, por ejemplo `cfdi:Comprobante`), atributos ordenados,
hijos y un valor de texto simple. Es la base de los [elementos](../crear/elementos-cfdi-40.md)
y de los complementos.

```go
import "github.com/codigoaxaca/go-cfdi-utils/nodes"
```

## Crear nodos

```go
emisor := nodes.New("cfdi:Emisor", nodes.Attrs{
	"Rfc":           "EKU9003173C9",
	"RegimenFiscal": "601",
})

root := nodes.New("cfdi:Comprobante", nodes.Attrs{"Version": "4.0"}, emisor)
```

## Atributos

- `Attr(name) string`: devuelve el valor, cadena vacía si no existe.
- `LookupAttr(name) (string, bool)`: devuelve el valor y si existe.
- `HasAttr(name) bool`: informa si existe (aunque esté vacío).
- `SetAttr(name, value) *Node` y `SetAttrs(attrs) *Node`.
- `RemoveAttr(name)`, `ClearAttrs()`.
- `Attributes() []Attribute`: copia ordenada de los atributos.
- `AttrsMap() Attrs`: copia como mapa.
- `SetAttributesOrder(names...)`: establece el orden preferido de los atributos.

```go
n := nodes.New("root", nodes.Attrs{"id": "1"})
fmt.Println(n.Attr("id"))        // 1
fmt.Println(n.Attr("no-existe")) // ""
n.SetAttr("atributo", "valor")
n.RemoveAttr("id")
for _, a := range n.Attributes() {
	fmt.Println(a.Name, a.Value)
}
```

## Hijos

- `Children() []*Node`, `Len() int`, `Child(i) *Node`.
- `AddChild(child) *Node`: agrega respetando el orden de hijos (`SetChildrenOrder`).
- `RemoveChild(child) bool`, `ClearChildren()`, `IndexOf(child)`.
- `FirstChild(name) *Node`, `ChildrenNamed(name) []*Node`.
- `SetChildrenOrder(names...)`: define el orden por nombre en que deben aparecer los hijos.
- `Walk(fn)`: recorre el nodo y sus descendientes.
- `Clone() *Node`: copia profunda.

## Búsqueda

- `SearchNode(path...) *Node`: primer nodo que coincide con la ruta de nombres, o `nil`.
- `SearchNodes(path...) []*Node`: todos los nodos que coinciden con el último nombre de la ruta.
- `SearchAttribute(path..., attr) string`: valor del atributo del nodo encontrado, o `""`.

```go
tfd := comprobante.SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital")
conceptos := comprobante.SearchNodes("cfdi:Conceptos", "cfdi:Concepto")
uuid := comprobante.SearchAttribute("cfdi:Complemento", "tfd:TimbreFiscalDigital", "UUID")
```

Los nodos devueltos son los mismos del árbol, por lo que modificarlos modifica el documento.

## Leer y escribir XML

```go
root, err := nodes.ParseXML(data)            // []byte
root, err := nodes.ParseXMLString(s)         // string
root, err := nodes.ParseXMLFile("cfdi.xml")  // archivo

xml := root.XML()     // con declaración <?xml ...?> y sangría
xml  = root.String()  // sin declaración ni formato
xml  = root.MarshalWith(nodes.MarshalOptions{Declaration: true, Indent: "\t"})
err  = root.WriteFile("salida.xml")
```

Advertencias:

- Los nodos no son una reescritura fiel de DOM: solo contienen atributos, hijos y texto simple.
  Se pierden comentarios e instrucciones de procesamiento.
- Los nombres se conservan tal como aparecen en el documento (con prefijo); no se resuelven
  espacios de nombres.
- Se aceptan documentos UTF-8 e ISO-8859-1.

## Contenido de texto

Algunos complementos (como el de ventas al detalle) usan texto en lugar de atributos.
Para ello se usan `Value()` y `SetValue(value)`.

## Espacios de nombres

- `MoveSatDefinitionsToRoot(root)`: mueve `xmlns:*` y `xsi:schemaLocation` del SAT al nodo raíz.
- `MoveNamespaceDefinitionsToRoot(root, filter)`: igual pero con un filtro propio.
- `ParseSchemaLocations(value) []SchemaLocation` y `FormatSchemaLocations(list) string`.

## Interfaz `Element`

```go
type Element interface {
	AsNode() *nodes.Node
}
```

`*nodes.Node` y todos los elementos (que incrustan `*nodes.Node`) la cumplen. Los métodos que
reciben "cualquier nodo", como `Comprobante.AddComplemento`, usan esta interfaz.
