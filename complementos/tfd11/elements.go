// Package tfd11 permite construir el complemento Tfd11 del SAT.
package tfd11

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// Attrs contiene los atributos de un elemento.
type Attrs = nodes.Attrs

// Namespace es el espacio de nombres del complemento.
const Namespace = "http://www.sat.gob.mx/TimbreFiscalDigital"

// XsdLocation es la ubicación del esquema SAT.
const XsdLocation = "http://www.sat.gob.mx/sitio_internet/cfd/TimbreFiscalDigital/TimbreFiscalDigitalv11.xsd"

// Version es la versión del complemento.
const Version = "1.1"

func newElement(name string, order, children []string, attrs Attrs) *nodes.Node {
	return nodes.New(name, nil).SetAttributesOrder(order...).SetChildrenOrder(children...).SetAttrs(attrs)
}

// TimbreFiscalDigital representa el elemento tfd:TimbreFiscalDigital.
type TimbreFiscalDigital struct{ *nodes.Node }

// NewTimbreFiscalDigital crea el elemento tfd:TimbreFiscalDigital.
func NewTimbreFiscalDigital(attrs Attrs) *TimbreFiscalDigital {
	n := newElement("tfd:TimbreFiscalDigital", []string{"Version", "UUID", "FechaTimbrado", "RfcProvCertif", "Leyenda", "SelloCFD", "NoCertificadoSAT"}, []string{}, attrs)

	n.SetAttr("xmlns:tfd", Namespace).SetAttr("xsi:schemaLocation", Namespace+" "+XsdLocation).SetAttr("Version", Version)

	return &TimbreFiscalDigital{n}
}
