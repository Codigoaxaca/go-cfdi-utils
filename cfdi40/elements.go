package cfdi40

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// Attrs es un mapa de atributos nombre => valor.
type Attrs = nodes.Attrs

const (
	// Namespace es el espacio de nombres de CFDI 4.0.
	Namespace = "http://www.sat.gob.mx/cfd/4"
	// XsdLocation es la ubicación del esquema XSD de CFDI 4.0.
	XsdLocation = "http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd"
	// XsiNamespace es el espacio de nombres de XML Schema Instance.
	XsiNamespace = "http://www.w3.org/2001/XMLSchema-instance"
	// Version es la versión del comprobante.
	Version = "4.0"
)

// newElement crea un nodo con nombre, orden de atributos y orden de hijos.
func newElement(name string, attrOrder, childOrder []string, attrs Attrs) *nodes.Node {
	n := nodes.New(name, nil)
	n.SetAttributesOrder(attrOrder...)
	n.SetChildrenOrder(childOrder...)
	n.SetAttrs(attrs)
	return n
}

// prepare establece en un nodo existente el orden de atributos y de hijos.
func prepare(n *nodes.Node, attrOrder, childOrder []string) *nodes.Node {
	if len(n.AttributesOrder()) == 0 && len(attrOrder) > 0 {
		n.SetAttributesOrder(attrOrder...)
	}
	if len(n.ChildrenOrder()) == 0 && len(childOrder) > 0 {
		n.SetChildrenOrder(childOrder...)
	}
	return n
}

// getOrAdd obtiene el primer hijo con el nombre dado o lo crea con factory.
func getOrAdd(parent *nodes.Node, name string, factory func() *nodes.Node) *nodes.Node {
	if child := parent.FirstChild(name); child != nil {
		return child
	}
	return parent.AddChild(factory())
}

// ---------------------------------------------------------------------------
// Comprobante

var (
	comprobanteAttrs = []string{
		"Version", "Serie", "Folio", "Fecha", "Sello", "FormaPago", "NoCertificado", "Certificado",
		"CondicionesDePago", "SubTotal", "Descuento", "Moneda", "TipoCambio", "Total",
		"TipoDeComprobante", "Exportacion", "MetodoPago", "LugarExpedicion", "Confirmacion",
	}
	comprobanteChildren = []string{
		"cfdi:InformacionGlobal", "cfdi:CfdiRelacionados", "cfdi:Emisor", "cfdi:Receptor",
		"cfdi:Conceptos", "cfdi:Impuestos", "cfdi:Complemento", "cfdi:Addenda",
	}
)

// Comprobante representa el nodo raíz cfdi:Comprobante.
type Comprobante struct{ *nodes.Node }

// NewComprobante crea un nodo cfdi:Comprobante con los atributos fijos
// (espacios de nombres, xsi:schemaLocation y Version="4.0") y los atributos dados.
func NewComprobante(attrs Attrs) *Comprobante {
	n := newElement("cfdi:Comprobante", comprobanteAttrs, comprobanteChildren, Attrs{
		"xmlns:cfdi":         Namespace,
		"xmlns:xsi":          XsiNamespace,
		"xsi:schemaLocation": Namespace + " " + XsdLocation,
		"Version":            Version,
	})
	n.SetAttrs(attrs)
	return &Comprobante{n}
}

// WrapComprobante usa un nodo existente (por ejemplo leído de un XML) como Comprobante.
func WrapComprobante(n *nodes.Node) *Comprobante {
	return &Comprobante{prepare(n, comprobanteAttrs, comprobanteChildren)}
}

// InformacionGlobal obtiene (y crea si no existe) el nodo único InformacionGlobal.
func (c *Comprobante) InformacionGlobal() *InformacionGlobal {
	return wrapInformacionGlobal(getOrAdd(c.Node, "cfdi:InformacionGlobal", func() *nodes.Node {
		return NewInformacionGlobal(nil).Node
	}))
}

// AddInformacionGlobal agrega atributos al nodo único InformacionGlobal y lo devuelve.
func (c *Comprobante) AddInformacionGlobal(attrs Attrs) *InformacionGlobal {
	e := c.InformacionGlobal()
	e.SetAttrs(attrs)
	return e
}

