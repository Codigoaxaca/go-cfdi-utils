// Package cartaporte31 permite construir el complemento CartaPorte31 del SAT.
package cartaporte31

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// Attrs contiene los atributos de un elemento.
type Attrs = nodes.Attrs

// Namespace es el espacio de nombres del complemento.
const Namespace = "http://www.sat.gob.mx/CartaPorte31"

// XsdLocation es la ubicación del esquema SAT.
const XsdLocation = "http://www.sat.gob.mx/sitio_internet/cfd/CartaPorte/CartaPorte31.xsd"

// Version es la versión del complemento.
const Version = "3.1"

func newElement(name string, order, children []string, attrs Attrs) *nodes.Node {
	return nodes.New(name, nil).SetAttributesOrder(order...).SetChildrenOrder(children...).SetAttrs(attrs)
}

// Autotransporte representa el elemento cartaporte31:Autotransporte.
type Autotransporte struct{ *nodes.Node }

// NewAutotransporte crea el elemento cartaporte31:Autotransporte.
func NewAutotransporte(attrs Attrs) *Autotransporte {
	n := newElement("cartaporte31:Autotransporte", []string{"PermSCT", "NumPermisoSCT"}, []string{"cartaporte31:IdentificacionVehicular", "cartaporte31:Seguros", "cartaporte31:Remolques"}, attrs)

	return &Autotransporte{n}
}

