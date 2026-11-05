// Package implocal10 permite construir el complemento ImpLocal10 del SAT.
package implocal10

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// Attrs contiene los atributos de un elemento.
type Attrs = nodes.Attrs

// Namespace es el espacio de nombres del complemento.
const Namespace = "http://www.sat.gob.mx/implocal"

// XsdLocation es la ubicación del esquema SAT.
const XsdLocation = "http://www.sat.gob.mx/sitio_internet/cfd/implocal/implocal.xsd"

// Version es la versión del complemento.
const Version = "1.0"

func newElement(name string, order, children []string, attrs Attrs) *nodes.Node {
	return nodes.New(name, nil).SetAttributesOrder(order...).SetChildrenOrder(children...).SetAttrs(attrs)
}

// ImpuestosLocales representa el elemento implocal:ImpuestosLocales.
type ImpuestosLocales struct{ *nodes.Node }

// NewImpuestosLocales crea el elemento implocal:ImpuestosLocales.
func NewImpuestosLocales(attrs Attrs) *ImpuestosLocales {
	n := newElement("implocal:ImpuestosLocales", []string{"version", "TotaldeRetenciones", "TotaldeTraslados", "ImpLocRetenido", "TasadeRetencion", "Importe", "ImpLocTrasladado", "TasadeTraslado"}, []string{"implocal:RetencionesLocales", "implocal:TrasladosLocales"}, attrs)

	n.SetAttr("xmlns:implocal", Namespace).SetAttr("xsi:schemaLocation", Namespace+" "+XsdLocation).SetAttr("version", Version)

	return &ImpuestosLocales{n}
}

// AddRetencionLocal agrega atributos al elemento RetencionesLocales y lo devuelve.
func (e *ImpuestosLocales) AddRetencionLocal(attributes ...Attrs) *RetencionesLocales {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewRetencionesLocales(attrs)
	e.AddChild(c.Node)
	return c
}

// AddTrasladoLocal agrega atributos al elemento TrasladosLocales y lo devuelve.
func (e *ImpuestosLocales) AddTrasladoLocal(attributes ...Attrs) *TrasladosLocales {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewTrasladosLocales(attrs)
	e.AddChild(c.Node)
	return c
}

// RetencionesLocales representa el elemento implocal:RetencionesLocales.
type RetencionesLocales struct{ *nodes.Node }

// NewRetencionesLocales crea el elemento implocal:RetencionesLocales.
func NewRetencionesLocales(attrs Attrs) *RetencionesLocales {
	n := newElement("implocal:RetencionesLocales", []string{"ImpLocRetenido", "TasadeRetencion", "Importe"}, []string{}, attrs)

	return &RetencionesLocales{n}
}

// TrasladosLocales representa el elemento implocal:TrasladosLocales.
type TrasladosLocales struct{ *nodes.Node }

// NewTrasladosLocales crea el elemento implocal:TrasladosLocales.
func NewTrasladosLocales(attrs Attrs) *TrasladosLocales {
	n := newElement("implocal:TrasladosLocales", []string{"ImpLocTrasladado", "TasadeTraslado", "Importe"}, []string{}, attrs)

	return &TrasladosLocales{n}
}