// AddCfdiRelacionados agrega y devuelve un nuevo nodo CfdiRelacionados.
func (c *Comprobante) AddCfdiRelacionados(attrs Attrs) *CfdiRelacionados {
	e := NewCfdiRelacionados(attrs)
	c.AddChild(e.Node)
	return e
}

// MultiCfdiRelacionados agrega varios nodos CfdiRelacionados y devuelve el comprobante.
func (c *Comprobante) MultiCfdiRelacionados(attrs ...Attrs) *Comprobante {
	for _, a := range attrs {
		c.AddCfdiRelacionados(a)
	}
	return c
}

// Emisor obtiene (y crea si no existe) el nodo único Emisor.
func (c *Comprobante) Emisor() *Emisor {
	return wrapEmisor(getOrAdd(c.Node, "cfdi:Emisor", func() *nodes.Node { return NewEmisor(nil).Node }))
}

// AddEmisor agrega atributos al nodo único Emisor y lo devuelve.
func (c *Comprobante) AddEmisor(attrs Attrs) *Emisor {
	e := c.Emisor()
	e.SetAttrs(attrs)
	return e
}

// Receptor obtiene (y crea si no existe) el nodo único Receptor.
func (c *Comprobante) Receptor() *Receptor {
	return wrapReceptor(getOrAdd(c.Node, "cfdi:Receptor", func() *nodes.Node { return NewReceptor(nil).Node }))
}

// AddReceptor agrega atributos al nodo único Receptor y lo devuelve.
func (c *Comprobante) AddReceptor(attrs Attrs) *Receptor {
	e := c.Receptor()
	e.SetAttrs(attrs)
	return e
}

// Conceptos obtiene (y crea si no existe) el nodo único Conceptos.
func (c *Comprobante) Conceptos() *Conceptos {
	return wrapConceptos(getOrAdd(c.Node, "cfdi:Conceptos", func() *nodes.Node { return NewConceptos().Node }))
}

// AddConcepto agrega y devuelve un nuevo nodo Concepto dentro de Conceptos.
func (c *Comprobante) AddConcepto(attrs Attrs) *Concepto {
	return c.Conceptos().AddConcepto(attrs)
}

// MultiConcepto agrega varios nodos Concepto y devuelve el comprobante.
func (c *Comprobante) MultiConcepto(attrs ...Attrs) *Comprobante {
	c.Conceptos().MultiConcepto(attrs...)
	return c
}

// Impuestos obtiene (y crea si no existe) el nodo único Impuestos del comprobante.
func (c *Comprobante) Impuestos() *Impuestos {
	return wrapImpuestos(getOrAdd(c.Node, "cfdi:Impuestos", func() *nodes.Node { return NewImpuestos(nil).Node }))
}

// AddImpuestos agrega atributos al nodo único Impuestos y lo devuelve.
func (c *Comprobante) AddImpuestos(attrs Attrs) *Impuestos {
	e := c.Impuestos()
	e.SetAttrs(attrs)
	return e
}

// AddTraslado agrega y devuelve un nuevo Traslado en Impuestos/Traslados.
func (c *Comprobante) AddTraslado(attrs Attrs) *Traslado {
	return c.Impuestos().Traslados().AddTraslado(attrs)
}

// MultiTraslado agrega varios Traslado en Impuestos/Traslados y devuelve el comprobante.
func (c *Comprobante) MultiTraslado(attrs ...Attrs) *Comprobante {
	c.Impuestos().Traslados().MultiTraslado(attrs...)
	return c
}

// AddRetencion agrega y devuelve una nueva Retencion en Impuestos/Retenciones.
func (c *Comprobante) AddRetencion(attrs Attrs) *Retencion {
	return c.Impuestos().Retenciones().AddRetencion(attrs)
}

// MultiRetencion agrega varias Retencion en Impuestos/Retenciones y devuelve el comprobante.
func (c *Comprobante) MultiRetencion(attrs ...Attrs) *Comprobante {
	c.Impuestos().Retenciones().MultiRetencion(attrs...)
	return c
}

// Complemento obtiene (y crea si no existe) el nodo único Complemento.
func (c *Comprobante) Complemento() *Complemento {
	return &Complemento{getOrAdd(c.Node, "cfdi:Complemento", func() *nodes.Node { return nodes.New("cfdi:Complemento", nil) })}
}

