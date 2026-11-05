// Package sumasconceptos calcula los totales de un comprobante con base en
// sus conceptos: subtotal, descuento, impuestos trasladados y retenidos
// agrupados (por Impuesto, TipoFactor y TasaOCuota), impuestos exentos,
// impuestos locales y total.
//
// Normalmente no se usa de forma directa sino a través de
// cfdi40.Creator.AddSumasConceptos, pero también sirve para comprobar los
// totales de un CFDI recibido.
package sumasconceptos

import (
	"strings"

	"github.com/codigoaxaca/go-cfdi-utils/internal/decimal"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

// Traslado es la agrupación de impuestos trasladados con el mismo
// Impuesto, TipoFactor y TasaOCuota.
type Traslado struct {
	Impuesto   string
	TipoFactor string
	TasaOCuota string
	Base       float64
	Importe    float64
}

// Exento es la agrupación de impuestos trasladados con TipoFactor Exento.
type Exento struct {
	Impuesto   string
	TipoFactor string
	Base       float64
}

// Retencion es la agrupación de impuestos retenidos con el mismo Impuesto.
type Retencion struct {
	Impuesto string
	Importe  float64
}

// ImpuestoLocal es un impuesto del complemento de impuestos locales (implocal).
type ImpuestoLocal struct {
	Impuesto string
	Tasa     float64
	Importe  float64
}

// SumasConceptos contiene el resultado de las sumas.
type SumasConceptos struct {
	Precision int

	SubTotal  float64
	Descuento float64
	Total     float64

	ImpuestosTrasladados float64
	ImpuestosRetenidos   float64
	Traslados            []Traslado
	Exentos              []Exento
	Retenciones          []Retencion

	LocalesImpuestosTrasladados float64
	LocalesImpuestosRetenidos   float64
	LocalesTraslados            []ImpuestoLocal
	LocalesRetenciones          []ImpuestoLocal

	// FoundAnyConceptWithDiscount indica si algún concepto tiene el atributo Descuento.
	FoundAnyConceptWithDiscount bool
}

// New calcula las sumas de los conceptos del comprobante (nodo cfdi:Comprobante)
// redondeando a la precisión dada (normalmente 2, los decimales de la moneda).
//
// Funciona con CFDI 3.3 y 4.0 porque la estructura de conceptos es la misma.
func New(comprobante nodes.Element, precision int) *SumasConceptos {
	s := &SumasConceptos{Precision: precision}
	root := comprobante.AsNode()
	traslados := map[string]int{}
	exentos := map[string]int{}
	retenciones := map[string]int{}

	for _, concepto := range root.SearchNodes("cfdi:Conceptos", "cfdi:Concepto") {
		s.SubTotal += decimal.Parse(concepto.Attr("Importe"))
		if concepto.HasAttr("Descuento") {
			s.FoundAnyConceptWithDiscount = true
		}
		s.Descuento += decimal.Parse(concepto.Attr("Descuento"))

		for _, t := range concepto.SearchNodes("cfdi:Impuestos", "cfdi:Traslados", "cfdi:Traslado") {
			if t.Attr("TipoFactor") == "Exento" {
				key := ImpuestoKey(t.Attr("Impuesto"), t.Attr("TipoFactor"), "")
				i, ok := exentos[key]
				if !ok {
					i = len(s.Exentos)
					exentos[key] = i
					s.Exentos = append(s.Exentos, Exento{Impuesto: t.Attr("Impuesto"), TipoFactor: t.Attr("TipoFactor")})
				}
				s.Exentos[i].Base += decimal.Parse(t.Attr("Base"))
				continue
			}
			key := ImpuestoKey(t.Attr("Impuesto"), t.Attr("TipoFactor"), t.Attr("TasaOCuota"))
			i, ok := traslados[key]
			if !ok {
				i = len(s.Traslados)
				traslados[key] = i
				s.Traslados = append(s.Traslados, Traslado{
					Impuesto:   t.Attr("Impuesto"),
					TipoFactor: t.Attr("TipoFactor"),
					TasaOCuota: t.Attr("TasaOCuota"),
				})
			}
			s.Traslados[i].Importe += decimal.Parse(t.Attr("Importe"))
			s.Traslados[i].Base += decimal.Parse(t.Attr("Base"))
		}

		for _, r := range concepto.SearchNodes("cfdi:Impuestos", "cfdi:Retenciones", "cfdi:Retencion") {
			key := ImpuestoKey(r.Attr("Impuesto"), "", "")
			i, ok := retenciones[key]
			if !ok {
				i = len(s.Retenciones)
				retenciones[key] = i
				s.Retenciones = append(s.Retenciones, Retencion{Impuesto: r.Attr("Impuesto")})
			}
			s.Retenciones[i].Importe += decimal.Parse(r.Attr("Importe"))
		}
	}

	s.LocalesTraslados = impuestosLocales(root, "TrasladosLocales", "Traslado")
	for _, l := range s.LocalesTraslados {
		s.LocalesImpuestosTrasladados += l.Importe
	}
	s.LocalesRetenciones = impuestosLocales(root, "RetencionesLocales", "Retenido")
	for _, l := range s.LocalesRetenciones {
		s.LocalesImpuestosRetenidos += l.Importe
	}

	for i := range s.Traslados {
		s.Traslados[i].Importe = decimal.Round(s.Traslados[i].Importe, precision)
		s.Traslados[i].Base = decimal.Round(s.Traslados[i].Base, precision)
		s.ImpuestosTrasladados += s.Traslados[i].Importe
	}
	for i := range s.Exentos {
		s.Exentos[i].Base = decimal.Round(s.Exentos[i].Base, precision)
	}
	for i := range s.Retenciones {
		s.Retenciones[i].Importe = decimal.Round(s.Retenciones[i].Importe, precision)
		s.ImpuestosRetenidos += s.Retenciones[i].Importe
	}

	s.ImpuestosTrasladados = decimal.Round(s.ImpuestosTrasladados, precision)
	s.ImpuestosRetenidos = decimal.Round(s.ImpuestosRetenidos, precision)
	s.SubTotal = decimal.Round(s.SubTotal, precision)
	s.Descuento = decimal.Round(s.Descuento, precision)
	s.Total = decimal.Round(
		s.SubTotal-s.Descuento+
			s.ImpuestosTrasladados-s.ImpuestosRetenidos+
			s.LocalesImpuestosTrasladados-s.LocalesImpuestosRetenidos,
		precision,
	)
	return s
}

// ImpuestoKey genera la llave de agrupación de un impuesto.
func ImpuestoKey(impuesto, tipoFactor, tasaOCuota string) string {
	return strings.Join([]string{impuesto, tipoFactor, tasaOCuota}, ":")
}

// HasTraslados informa si existen impuestos trasladados (no exentos).
func (s *SumasConceptos) HasTraslados() bool { return len(s.Traslados) > 0 }

// HasExentos informa si existen impuestos trasladados exentos.
func (s *SumasConceptos) HasExentos() bool { return len(s.Exentos) > 0 }

// HasRetenciones informa si existen impuestos retenidos.
func (s *SumasConceptos) HasRetenciones() bool { return len(s.Retenciones) > 0 }

// HasLocalesTraslados informa si existen impuestos locales trasladados.
func (s *SumasConceptos) HasLocalesTraslados() bool { return len(s.LocalesTraslados) > 0 }

// HasLocalesRetenciones informa si existen impuestos locales retenidos.
func (s *SumasConceptos) HasLocalesRetenciones() bool { return len(s.LocalesRetenciones) > 0 }

// Traslado devuelve la agrupación de traslado con la llave dada.
func (s *SumasConceptos) Traslado(impuesto, tipoFactor, tasaOCuota string) (Traslado, bool) {
	key := ImpuestoKey(impuesto, tipoFactor, tasaOCuota)
	for _, t := range s.Traslados {
		if ImpuestoKey(t.Impuesto, t.TipoFactor, t.TasaOCuota) == key {
			return t, true
		}
	}
	return Traslado{}, false
}

// Retencion devuelve la agrupación de retención del impuesto dado.
func (s *SumasConceptos) Retencion(impuesto string) (Retencion, bool) {
	for _, r := range s.Retenciones {
		if r.Impuesto == impuesto {
			return r, true
		}
	}
	return Retencion{}, false
}

func impuestosLocales(root *nodes.Node, plural, singular string) []ImpuestoLocal {
	var out []ImpuestoLocal
	for _, l := range root.SearchNodes("cfdi:Complemento", "implocal:ImpuestosLocales", "implocal:"+plural) {
		out = append(out, ImpuestoLocal{
			Impuesto: l.Attr("ImpLoc" + singular),
			Tasa:     decimal.Parse(l.Attr("Tasade" + singular)),
			Importe:  decimal.Parse(l.Attr("Importe")),
		})
	}
	return out
}
