package cfdi40

import (
	"github.com/codigoaxaca/go-cfdi-utils/internal/decimal"
	"github.com/codigoaxaca/go-cfdi-utils/sumasconceptos"
)

// WriteSumasConceptos escribe en el comprobante los valores de las sumas:
//
//   - SubTotal, Total y Descuento del comprobante (Descuento solo si algún
//     concepto lo tiene o es mayor a cero).
//   - El nodo Impuestos con TotalImpuestosTrasladados, TotalImpuestosRetenidos,
//     los traslados agrupados (incluyendo Base y los exentos) y las retenciones.
//     Si no hay impuestos el nodo se elimina.
//   - TotaldeRetenciones y TotaldeTraslados del complemento de impuestos locales,
//     si existe.
//
// precision es la cantidad de decimales con que se escriben los importes.
func WriteSumasConceptos(c *Comprobante, sumas *sumasconceptos.SumasConceptos, precision int) {
	format := func(v float64) string { return decimal.Format(v, precision) }

	// sumas del comprobante
	c.SetAttr("SubTotal", format(sumas.SubTotal))
	c.SetAttr("Total", format(sumas.Total))
	c.SetAttr("Descuento", format(sumas.Descuento))
	if !sumas.FoundAnyConceptWithDiscount && !(decimal.Round(sumas.Descuento, precision) > 0) {
		c.RemoveAttr("Descuento")
	}

	// nodo de impuestos
	impuestos := c.Impuestos()
	if !sumas.HasTraslados() && !sumas.HasRetenciones() && !sumas.HasExentos() {
		c.RemoveChild(impuestos.Node)
	} else {
		impuestos.Clear()
		if sumas.HasTraslados() {
			impuestos.SetAttr("TotalImpuestosTrasladados", format(sumas.ImpuestosTrasladados))
			traslados := impuestos.Traslados()
			for _, t := range sumas.Traslados {
				traslados.AddTraslado(Attrs{
					"Base":       format(t.Base),
					"Impuesto":   t.Impuesto,
					"TipoFactor": t.TipoFactor,
					"TasaOCuota": t.TasaOCuota,
					"Importe":    format(t.Importe),
				})
			}
		}
		if sumas.HasExentos() {
			traslados := impuestos.Traslados()
			for _, e := range sumas.Exentos {
				traslados.AddTraslado(Attrs{
					"Base":       format(e.Base),
					"Impuesto":   e.Impuesto,
					"TipoFactor": e.TipoFactor,
				})
			}
		}
		if sumas.HasRetenciones() {
			impuestos.SetAttr("TotalImpuestosRetenidos", format(sumas.ImpuestosRetenidos))
			retenciones := impuestos.Retenciones()
			for _, r := range sumas.Retenciones {
				retenciones.AddRetencion(Attrs{
					"Impuesto": r.Impuesto,
					"Importe":  format(r.Importe),
				})
			}
		}
	}

	// complemento de impuestos locales
	impLocal := c.SearchNode("cfdi:Complemento", "implocal:ImpuestosLocales")
	if impLocal == nil {
		return
	}
	if !sumas.HasLocalesTraslados() && !sumas.HasLocalesRetenciones() {
		complemento := c.Complemento()
		complemento.RemoveChild(impLocal)
		if complemento.Len() == 0 {
			c.RemoveChild(complemento.Node)
		}
		return
	}
	impLocal.SetAttr("TotaldeRetenciones", format(sumas.LocalesImpuestosRetenidos))
	impLocal.SetAttr("TotaldeTraslados", format(sumas.LocalesImpuestosTrasladados))
}