// AddComplemento agrega un complemento (por ejemplo pagos20.Pagos) dentro del
// nodo único Complemento y devuelve el comprobante.
func (c *Comprobante) AddComplemento(complemento nodes.Element) *Comprobante {
	c.Complemento().AddChild(complemento.AsNode())
	return c
}

// Addenda obtiene (y crea si no existe) el nodo único Addenda.
func (c *Comprobante) Addenda() *Addenda {
	return &Addenda{getOrAdd(c.Node, "cfdi:Addenda", func() *nodes.Node { return nodes.New("cfdi:Addenda", nil) })}
}

// AddAddenda agrega un nodo dentro del nodo único Addenda y devuelve el comprobante.
func (c *Comprobante) AddAddenda(addenda nodes.Element) *Comprobante {
	c.Addenda().AddChild(addenda.AsNode())
	return c
}

// ---------------------------------------------------------------------------
// InformacionGlobal

var informacionGlobalAttrs = []string{"Periodicidad", "Meses", "Año"}

// InformacionGlobal representa el nodo cfdi:InformacionGlobal.
type InformacionGlobal struct{ *nodes.Node }

// NewInformacionGlobal crea un nodo cfdi:InformacionGlobal.
func NewInformacionGlobal(attrs Attrs) *InformacionGlobal {
	return &InformacionGlobal{newElement("cfdi:InformacionGlobal", informacionGlobalAttrs, nil, attrs)}
}

func wrapInformacionGlobal(n *nodes.Node) *InformacionGlobal {
	return &InformacionGlobal{prepare(n, informacionGlobalAttrs, nil)}
}

// ---------------------------------------------------------------------------
// CfdiRelacionados

// CfdiRelacionados representa el nodo cfdi:CfdiRelacionados.
type CfdiRelacionados struct{ *nodes.Node }

// NewCfdiRelacionados crea un nodo cfdi:CfdiRelacionados.
func NewCfdiRelacionados(attrs Attrs) *CfdiRelacionados {
	return &CfdiRelacionados{newElement("cfdi:CfdiRelacionados", []string{"TipoRelacion"}, []string{"cfdi:CfdiRelacionado"}, attrs)}
}

// AddCfdiRelacionado agrega y devuelve un nuevo nodo CfdiRelacionado.
func (e *CfdiRelacionados) AddCfdiRelacionado(attrs Attrs) *CfdiRelacionado {
	child := &CfdiRelacionado{newElement("cfdi:CfdiRelacionado", []string{"UUID"}, nil, attrs)}
	e.AddChild(child.Node)
	return child
}

// MultiCfdiRelacionado agrega varios nodos CfdiRelacionado y devuelve el mismo CfdiRelacionados.
func (e *CfdiRelacionados) MultiCfdiRelacionado(attrs ...Attrs) *CfdiRelacionados {
	for _, a := range attrs {
		e.AddCfdiRelacionado(a)
	}
	return e
}

// CfdiRelacionado representa el nodo cfdi:CfdiRelacionado.
type CfdiRelacionado struct{ *nodes.Node }

// ---------------------------------------------------------------------------
// Emisor y Receptor

var (
	emisorAttrs   = []string{"Rfc", "Nombre", "RegimenFiscal", "FacAtrAdquirente"}
	receptorAttrs = []string{"Rfc", "Nombre", "DomicilioFiscalReceptor", "ResidenciaFiscal", "NumRegIdTrib", "RegimenFiscalReceptor", "UsoCFDI"}
)

// Emisor representa el nodo cfdi:Emisor.
type Emisor struct{ *nodes.Node }

// NewEmisor crea un nodo cfdi:Emisor.
func NewEmisor(attrs Attrs) *Emisor {
	return &Emisor{newElement("cfdi:Emisor", emisorAttrs, nil, attrs)}
}

func wrapEmisor(n *nodes.Node) *Emisor { return &Emisor{prepare(n, emisorAttrs, nil)} }

// Receptor representa el nodo cfdi:Receptor.
type Receptor struct{ *nodes.Node }

// NewReceptor crea un nodo cfdi:Receptor.
func NewReceptor(attrs Attrs) *Receptor {
	return &Receptor{newElement("cfdi:Receptor", receptorAttrs, nil, attrs)}
}

func wrapReceptor(n *nodes.Node) *Receptor { return &Receptor{prepare(n, receptorAttrs, nil)} }

// ---------------------------------------------------------------------------
// Conceptos

