// Package pagos20 permite construir el complemento Pagos20 del SAT.
package pagos20

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// Attrs contiene los atributos de un elemento.
type Attrs = nodes.Attrs

// Namespace es el espacio de nombres del complemento.
const Namespace = "http://www.sat.gob.mx/Pagos20"

// XsdLocation es la ubicación del esquema SAT.
const XsdLocation = "http://www.sat.gob.mx/sitio_internet/cfd/Pagos/Pagos20.xsd"

// Version es la versión del complemento.
const Version = "2.0"

func newElement(name string, order, children []string, attrs Attrs) *nodes.Node {
	return nodes.New(name, nil).SetAttributesOrder(order...).SetChildrenOrder(children...).SetAttrs(attrs)
}

// DoctoRelacionado representa el elemento pago20:DoctoRelacionado.
type DoctoRelacionado struct{ *nodes.Node }

// NewDoctoRelacionado crea el elemento pago20:DoctoRelacionado.
func NewDoctoRelacionado(attrs Attrs) *DoctoRelacionado {
	n := newElement("pago20:DoctoRelacionado", []string{"IdDocumento", "Serie", "Folio", "MonedaDR", "EquivalenciaDR", "NumParcialidad", "ImpSaldoAnt", "ImpPagado", "ImpSaldoInsoluto", "ObjetoImpDR"}, []string{}, attrs)

	return &DoctoRelacionado{n}
}

