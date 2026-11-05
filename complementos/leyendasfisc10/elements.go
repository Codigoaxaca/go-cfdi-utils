// Package leyendasfisc10 permite construir el complemento LeyendasFiscales10 del SAT.
package leyendasfisc10

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// Attrs contiene los atributos de un elemento.
type Attrs = nodes.Attrs

// Namespace es el espacio de nombres del complemento.
const Namespace = "http://www.sat.gob.mx/leyendasFiscales"

// XsdLocation es la ubicación del esquema SAT.
const XsdLocation = "http://www.sat.gob.mx/sitio_internet/cfd/leyendasFiscales/leyendasFisc.xsd"

// Version es la versión del complemento.
const Version = "1.0"

func newElement(name string, order, children []string, attrs Attrs) *nodes.Node {
	return nodes.New(name, nil).SetAttributesOrder(order...).SetChildrenOrder(children...).SetAttrs(attrs)
}

// Leyenda representa el elemento leyendasFisc:Leyenda.
type Leyenda struct{ *nodes.Node }

// NewLeyenda crea el elemento leyendasFisc:Leyenda.
func NewLeyenda(attrs Attrs) *Leyenda {
	n := newElement("leyendasFisc:Leyenda", []string{"disposicionFiscal", "norma", "textoLeyenda"}, []string{}, attrs)

	return &Leyenda{n}
}

// LeyendasFiscales representa el elemento leyendasFisc:LeyendasFiscales.
type LeyendasFiscales struct{ *nodes.Node }

// NewLeyendasFiscales crea el elemento leyendasFisc:LeyendasFiscales.
func NewLeyendasFiscales(attrs Attrs) *LeyendasFiscales {
	n := newElement("leyendasFisc:LeyendasFiscales", []string{"version"}, []string{}, attrs)

	n.SetAttr("xmlns:leyendasFisc", Namespace).SetAttr("xsi:schemaLocation", Namespace+" "+XsdLocation).SetAttr("version", Version)

	return &LeyendasFiscales{n}
}

// AddLeyenda agrega atributos al elemento Leyenda y lo devuelve.
func (e *LeyendasFiscales) AddLeyenda(attributes ...Attrs) *Leyenda {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewLeyenda(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiLeyenda agrega varios elementos y devuelve el contenedor.
func (e *LeyendasFiscales) MultiLeyenda(attrs ...Attrs) *LeyendasFiscales {
	for _, a := range attrs {
		e.AddLeyenda(a)
	}
	return e
}