// Conceptos representa el nodo cfdi:Conceptos.
type Conceptos struct{ *nodes.Node }

// NewConceptos crea un nodo cfdi:Conceptos.
func NewConceptos() *Conceptos {
	return &Conceptos{newElement("cfdi:Conceptos", nil, []string{"cfdi:Concepto"}, nil)}
}

func wrapConceptos(n *nodes.Node) *Conceptos {
	return &Conceptos{prepare(n, nil, []string{"cfdi:Concepto"})}
}

// AddConcepto agrega y devuelve un nuevo nodo Concepto.
func (e *Conceptos) AddConcepto(attrs Attrs) *Concepto {
	c := NewConcepto(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiConcepto agrega varios nodos Concepto y devuelve el mismo Conceptos.
func (e *Conceptos) MultiConcepto(attrs ...Attrs) *Conceptos {
	for _, a := range attrs {
		e.AddConcepto(a)
	}
	return e
}

// Items devuelve los conceptos existentes.
func (e *Conceptos) Items() []*Concepto {
	var out []*Concepto
	for _, n := range e.ChildrenNamed("cfdi:Concepto") {
		out = append(out, WrapConcepto(n))
	}
	return out
}

var (
	conceptoAttrs = []string{
		"ClaveProdServ", "NoIdentificacion", "Cantidad", "ClaveUnidad", "Unidad", "Descripcion",
		"ValorUnitario", "Importe", "Descuento", "ObjetoImp",
	}
	conceptoChildren = []string{
		"cfdi:Impuestos", "cfdi:ACuentaTerceros", "cfdi:InformacionAduanera", "cfdi:CuentaPredial",
		"cfdi:ComplementoConcepto", "cfdi:Parte",
	}
)

// Concepto representa el nodo cfdi:Concepto.
type Concepto struct{ *nodes.Node }

// NewConcepto crea un nodo cfdi:Concepto.
func NewConcepto(attrs Attrs) *Concepto {
	return &Concepto{newElement("cfdi:Concepto", conceptoAttrs, conceptoChildren, attrs)}
}

// WrapConcepto usa un nodo existente como Concepto.
func WrapConcepto(n *nodes.Node) *Concepto {
	return &Concepto{prepare(n, conceptoAttrs, conceptoChildren)}
}

// Impuestos obtiene (y crea si no existe) el nodo Impuestos del concepto.
func (e *Concepto) Impuestos() *ConceptoImpuestos {
	n := getOrAdd(e.Node, "cfdi:Impuestos", func() *nodes.Node {
		return newElement("cfdi:Impuestos", nil, []string{"cfdi:Traslados", "cfdi:Retenciones"}, nil)
	})
	return &ConceptoImpuestos{prepare(n, nil, []string{"cfdi:Traslados", "cfdi:Retenciones"})}
}

// AddTraslado agrega y devuelve un nuevo Traslado en Impuestos/Traslados del concepto.
func (e *Concepto) AddTraslado(attrs Attrs) *Traslado {
	return e.Impuestos().Traslados().AddTraslado(attrs)
}

// MultiTraslado agrega varios Traslado y devuelve el mismo concepto.
func (e *Concepto) MultiTraslado(attrs ...Attrs) *Concepto {
	e.Impuestos().Traslados().MultiTraslado(attrs...)
	return e
}

// AddRetencion agrega y devuelve una nueva Retencion en Impuestos/Retenciones del concepto.
func (e *Concepto) AddRetencion(attrs Attrs) *Retencion {
	return e.Impuestos().Retenciones().AddRetencion(attrs)
}

// MultiRetencion agrega varias Retencion y devuelve el mismo concepto.
func (e *Concepto) MultiRetencion(attrs ...Attrs) *Concepto {
	e.Impuestos().Retenciones().MultiRetencion(attrs...)
	return e
}

// ACuentaTerceros obtiene (y crea si no existe) el nodo único ACuentaTerceros.
func (e *Concepto) ACuentaTerceros() *ACuentaTerceros {
	attrs := []string{"RfcACuentaTerceros", "NombreACuentaTerceros", "RegimenFiscalACuentaTerceros", "DomicilioFiscalACuentaTerceros"}
	n := getOrAdd(e.Node, "cfdi:ACuentaTerceros", func() *nodes.Node {
		return newElement("cfdi:ACuentaTerceros", attrs, nil, nil)
	})
	return &ACuentaTerceros{prepare(n, attrs, nil)}
}

// AddACuentaTerceros agrega atributos al nodo único ACuentaTerceros y lo devuelve.
func (e *Concepto) AddACuentaTerceros(attrs Attrs) *ACuentaTerceros {
	c := e.ACuentaTerceros()
	c.SetAttrs(attrs)
	return c
}

// AddInformacionAduanera agrega y devuelve un nuevo nodo InformacionAduanera.
func (e *Concepto) AddInformacionAduanera(attrs Attrs) *InformacionAduanera {
	c := NewInformacionAduanera(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiInformacionAduanera agrega varios nodos InformacionAduanera y devuelve el mismo concepto.
func (e *Concepto) MultiInformacionAduanera(attrs ...Attrs) *Concepto {
	for _, a := range attrs {
		e.AddInformacionAduanera(a)
	}
	return e
}

// AddCuentaPredial agrega y devuelve un nuevo nodo CuentaPredial.
func (e *Concepto) AddCuentaPredial(attrs Attrs) *CuentaPredial {
	c := &CuentaPredial{newElement("cfdi:CuentaPredial", []string{"Numero"}, nil, attrs)}
	e.AddChild(c.Node)
	return c
}

// MultiCuentaPredial agrega varios nodos CuentaPredial y devuelve el mismo concepto.
func (e *Concepto) MultiCuentaPredial(attrs ...Attrs) *Concepto {
	for _, a := range attrs {
		e.AddCuentaPredial(a)
	}
	return e
}

// ComplementoConcepto obtiene (y crea si no existe) el nodo único ComplementoConcepto.
func (e *Concepto) ComplementoConcepto() *ComplementoConcepto {
	return &ComplementoConcepto{getOrAdd(e.Node, "cfdi:ComplementoConcepto", func() *nodes.Node {
		return nodes.New("cfdi:ComplementoConcepto", nil)
	})}
}

// AddComplementoConcepto agrega un complemento de concepto (por ejemplo
// instEducativas) dentro del nodo único ComplementoConcepto y devuelve el mismo concepto.
func (e *Concepto) AddComplementoConcepto(complemento nodes.Element) *Concepto {
	e.ComplementoConcepto().AddChild(complemento.AsNode())
	return e
}

// AddParte agrega y devuelve un nuevo nodo Parte.
func (e *Concepto) AddParte(attrs Attrs) *Parte {
	p := &Parte{newElement("cfdi:Parte",
		[]string{"ClaveProdServ", "NoIdentificacion", "Cantidad", "Unidad", "Descripcion", "ValorUnitario", "Importe"},
		[]string{"cfdi:InformacionAduanera"}, attrs)}
	e.AddChild(p.Node)
	return p
}

// MultiParte agrega varios nodos Parte y devuelve el mismo concepto.
func (e *Concepto) MultiParte(attrs ...Attrs) *Concepto {
	for _, a := range attrs {
		e.AddParte(a)
	}
	return e
}

// ConceptoImpuestos representa el nodo cfdi:Impuestos de un concepto.
// A diferencia de Impuestos del comprobante, aquí Traslados va antes que Retenciones.
type ConceptoImpuestos struct{ *nodes.Node }

// Traslados obtiene (y crea si no existe) el nodo único Traslados.
func (e *ConceptoImpuestos) Traslados() *Traslados { return traslados(e.Node) }

// Retenciones obtiene (y crea si no existe) el nodo único Retenciones.
func (e *ConceptoImpuestos) Retenciones() *Retenciones { return retenciones(e.Node) }

// ACuentaTerceros representa el nodo cfdi:ACuentaTerceros.
type ACuentaTerceros struct{ *nodes.Node }

// InformacionAduanera representa el nodo cfdi:InformacionAduanera.
type InformacionAduanera struct{ *nodes.Node }

// NewInformacionAduanera crea un nodo cfdi:InformacionAduanera.
func NewInformacionAduanera(attrs Attrs) *InformacionAduanera {
	return &InformacionAduanera{newElement("cfdi:InformacionAduanera", []string{"NumeroPedimento"}, nil, attrs)}
}

// CuentaPredial representa el nodo cfdi:CuentaPredial.
type CuentaPredial struct{ *nodes.Node }

// ComplementoConcepto representa el nodo cfdi:ComplementoConcepto.
type ComplementoConcepto struct{ *nodes.Node }

// Parte representa el nodo cfdi:Parte.
type Parte struct{ *nodes.Node }

// AddInformacionAduanera agrega y devuelve un nuevo nodo InformacionAduanera.
func (e *Parte) AddInformacionAduanera(attrs Attrs) *InformacionAduanera {
	c := NewInformacionAduanera(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiInformacionAduanera agrega varios nodos InformacionAduanera y devuelve la misma parte.
func (e *Parte) MultiInformacionAduanera(attrs ...Attrs) *Parte {
	for _, a := range attrs {
		e.AddInformacionAduanera(a)
	}
	return e
}

// ---------------------------------------------------------------------------
// Impuestos

var impuestosAttrs = []string{"TotalImpuestosRetenidos", "TotalImpuestosTrasladados"}

// Impuestos representa el nodo cfdi:Impuestos del comprobante.
// Aquí Retenciones va antes que Traslados.
type Impuestos struct{ *nodes.Node }

// NewImpuestos crea un nodo cfdi:Impuestos para el comprobante.
func NewImpuestos(attrs Attrs) *Impuestos {
	return &Impuestos{newElement("cfdi:Impuestos", impuestosAttrs, []string{"cfdi:Retenciones", "cfdi:Traslados"}, attrs)}
}

func wrapImpuestos(n *nodes.Node) *Impuestos {
	return &Impuestos{prepare(n, impuestosAttrs, []string{"cfdi:Retenciones", "cfdi:Traslados"})}
}

// Traslados obtiene (y crea si no existe) el nodo único Traslados.
func (e *Impuestos) Traslados() *Traslados { return traslados(e.Node) }

// Retenciones obtiene (y crea si no existe) el nodo único Retenciones.
func (e *Impuestos) Retenciones() *Retenciones { return retenciones(e.Node) }

var impuestoAttrs = []string{"Base", "Impuesto", "TipoFactor", "TasaOCuota", "Importe"}

func traslados(parent *nodes.Node) *Traslados {
	return &Traslados{getOrAdd(parent, "cfdi:Traslados", func() *nodes.Node {
		return newElement("cfdi:Traslados", nil, []string{"cfdi:Traslado"}, nil)
	})}
}

func retenciones(parent *nodes.Node) *Retenciones {
	return &Retenciones{getOrAdd(parent, "cfdi:Retenciones", func() *nodes.Node {
		return newElement("cfdi:Retenciones", nil, []string{"cfdi:Retencion"}, nil)
	})}
}

// Traslados representa el nodo cfdi:Traslados.
type Traslados struct{ *nodes.Node }

// AddTraslado agrega y devuelve un nuevo nodo Traslado.
func (e *Traslados) AddTraslado(attrs Attrs) *Traslado {
	t := &Traslado{newElement("cfdi:Traslado", impuestoAttrs, nil, attrs)}
	e.AddChild(t.Node)
	return t
}

// MultiTraslado agrega varios nodos Traslado y devuelve el mismo Traslados.
func (e *Traslados) MultiTraslado(attrs ...Attrs) *Traslados {
	for _, a := range attrs {
		e.AddTraslado(a)
	}
	return e
}

// Retenciones representa el nodo cfdi:Retenciones.
type Retenciones struct{ *nodes.Node }

// AddRetencion agrega y devuelve un nuevo nodo Retencion.
func (e *Retenciones) AddRetencion(attrs Attrs) *Retencion {
	r := &Retencion{newElement("cfdi:Retencion", impuestoAttrs, nil, attrs)}
	e.AddChild(r.Node)
	return r
}

// MultiRetencion agrega varios nodos Retencion y devuelve el mismo Retenciones.
func (e *Retenciones) MultiRetencion(attrs ...Attrs) *Retenciones {
	for _, a := range attrs {
		e.AddRetencion(a)
	}
	return e
}

// Traslado representa el nodo cfdi:Traslado.
type Traslado struct{ *nodes.Node }

// Retencion representa el nodo cfdi:Retencion.
type Retencion struct{ *nodes.Node }

// ---------------------------------------------------------------------------
// Complemento y Addenda

// Complemento representa el nodo cfdi:Complemento.
type Complemento struct{ *nodes.Node }

// Addenda representa el nodo cfdi:Addenda.
type Addenda struct{ *nodes.Node }