// ImpuestosDR obtiene o crea el hijo único ImpuestosDR.
func (e *DoctoRelacionado) ImpuestosDR() *ImpuestosDR {
	c := NewImpuestosDR(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &ImpuestosDR{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddImpuestosDR agrega atributos al elemento ImpuestosDR y lo devuelve.
func (e *DoctoRelacionado) AddImpuestosDR(attributes ...Attrs) *ImpuestosDR {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.ImpuestosDR()
	c.SetAttrs(attrs)
	return c
}

// ImpuestosDR representa el elemento pago20:ImpuestosDR.
type ImpuestosDR struct{ *nodes.Node }

// NewImpuestosDR crea el elemento pago20:ImpuestosDR.
func NewImpuestosDR(attrs Attrs) *ImpuestosDR {
	n := newElement("pago20:ImpuestosDR", []string{}, []string{"pago20:RetencionesDR", "pago20:TrasladosDR"}, attrs)

	return &ImpuestosDR{n}
}

// RetencionesDR obtiene o crea el hijo único RetencionesDR.
func (e *ImpuestosDR) RetencionesDR() *RetencionesDR {
	c := NewRetencionesDR(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &RetencionesDR{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddRetencionesDR agrega atributos al elemento RetencionesDR y lo devuelve.
func (e *ImpuestosDR) AddRetencionesDR(attributes ...Attrs) *RetencionesDR {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.RetencionesDR()
	c.SetAttrs(attrs)
	return c
}

// TrasladosDR obtiene o crea el hijo único TrasladosDR.
func (e *ImpuestosDR) TrasladosDR() *TrasladosDR {
	c := NewTrasladosDR(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &TrasladosDR{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddTrasladosDR agrega atributos al elemento TrasladosDR y lo devuelve.
func (e *ImpuestosDR) AddTrasladosDR(attributes ...Attrs) *TrasladosDR {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.TrasladosDR()
	c.SetAttrs(attrs)
	return c
}

// ImpuestosP representa el elemento pago20:ImpuestosP.
type ImpuestosP struct{ *nodes.Node }

// NewImpuestosP crea el elemento pago20:ImpuestosP.
func NewImpuestosP(attrs Attrs) *ImpuestosP {
	n := newElement("pago20:ImpuestosP", []string{}, []string{"pago20:RetencionesP", "pago20:TrasladosP"}, attrs)

	return &ImpuestosP{n}
}

// RetencionesP obtiene o crea el hijo único RetencionesP.
func (e *ImpuestosP) RetencionesP() *RetencionesP {
	c := NewRetencionesP(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &RetencionesP{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddRetencionesP agrega atributos al elemento RetencionesP y lo devuelve.
func (e *ImpuestosP) AddRetencionesP(attributes ...Attrs) *RetencionesP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.RetencionesP()
	c.SetAttrs(attrs)
	return c
}

// TrasladosP obtiene o crea el hijo único TrasladosP.
func (e *ImpuestosP) TrasladosP() *TrasladosP {
	c := NewTrasladosP(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &TrasladosP{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddTrasladosP agrega atributos al elemento TrasladosP y lo devuelve.
func (e *ImpuestosP) AddTrasladosP(attributes ...Attrs) *TrasladosP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.TrasladosP()
	c.SetAttrs(attrs)
	return c
}

// Pago representa el elemento pago20:Pago.
type Pago struct{ *nodes.Node }

// NewPago crea el elemento pago20:Pago.
func NewPago(attrs Attrs) *Pago {
	n := newElement("pago20:Pago", []string{"FechaPago", "FormaDePagoP", "MonedaP", "TipoCambioP", "Monto", "NumOperacion", "RfcEmisorCtaOrd", "NomBancoOrdExt", "CtaOrdenante", "RfcEmisorCtaBen", "CtaBeneficiario", "TipoCadPago", "CertPago", "CadPago", "SelloPago"}, []string{"pago20:DoctoRelacionado", "pago20:ImpuestosP"}, attrs)

	return &Pago{n}
}

// AddDoctoRelacionado agrega atributos al elemento DoctoRelacionado y lo devuelve.
func (e *Pago) AddDoctoRelacionado(attributes ...Attrs) *DoctoRelacionado {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewDoctoRelacionado(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiDoctoRelacionado agrega varios elementos y devuelve el contenedor.
func (e *Pago) MultiDoctoRelacionado(attrs ...Attrs) *Pago {
	for _, a := range attrs {
		e.AddDoctoRelacionado(a)
	}
	return e
}

// ImpuestosP obtiene o crea el hijo único ImpuestosP.
func (e *Pago) ImpuestosP() *ImpuestosP {
	c := NewImpuestosP(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &ImpuestosP{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddImpuestosP agrega atributos al elemento ImpuestosP y lo devuelve.
func (e *Pago) AddImpuestosP(attributes ...Attrs) *ImpuestosP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.ImpuestosP()
	c.SetAttrs(attrs)
	return c
}

// Pagos representa el elemento pago20:Pagos.
type Pagos struct{ *nodes.Node }

// NewPagos crea el elemento pago20:Pagos.
func NewPagos(attrs Attrs) *Pagos {
	n := newElement("pago20:Pagos", []string{"Version"}, []string{"pago20:Totales", "pago20:Pago"}, attrs)

	n.SetAttr("xmlns:pago20", Namespace).SetAttr("xsi:schemaLocation", Namespace+" "+XsdLocation).SetAttr("Version", Version)

	return &Pagos{n}
}

// Totales obtiene o crea el hijo único Totales.
func (e *Pagos) Totales() *Totales {
	c := NewTotales(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Totales{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddTotales agrega atributos al elemento Totales y lo devuelve.
func (e *Pagos) AddTotales(attributes ...Attrs) *Totales {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Totales()
	c.SetAttrs(attrs)
	return c
}

// AddPago agrega atributos al elemento Pago y lo devuelve.
func (e *Pagos) AddPago(attributes ...Attrs) *Pago {
	c := NewPago(nil)
	for _, attrs := range attributes {
		c.SetAttrs(attrs)
	}
	e.AddChild(c.Node)
	return c
}

// MultiPago agrega varios elementos y devuelve el contenedor.
func (e *Pagos) MultiPago(attrs ...Attrs) *Pagos {
	for _, a := range attrs {
		e.AddPago(a)
	}
	return e
}

// RetencionDR representa el elemento pago20:RetencionDR.
type RetencionDR struct{ *nodes.Node }

// NewRetencionDR crea el elemento pago20:RetencionDR.
func NewRetencionDR(attrs Attrs) *RetencionDR {
	n := newElement("pago20:RetencionDR", []string{"BaseDR", "ImpuestoDR", "TipoFactorDR", "TasaOCuotaDR", "ImporteDR"}, []string{}, attrs)

	return &RetencionDR{n}
}

// RetencionP representa el elemento pago20:RetencionP.
type RetencionP struct{ *nodes.Node }

// NewRetencionP crea el elemento pago20:RetencionP.
func NewRetencionP(attrs Attrs) *RetencionP {
	n := newElement("pago20:RetencionP", []string{"ImpuestoP", "ImporteP"}, []string{}, attrs)

	return &RetencionP{n}
}

// RetencionesDR representa el elemento pago20:RetencionesDR.
type RetencionesDR struct{ *nodes.Node }

// NewRetencionesDR crea el elemento pago20:RetencionesDR.
func NewRetencionesDR(attrs Attrs) *RetencionesDR {
	n := newElement("pago20:RetencionesDR", []string{}, []string{}, attrs)

	return &RetencionesDR{n}
}

// AddRetencionDR agrega atributos al elemento RetencionDR y lo devuelve.
func (e *RetencionesDR) AddRetencionDR(attributes ...Attrs) *RetencionDR {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewRetencionDR(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiRetencionDR agrega varios elementos y devuelve el contenedor.
func (e *RetencionesDR) MultiRetencionDR(attrs ...Attrs) *RetencionesDR {
	for _, a := range attrs {
		e.AddRetencionDR(a)
	}
	return e
}

// RetencionesP representa el elemento pago20:RetencionesP.
type RetencionesP struct{ *nodes.Node }

// NewRetencionesP crea el elemento pago20:RetencionesP.
func NewRetencionesP(attrs Attrs) *RetencionesP {
	n := newElement("pago20:RetencionesP", []string{}, []string{}, attrs)

	return &RetencionesP{n}
}

// AddRetencionP agrega atributos al elemento RetencionP y lo devuelve.
func (e *RetencionesP) AddRetencionP(attributes ...Attrs) *RetencionP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewRetencionP(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiRetencionP agrega varios elementos y devuelve el contenedor.
func (e *RetencionesP) MultiRetencionP(attrs ...Attrs) *RetencionesP {
	for _, a := range attrs {
		e.AddRetencionP(a)
	}
	return e
}

// Totales representa el elemento pago20:Totales.
type Totales struct{ *nodes.Node }

// NewTotales crea el elemento pago20:Totales.
func NewTotales(attrs Attrs) *Totales {
	n := newElement("pago20:Totales", []string{"TotalRetencionesIVA", "TotalRetencionesISR", "TotalRetencionesIEPS", "TotalTrasladosBaseIVA16", "TotalTrasladosImpuestoIVA16", "TotalTrasladosBaseIVA8", "TotalTrasladosImpuestoIVA8", "TotalTrasladosBaseIVA0", "TotalTrasladosImpuestoIVA0", "TotalTrasladosBaseIVAExento", "MontoTotalPagos"}, []string{}, attrs)

	return &Totales{n}
}

// TrasladoDR representa el elemento pago20:TrasladoDR.
type TrasladoDR struct{ *nodes.Node }

// NewTrasladoDR crea el elemento pago20:TrasladoDR.
func NewTrasladoDR(attrs Attrs) *TrasladoDR {
	n := newElement("pago20:TrasladoDR", []string{"BaseDR", "ImpuestoDR", "TipoFactorDR", "TasaOCuotaDR", "ImporteDR"}, []string{}, attrs)

	return &TrasladoDR{n}
}

// TrasladoP representa el elemento pago20:TrasladoP.
type TrasladoP struct{ *nodes.Node }

// NewTrasladoP crea el elemento pago20:TrasladoP.
func NewTrasladoP(attrs Attrs) *TrasladoP {
	n := newElement("pago20:TrasladoP", []string{"BaseP", "ImpuestoP", "TipoFactorP", "TasaOCuotaP", "ImporteP"}, []string{}, attrs)

	return &TrasladoP{n}
}

// TrasladosDR representa el elemento pago20:TrasladosDR.
type TrasladosDR struct{ *nodes.Node }

// NewTrasladosDR crea el elemento pago20:TrasladosDR.
func NewTrasladosDR(attrs Attrs) *TrasladosDR {
	n := newElement("pago20:TrasladosDR", []string{}, []string{}, attrs)

	return &TrasladosDR{n}
}

// AddTrasladoDR agrega atributos al elemento TrasladoDR y lo devuelve.
func (e *TrasladosDR) AddTrasladoDR(attributes ...Attrs) *TrasladoDR {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewTrasladoDR(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiTrasladoDR agrega varios elementos y devuelve el contenedor.
func (e *TrasladosDR) MultiTrasladoDR(attrs ...Attrs) *TrasladosDR {
	for _, a := range attrs {
		e.AddTrasladoDR(a)
	}
	return e
}

// TrasladosP representa el elemento pago20:TrasladosP.
type TrasladosP struct{ *nodes.Node }

// NewTrasladosP crea el elemento pago20:TrasladosP.
func NewTrasladosP(attrs Attrs) *TrasladosP {
	n := newElement("pago20:TrasladosP", []string{}, []string{}, attrs)

	return &TrasladosP{n}
}

// AddTrasladoP agrega atributos al elemento TrasladoP y lo devuelve.
func (e *TrasladosP) AddTrasladoP(attributes ...Attrs) *TrasladoP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewTrasladoP(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiTrasladoP agrega varios elementos y devuelve el contenedor.
func (e *TrasladosP) MultiTrasladoP(attrs ...Attrs) *TrasladosP {
	for _, a := range attrs {
		e.AddTrasladoP(a)
	}
	return e
}