// IdentificacionVehicular obtiene o crea el hijo único IdentificacionVehicular.
func (e *Autotransporte) IdentificacionVehicular() *IdentificacionVehicular {
	c := NewIdentificacionVehicular(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &IdentificacionVehicular{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddIdentificacionVehicular agrega atributos al elemento IdentificacionVehicular y lo devuelve.
func (e *Autotransporte) AddIdentificacionVehicular(attributes ...Attrs) *IdentificacionVehicular {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.IdentificacionVehicular()
	c.SetAttrs(attrs)
	return c
}

// Seguros obtiene o crea el hijo único Seguros.
func (e *Autotransporte) Seguros() *Seguros {
	c := NewSeguros(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Seguros{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddSeguros agrega atributos al elemento Seguros y lo devuelve.
func (e *Autotransporte) AddSeguros(attributes ...Attrs) *Seguros {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Seguros()
	c.SetAttrs(attrs)
	return c
}

// Remolques obtiene o crea el hijo único Remolques.
func (e *Autotransporte) Remolques() *Remolques {
	c := NewRemolques(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Remolques{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddRemolques agrega atributos al elemento Remolques y lo devuelve.
func (e *Autotransporte) AddRemolques(attributes ...Attrs) *Remolques {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Remolques()
	c.SetAttrs(attrs)
	return c
}

// CantidadTransporta representa el elemento cartaporte31:CantidadTransporta.
type CantidadTransporta struct{ *nodes.Node }

// NewCantidadTransporta crea el elemento cartaporte31:CantidadTransporta.
func NewCantidadTransporta(attrs Attrs) *CantidadTransporta {
	n := newElement("cartaporte31:CantidadTransporta", []string{"Cantidad", "IDOrigen", "IDDestino", "CvesTransporte"}, []string{}, attrs)

	return &CantidadTransporta{n}
}

// Carro representa el elemento cartaporte31:Carro.
type Carro struct{ *nodes.Node }

// NewCarro crea el elemento cartaporte31:Carro.
func NewCarro(attrs Attrs) *Carro {
	n := newElement("cartaporte31:Carro", []string{"TipoCarro", "MatriculaCarro", "GuiaCarro", "ToneladasNetasCarro"}, []string{}, attrs)

	return &Carro{n}
}

// AddContenedor agrega atributos al elemento Contenedor y lo devuelve.
func (e *Carro) AddContenedor(attributes ...Attrs) *ContenedorFerroviario {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewContenedorFerroviario(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiContenedor agrega varios elementos y devuelve el contenedor.
func (e *Carro) MultiContenedor(attrs ...Attrs) *Carro {
	for _, a := range attrs {
		e.AddContenedor(a)
	}
	return e
}

// CartaPorte representa el elemento cartaporte31:CartaPorte.
type CartaPorte struct{ *nodes.Node }

// NewCartaPorte crea el elemento cartaporte31:CartaPorte.
func NewCartaPorte(attrs Attrs) *CartaPorte {
	n := newElement("cartaporte31:CartaPorte", []string{"Version", "IdCCP", "TranspInternac", "EntradaSalidaMerc", "PaisOrigenDestino", "ViaEntradaSalida", "TotalDistRec", "RegistroISTMO", "UbicacionPoloOrigen", "UbicacionPoloDestino"}, []string{"cartaporte31:RegimenesAduaneros", "cartaporte31:Ubicaciones", "cartaporte31:Mercancias", "cartaporte31:FiguraTransporte"}, attrs)

	n.SetAttr("xmlns:cartaporte31", Namespace).SetAttr("xsi:schemaLocation", Namespace+" "+XsdLocation).SetAttr("Version", Version)

	return &CartaPorte{n}
}

// RegimenesAduaneros obtiene o crea el hijo único RegimenesAduaneros.
func (e *CartaPorte) RegimenesAduaneros() *RegimenesAduaneros {
	c := NewRegimenesAduaneros(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &RegimenesAduaneros{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddRegimenesAduaneros agrega atributos al elemento RegimenesAduaneros y lo devuelve.
func (e *CartaPorte) AddRegimenesAduaneros(attributes ...Attrs) *RegimenesAduaneros {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.RegimenesAduaneros()
	c.SetAttrs(attrs)
	return c
}

// Ubicaciones obtiene o crea el hijo único Ubicaciones.
func (e *CartaPorte) Ubicaciones() *Ubicaciones {
	c := NewUbicaciones(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Ubicaciones{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddUbicaciones agrega atributos al elemento Ubicaciones y lo devuelve.
func (e *CartaPorte) AddUbicaciones(attributes ...Attrs) *Ubicaciones {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Ubicaciones()
	c.SetAttrs(attrs)
	return c
}

// Mercancias obtiene o crea el hijo único Mercancias.
func (e *CartaPorte) Mercancias() *Mercancias {
	c := NewMercancias(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Mercancias{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddMercancias agrega atributos al elemento Mercancias y lo devuelve.
func (e *CartaPorte) AddMercancias(attributes ...Attrs) *Mercancias {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Mercancias()
	c.SetAttrs(attrs)
	return c
}

// FiguraTransporte obtiene o crea el hijo único FiguraTransporte.
func (e *CartaPorte) FiguraTransporte() *FiguraTransporte {
	c := NewFiguraTransporte(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &FiguraTransporte{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddFiguraTransporte agrega atributos al elemento FiguraTransporte y lo devuelve.
func (e *CartaPorte) AddFiguraTransporte(attributes ...Attrs) *FiguraTransporte {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.FiguraTransporte()
	c.SetAttrs(attrs)
	return c
}

// ContenedorFerroviario representa el elemento cartaporte31:Contenedor.
type ContenedorFerroviario struct{ *nodes.Node }

// NewContenedorFerroviario crea el elemento cartaporte31:Contenedor.
func NewContenedorFerroviario(attrs Attrs) *ContenedorFerroviario {
	n := newElement("cartaporte31:Contenedor", []string{"TipoContenedor", "PesoContenedorVacio", "PesoNetoMercancia"}, []string{}, attrs)

	return &ContenedorFerroviario{n}
}

// ContenedorMaritimo representa el elemento cartaporte31:Contenedor.
type ContenedorMaritimo struct{ *nodes.Node }

// NewContenedorMaritimo crea el elemento cartaporte31:Contenedor.
func NewContenedorMaritimo(attrs Attrs) *ContenedorMaritimo {
	n := newElement("cartaporte31:Contenedor", []string{"TipoContenedor", "MatriculaContenedor", "NumPrecinto", "IdCCPRelacionado", "PlacaVMCCP", "FechaCertificacionCCP"}, []string{}, attrs)

	return &ContenedorMaritimo{n}
}

// RemolquesCCP obtiene o crea el hijo único RemolquesCCP.
func (e *ContenedorMaritimo) RemolquesCCP() *RemolquesCCP {
	c := NewRemolquesCCP(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &RemolquesCCP{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddRemolquesCCP agrega atributos al elemento RemolquesCCP y lo devuelve.
func (e *ContenedorMaritimo) AddRemolquesCCP(attributes ...Attrs) *RemolquesCCP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.RemolquesCCP()
	c.SetAttrs(attrs)
	return c
}

// DerechosDePaso representa el elemento cartaporte31:DerechosDePaso.
type DerechosDePaso struct{ *nodes.Node }

// NewDerechosDePaso crea el elemento cartaporte31:DerechosDePaso.
func NewDerechosDePaso(attrs Attrs) *DerechosDePaso {
	n := newElement("cartaporte31:DerechosDePaso", []string{"TipoDerechoDePaso", "KilometrajePagado"}, []string{}, attrs)

	return &DerechosDePaso{n}
}

// DetalleMercancia representa el elemento cartaporte31:DetalleMercancia.
type DetalleMercancia struct{ *nodes.Node }

// NewDetalleMercancia crea el elemento cartaporte31:DetalleMercancia.
func NewDetalleMercancia(attrs Attrs) *DetalleMercancia {
	n := newElement("cartaporte31:DetalleMercancia", []string{"UnidadPesoMerc", "PesoBruto", "PesoNeto", "PesoTara", "NumPiezas"}, []string{}, attrs)

	return &DetalleMercancia{n}
}

// DocumentacionAduanera representa el elemento cartaporte31:DocumentacionAduanera.
type DocumentacionAduanera struct{ *nodes.Node }

// NewDocumentacionAduanera crea el elemento cartaporte31:DocumentacionAduanera.
func NewDocumentacionAduanera(attrs Attrs) *DocumentacionAduanera {
	n := newElement("cartaporte31:DocumentacionAduanera", []string{"TipoDocumento", "NumPedimento", "IdentDocAduanero", "RFCImpo"}, []string{}, attrs)

	return &DocumentacionAduanera{n}
}

// Domicilio representa el elemento cartaporte31:Domicilio.
type Domicilio struct{ *nodes.Node }

// NewDomicilio crea el elemento cartaporte31:Domicilio.
func NewDomicilio(attrs Attrs) *Domicilio {
	n := newElement("cartaporte31:Domicilio", []string{"Calle", "NumeroExterior", "NumeroInterior", "Colonia", "Localidad", "Referencia", "Municipio", "Estado", "Pais", "CodigoPostal"}, []string{}, attrs)

	return &Domicilio{n}
}

// FiguraTransporte representa el elemento cartaporte31:FiguraTransporte.
type FiguraTransporte struct{ *nodes.Node }

// NewFiguraTransporte crea el elemento cartaporte31:FiguraTransporte.
func NewFiguraTransporte(attrs Attrs) *FiguraTransporte {
	n := newElement("cartaporte31:FiguraTransporte", []string{}, []string{}, attrs)

	return &FiguraTransporte{n}
}

// AddTiposFigura agrega atributos al elemento TiposFigura y lo devuelve.
func (e *FiguraTransporte) AddTiposFigura(attributes ...Attrs) *TiposFigura {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewTiposFigura(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiTiposFigura agrega varios elementos y devuelve el contenedor.
func (e *FiguraTransporte) MultiTiposFigura(attrs ...Attrs) *FiguraTransporte {
	for _, a := range attrs {
		e.AddTiposFigura(a)
	}
	return e
}

// GuiasIdentificacion representa el elemento cartaporte31:GuiasIdentificacion.
type GuiasIdentificacion struct{ *nodes.Node }

// NewGuiasIdentificacion crea el elemento cartaporte31:GuiasIdentificacion.
func NewGuiasIdentificacion(attrs Attrs) *GuiasIdentificacion {
	n := newElement("cartaporte31:GuiasIdentificacion", []string{"NumeroGuiaIdentificacion", "DescripGuiaIdentificacion", "PesoGuiaIdentificacion"}, []string{}, attrs)

	return &GuiasIdentificacion{n}
}

// IdentificacionVehicular representa el elemento cartaporte31:IdentificacionVehicular.
type IdentificacionVehicular struct{ *nodes.Node }

// NewIdentificacionVehicular crea el elemento cartaporte31:IdentificacionVehicular.
func NewIdentificacionVehicular(attrs Attrs) *IdentificacionVehicular {
	n := newElement("cartaporte31:IdentificacionVehicular", []string{"ConfigVehicular", "PesoBrutoVehicular", "PlacaVM", "AnioModeloVM"}, []string{}, attrs)

	return &IdentificacionVehicular{n}
}

// Mercancia representa el elemento cartaporte31:Mercancia.
type Mercancia struct{ *nodes.Node }

// NewMercancia crea el elemento cartaporte31:Mercancia.
func NewMercancia(attrs Attrs) *Mercancia {
	n := newElement("cartaporte31:Mercancia", []string{"BienesTransp", "ClaveSTCC", "Descripcion", "Cantidad", "ClaveUnidad", "Unidad", "Dimensiones", "MaterialPeligroso", "CveMaterialPeligroso", "Embalaje", "DescripEmbalaje", "SectorCOFEPRIS", "NombreIngredienteActivo", "NomQuimico", "DenominacionGenericaProd", "DenominacionDistintivaProd", "Fabricante", "FechaCaducidad", "LoteMedicamento", "FormaFarmaceutica", "CondicionesEspTransp", "RegistroSanitarioFolioAutorizacion", "PermisoImportacion", "FolioImpoVUCEM", "NumCAS", "RazonSocialEmpImp", "NumRegSanPlagCOFEPRIS", "DatosFabricante", "DatosFormulador", "DatosMaquilador", "UsoAutorizado", "PesoEnKg", "ValorMercancia", "Moneda", "FraccionArancelaria", "UUIDComercioExt", "TipoMateria", "DescripcionMateria"}, []string{"cartaporte31:DocumentacionAduanera", "cartaporte31:GuiasIdentificacion", "cartaporte31:CantidadTransporta", "cartaporte31:DetalleMercancia"}, attrs)

	return &Mercancia{n}
}

// AddDocumentacionAduanera agrega atributos al elemento DocumentacionAduanera y lo devuelve.
func (e *Mercancia) AddDocumentacionAduanera(attributes ...Attrs) *DocumentacionAduanera {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewDocumentacionAduanera(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiDocumentacionAduanera agrega varios elementos y devuelve el contenedor.
func (e *Mercancia) MultiDocumentacionAduanera(attrs ...Attrs) *Mercancia {
	for _, a := range attrs {
		e.AddDocumentacionAduanera(a)
	}
	return e
}

// AddGuiasIdentificacion agrega atributos al elemento GuiasIdentificacion y lo devuelve.
func (e *Mercancia) AddGuiasIdentificacion(attributes ...Attrs) *GuiasIdentificacion {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewGuiasIdentificacion(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiGuiasIdentificacion agrega varios elementos y devuelve el contenedor.
func (e *Mercancia) MultiGuiasIdentificacion(attrs ...Attrs) *Mercancia {
	for _, a := range attrs {
		e.AddGuiasIdentificacion(a)
	}
	return e
}

// AddCantidadTransporta agrega atributos al elemento CantidadTransporta y lo devuelve.
func (e *Mercancia) AddCantidadTransporta(attributes ...Attrs) *CantidadTransporta {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewCantidadTransporta(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiCantidadTransporta agrega varios elementos y devuelve el contenedor.
func (e *Mercancia) MultiCantidadTransporta(attrs ...Attrs) *Mercancia {
	for _, a := range attrs {
		e.AddCantidadTransporta(a)
	}
	return e
}

// DetalleMercancia obtiene o crea el hijo único DetalleMercancia.
func (e *Mercancia) DetalleMercancia() *DetalleMercancia {
	c := NewDetalleMercancia(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &DetalleMercancia{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddDetalleMercancia agrega atributos al elemento DetalleMercancia y lo devuelve.
func (e *Mercancia) AddDetalleMercancia(attributes ...Attrs) *DetalleMercancia {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.DetalleMercancia()
	c.SetAttrs(attrs)
	return c
}

// Mercancias representa el elemento cartaporte31:Mercancias.
type Mercancias struct{ *nodes.Node }

// NewMercancias crea el elemento cartaporte31:Mercancias.
func NewMercancias(attrs Attrs) *Mercancias {
	n := newElement("cartaporte31:Mercancias", []string{"PesoBrutoTotal", "UnidadPeso", "PesoNetoTotal", "NumTotalMercancias", "CargoPorTasacion", "LogisticaInversaRecoleccionDevolucion"}, []string{"cartaporte31:Mercancia", "cartaporte31:Autotransporte", "cartaporte31:TransporteMaritimo", "cartaporte31:TransporteAereo", "cartaporte31:TransporteFerroviario"}, attrs)

	return &Mercancias{n}
}

// AddMercancia agrega atributos al elemento Mercancia y lo devuelve.
func (e *Mercancias) AddMercancia(attributes ...Attrs) *Mercancia {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewMercancia(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiMercancia agrega varios elementos y devuelve el contenedor.
func (e *Mercancias) MultiMercancia(attrs ...Attrs) *Mercancias {
	for _, a := range attrs {
		e.AddMercancia(a)
	}
	return e
}

// Autotransporte obtiene o crea el hijo único Autotransporte.
func (e *Mercancias) Autotransporte() *Autotransporte {
	c := NewAutotransporte(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Autotransporte{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddAutotransporte agrega atributos al elemento Autotransporte y lo devuelve.
func (e *Mercancias) AddAutotransporte(attributes ...Attrs) *Autotransporte {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Autotransporte()
	c.SetAttrs(attrs)
	return c
}

// TransporteMaritimo obtiene o crea el hijo único TransporteMaritimo.
func (e *Mercancias) TransporteMaritimo() *TransporteMaritimo {
	c := NewTransporteMaritimo(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &TransporteMaritimo{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddTransporteMaritimo agrega atributos al elemento TransporteMaritimo y lo devuelve.
func (e *Mercancias) AddTransporteMaritimo(attributes ...Attrs) *TransporteMaritimo {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.TransporteMaritimo()
	c.SetAttrs(attrs)
	return c
}

// TransporteAereo obtiene o crea el hijo único TransporteAereo.
func (e *Mercancias) TransporteAereo() *TransporteAereo {
	c := NewTransporteAereo(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &TransporteAereo{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddTransporteAereo agrega atributos al elemento TransporteAereo y lo devuelve.
func (e *Mercancias) AddTransporteAereo(attributes ...Attrs) *TransporteAereo {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.TransporteAereo()
	c.SetAttrs(attrs)
	return c
}

// TransporteFerroviario obtiene o crea el hijo único TransporteFerroviario.
func (e *Mercancias) TransporteFerroviario() *TransporteFerroviario {
	c := NewTransporteFerroviario(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &TransporteFerroviario{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddTransporteFerroviario agrega atributos al elemento TransporteFerroviario y lo devuelve.
func (e *Mercancias) AddTransporteFerroviario(attributes ...Attrs) *TransporteFerroviario {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.TransporteFerroviario()
	c.SetAttrs(attrs)
	return c
}

// PartesTransporte representa el elemento cartaporte31:PartesTransporte.
type PartesTransporte struct{ *nodes.Node }

// NewPartesTransporte crea el elemento cartaporte31:PartesTransporte.
func NewPartesTransporte(attrs Attrs) *PartesTransporte {
	n := newElement("cartaporte31:PartesTransporte", []string{"ParteTransporte"}, []string{}, attrs)

	return &PartesTransporte{n}
}

// RegimenAduaneroCCP representa el elemento cartaporte31:RegimenAduaneroCCP.
type RegimenAduaneroCCP struct{ *nodes.Node }

// NewRegimenAduaneroCCP crea el elemento cartaporte31:RegimenAduaneroCCP.
func NewRegimenAduaneroCCP(attrs Attrs) *RegimenAduaneroCCP {
	n := newElement("cartaporte31:RegimenAduaneroCCP", []string{"RegimenAduanero"}, []string{}, attrs)

	return &RegimenAduaneroCCP{n}
}

// RegimenesAduaneros representa el elemento cartaporte31:RegimenesAduaneros.
type RegimenesAduaneros struct{ *nodes.Node }

// NewRegimenesAduaneros crea el elemento cartaporte31:RegimenesAduaneros.
func NewRegimenesAduaneros(attrs Attrs) *RegimenesAduaneros {
	n := newElement("cartaporte31:RegimenesAduaneros", []string{}, []string{}, attrs)

	return &RegimenesAduaneros{n}
}

// AddRegimenAduaneroCCP agrega atributos al elemento RegimenAduaneroCCP y lo devuelve.
func (e *RegimenesAduaneros) AddRegimenAduaneroCCP(attributes ...Attrs) *RegimenAduaneroCCP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewRegimenAduaneroCCP(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiRegimenAduaneroCCP agrega varios elementos y devuelve el contenedor.
func (e *RegimenesAduaneros) MultiRegimenAduaneroCCP(attrs ...Attrs) *RegimenesAduaneros {
	for _, a := range attrs {
		e.AddRegimenAduaneroCCP(a)
	}
	return e
}

// Remolque representa el elemento cartaporte31:Remolque.
type Remolque struct{ *nodes.Node }

// NewRemolque crea el elemento cartaporte31:Remolque.
func NewRemolque(attrs Attrs) *Remolque {
	n := newElement("cartaporte31:Remolque", []string{"SubTipoRem", "Placa"}, []string{}, attrs)

	return &Remolque{n}
}

// RemolqueCCP representa el elemento cartaporte31:RemolqueCCP.
type RemolqueCCP struct{ *nodes.Node }

// NewRemolqueCCP crea el elemento cartaporte31:RemolqueCCP.
func NewRemolqueCCP(attrs Attrs) *RemolqueCCP {
	n := newElement("cartaporte31:RemolqueCCP", []string{"SubTipoRemCCP", "PlacaCCP"}, []string{}, attrs)

	return &RemolqueCCP{n}
}

// Remolques representa el elemento cartaporte31:Remolques.
type Remolques struct{ *nodes.Node }

// NewRemolques crea el elemento cartaporte31:Remolques.
func NewRemolques(attrs Attrs) *Remolques {
	n := newElement("cartaporte31:Remolques", []string{}, []string{}, attrs)

	return &Remolques{n}
}

// AddRemolque agrega atributos al elemento Remolque y lo devuelve.
func (e *Remolques) AddRemolque(attributes ...Attrs) *Remolque {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewRemolque(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiRemolque agrega varios elementos y devuelve el contenedor.
func (e *Remolques) MultiRemolque(attrs ...Attrs) *Remolques {
	for _, a := range attrs {
		e.AddRemolque(a)
	}
	return e
}

// RemolquesCCP representa el elemento cartaporte31:RemolquesCCP.
type RemolquesCCP struct{ *nodes.Node }

// NewRemolquesCCP crea el elemento cartaporte31:RemolquesCCP.
func NewRemolquesCCP(attrs Attrs) *RemolquesCCP {
	n := newElement("cartaporte31:RemolquesCCP", []string{}, []string{}, attrs)

	return &RemolquesCCP{n}
}

// AddRemolqueCCP agrega atributos al elemento RemolqueCCP y lo devuelve.
func (e *RemolquesCCP) AddRemolqueCCP(attributes ...Attrs) *RemolqueCCP {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewRemolqueCCP(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiRemolqueCCP agrega varios elementos y devuelve el contenedor.
func (e *RemolquesCCP) MultiRemolqueCCP(attrs ...Attrs) *RemolquesCCP {
	for _, a := range attrs {
		e.AddRemolqueCCP(a)
	}
	return e
}

// Seguros representa el elemento cartaporte31:Seguros.
type Seguros struct{ *nodes.Node }

// NewSeguros crea el elemento cartaporte31:Seguros.
func NewSeguros(attrs Attrs) *Seguros {
	n := newElement("cartaporte31:Seguros", []string{"AseguraRespCivil", "PolizaRespCivil", "AseguraMedAmbiente", "PolizaMedAmbiente", "AseguraCarga", "PolizaCarga", "PrimaSeguro"}, []string{}, attrs)

	return &Seguros{n}
}

// TiposFigura representa el elemento cartaporte31:TiposFigura.
type TiposFigura struct{ *nodes.Node }

// NewTiposFigura crea el elemento cartaporte31:TiposFigura.
func NewTiposFigura(attrs Attrs) *TiposFigura {
	n := newElement("cartaporte31:TiposFigura", []string{"TipoFigura", "RFCFigura", "NumLicencia", "NombreFigura", "NumRegIdTribFigura", "ResidenciaFiscalFigura"}, []string{"cartaporte31:PartesTransporte", "cartaporte31:Domicilio"}, attrs)

	return &TiposFigura{n}
}

// AddPartesTransporte agrega atributos al elemento PartesTransporte y lo devuelve.
func (e *TiposFigura) AddPartesTransporte(attributes ...Attrs) *PartesTransporte {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewPartesTransporte(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiPartesTransporte agrega varios elementos y devuelve el contenedor.
func (e *TiposFigura) MultiPartesTransporte(attrs ...Attrs) *TiposFigura {
	for _, a := range attrs {
		e.AddPartesTransporte(a)
	}
	return e
}

// Domicilio obtiene o crea el hijo único Domicilio.
func (e *TiposFigura) Domicilio() *Domicilio {
	c := NewDomicilio(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Domicilio{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddDomicilio agrega atributos al elemento Domicilio y lo devuelve.
func (e *TiposFigura) AddDomicilio(attributes ...Attrs) *Domicilio {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Domicilio()
	c.SetAttrs(attrs)
	return c
}

// TransporteAereo representa el elemento cartaporte31:TransporteAereo.
type TransporteAereo struct{ *nodes.Node }

// NewTransporteAereo crea el elemento cartaporte31:TransporteAereo.
func NewTransporteAereo(attrs Attrs) *TransporteAereo {
	n := newElement("cartaporte31:TransporteAereo", []string{"PermSCT", "NumPermisoSCT", "MatriculaAeronave", "NombreAseg", "NumPolizaSeguro", "NumeroGuia", "LugarContrato", "CodigoTransportista", "RFCEmbarcador", "NumRegIdTribEmbarc", "ResidenciaFiscalEmbarc", "NombreEmbarcador"}, []string{}, attrs)

	return &TransporteAereo{n}
}

// TransporteFerroviario representa el elemento cartaporte31:TransporteFerroviario.
type TransporteFerroviario struct{ *nodes.Node }

// NewTransporteFerroviario crea el elemento cartaporte31:TransporteFerroviario.
func NewTransporteFerroviario(attrs Attrs) *TransporteFerroviario {
	n := newElement("cartaporte31:TransporteFerroviario", []string{"TipoDeServicio", "TipoDeTrafico", "NombreAseg", "NumPolizaSeguro"}, []string{"cartaporte31:DerechosDePaso", "cartaporte31:Carro"}, attrs)

	return &TransporteFerroviario{n}
}

// AddDerechosDePaso agrega atributos al elemento DerechosDePaso y lo devuelve.
func (e *TransporteFerroviario) AddDerechosDePaso(attributes ...Attrs) *DerechosDePaso {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewDerechosDePaso(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiDerechosDePaso agrega varios elementos y devuelve el contenedor.
func (e *TransporteFerroviario) MultiDerechosDePaso(attrs ...Attrs) *TransporteFerroviario {
	for _, a := range attrs {
		e.AddDerechosDePaso(a)
	}
	return e
}

// AddCarro agrega atributos al elemento Carro y lo devuelve.
func (e *TransporteFerroviario) AddCarro(attributes ...Attrs) *Carro {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewCarro(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiCarro agrega varios elementos y devuelve el contenedor.
func (e *TransporteFerroviario) MultiCarro(attrs ...Attrs) *TransporteFerroviario {
	for _, a := range attrs {
		e.AddCarro(a)
	}
	return e
}

// TransporteMaritimo representa el elemento cartaporte31:TransporteMaritimo.
type TransporteMaritimo struct{ *nodes.Node }

// NewTransporteMaritimo crea el elemento cartaporte31:TransporteMaritimo.
func NewTransporteMaritimo(attrs Attrs) *TransporteMaritimo {
	n := newElement("cartaporte31:TransporteMaritimo", []string{"PermSCT", "NumPermisoSCT", "NombreAseg", "NumPolizaSeguro", "TipoEmbarcacion", "Matricula", "NumeroOMI", "AnioEmbarcacion", "NombreEmbarc", "NacionalidadEmbarc", "UnidadesDeArqBruto", "TipoCarga", "Eslora", "Manga", "Calado", "Puntal", "LineaNaviera", "NombreAgenteNaviero", "NumAutorizacionNaviero", "NumViaje", "NumConocEmbarc", "PermisoTempNavegacion"}, []string{}, attrs)

	return &TransporteMaritimo{n}
}

// AddContenedor agrega atributos al elemento Contenedor y lo devuelve.
func (e *TransporteMaritimo) AddContenedor(attributes ...Attrs) *ContenedorMaritimo {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewContenedorMaritimo(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiContenedor agrega varios elementos y devuelve el contenedor.
func (e *TransporteMaritimo) MultiContenedor(attrs ...Attrs) *TransporteMaritimo {
	for _, a := range attrs {
		e.AddContenedor(a)
	}
	return e
}

// Ubicacion representa el elemento cartaporte31:Ubicacion.
type Ubicacion struct{ *nodes.Node }

// NewUbicacion crea el elemento cartaporte31:Ubicacion.
func NewUbicacion(attrs Attrs) *Ubicacion {
	n := newElement("cartaporte31:Ubicacion", []string{"TipoUbicacion", "IDUbicacion", "RFCRemitenteDestinatario", "NombreRemitenteDestinatario", "NumRegIdTrib", "ResidenciaFiscal", "NumEstacion", "NombreEstacion", "NavegacionTrafico", "FechaHoraSalidaLlegada", "TipoEstacion", "DistanciaRecorrida"}, []string{}, attrs)

	return &Ubicacion{n}
}

// Domicilio obtiene o crea el hijo único Domicilio.
func (e *Ubicacion) Domicilio() *Domicilio {
	c := NewDomicilio(nil)
	if n := e.FirstChild(c.Name()); n != nil {
		n.SetAttributesOrder(c.AttributesOrder()...).SetChildrenOrder(c.ChildrenOrder()...)
		return &Domicilio{n}
	}
	e.AddChild(c.Node)
	return c
}

// AddDomicilio agrega atributos al elemento Domicilio y lo devuelve.
func (e *Ubicacion) AddDomicilio(attributes ...Attrs) *Domicilio {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := e.Domicilio()
	c.SetAttrs(attrs)
	return c
}

// Ubicaciones representa el elemento cartaporte31:Ubicaciones.
type Ubicaciones struct{ *nodes.Node }

// NewUbicaciones crea el elemento cartaporte31:Ubicaciones.
func NewUbicaciones(attrs Attrs) *Ubicaciones {
	n := newElement("cartaporte31:Ubicaciones", []string{}, []string{}, attrs)

	return &Ubicaciones{n}
}

// AddUbicacion agrega atributos al elemento Ubicacion y lo devuelve.
func (e *Ubicaciones) AddUbicacion(attributes ...Attrs) *Ubicacion {
	attrs := Attrs{}
	for _, a := range attributes {
		for k, v := range a {
			attrs[k] = v
		}
	}
	c := NewUbicacion(attrs)
	e.AddChild(c.Node)
	return c
}

// MultiUbicacion agrega varios elementos y devuelve el contenedor.
func (e *Ubicaciones) MultiUbicacion(attrs ...Attrs) *Ubicaciones {
	for _, a := range attrs {
		e.AddUbicacion(a)
	}
	return e
}
