package cfdi40validators

import (
	"context"
	"fmt"
	"math"
	"regexp"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// ComprobanteDecimalesMoneda valida que los importes del comprobante no
// tengan más decimales de los que soporta la moneda.
//
//   - MONDEC01: El subtotal del comprobante no contiene más de los decimales de la moneda
//   - MONDEC02: El descuento del comprobante no contiene más de los decimales de la moneda
//   - MONDEC03: El total del comprobante no contiene más de los decimales de la moneda
//   - MONDEC04: El total de impuestos trasladados no contiene más de los decimales de la moneda
//   - MONDEC05: El total de impuestos retenidos no contiene más de los decimales de la moneda
//
// Solo se conocen los decimales de MXN, USD y EUR (2) y XXX (0); con otra
// moneda todos los Assert quedan en NONE. MONDEC04 y MONDEC05 quedan en NONE
// si no existe el nodo cfdi:Impuestos.
type ComprobanteDecimalesMoneda struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteDecimalesMoneda) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"MONDEC01", "El subtotal del comprobante no contiene más de los decimales de la moneda"},
		{"MONDEC02", "El descuento del comprobante no contiene más de los decimales de la moneda"},
		{"MONDEC03", "El total del comprobante no contiene más de los decimales de la moneda"},
		{"MONDEC04", "El total de impuestos trasladados no contiene más de los decimales de la moneda"},
		{"MONDEC05", "El total de impuestos retenidos no contiene más de los decimales de la moneda"},
	})
	moneda := c.Attr("Moneda")
	decimals, ok := knownCurrencyDecimals(moneda)
	if !ok {
		asserts.Get("MONDEC01").SetExplanation(fmt.Sprintf("No se conocen los decimales de la moneda %q", moneda))
		return
	}
	check := func(code string, node *nodes.Node, attr string, required bool) {
		value, exists := node.LookupAttr(attr)
		valid := !(required && !exists) && decimalsCount(value) <= decimals
		asserts.PutStatus(code, validate.StatusWhen(valid),
			fmt.Sprintf("Valor: %q, Moneda: \"%s - %d decimales\"", value, moneda, decimals))
	}
	check("MONDEC01", c, "SubTotal", true)
	check("MONDEC02", c, "Descuento", false)
	check("MONDEC03", c, "Total", true)
	if impuestos := c.SearchNode("cfdi:Impuestos"); impuestos != nil {
		check("MONDEC04", impuestos, "TotalImpuestosTrasladados", false)
		check("MONDEC05", impuestos, "TotalImpuestosRetenidos", false)
	}
}

// ---------------------------------------------------------------------------

// ComprobanteDescuento valida el descuento del comprobante.
//
//   - DESCUENTO01: Si existe el atributo descuento, entonces debe ser menor o igual que el subtotal y mayor o igual que cero
type ComprobanteDescuento struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteDescuento) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	a := asserts.Put("DESCUENTO01", "Si existe el atributo descuento, entonces debe ser menor o igual que el subtotal y mayor o igual que cero")
	value, exists := c.LookupAttr("Descuento")
	if !exists {
		return
	}
	descuento, subtotal := toFloat(value), toFloat(c.Attr("SubTotal"))
	a.SetStatus(validate.StatusWhen(value != "" && descuento >= 0 && descuento <= subtotal),
		fmt.Sprintf("Descuento: %q, SubTotal: %q", value, c.Attr("SubTotal")))
}

// ---------------------------------------------------------------------------

// ComprobanteFormaPago valida la forma de pago cuando existe el complemento
// para recepción de pagos (pago20:Pagos en CFDI 4.0; también se reconoce
// pago10:Pagos).
//
//   - FORMAPAGO01: El campo forma de pago no debe existir cuando existe el complemento para recepción de pagos
type ComprobanteFormaPago struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteFormaPago) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	a := asserts.Put("FORMAPAGO01", "El campo forma de pago no debe existir cuando existe el complemento para recepción de pagos")
	if c.SearchNode("cfdi:Complemento", "pago20:Pagos") == nil && c.SearchNode("cfdi:Complemento", "pago10:Pagos") == nil {
		return
	}
	a.SetStatus(validate.StatusWhen(!c.HasAttr("FormaPago")))
}

// ---------------------------------------------------------------------------

// ComprobanteImpuestos valida el nodo cfdi:Impuestos del comprobante.
//
//   - COMPIMPUESTOSC01: Si existe el nodo impuestos entonces debe incluir el total de traslados y/o el total de retenciones, salvo que solo contenga traslados exentos
//   - COMPIMPUESTOSC02: Si existe al menos un traslado que no es exento entonces debe existir el total de traslados
//   - COMPIMPUESTOSC03: Si existe al menos una retención entonces debe existir el total de retenciones
//   - COMPIMPUESTOSC04: Si todos los traslados son exentos entonces no debe existir el total de traslados
//
// En CFDI 4.0 los traslados exentos se registran en el nodo de impuestos del
// comprobante (con Base, sin TasaOCuota ni Importe) y, si todos los
// traslados son exentos, no se registra TotalImpuestosTrasladados.
// Todos quedan en NONE si no existe el nodo cfdi:Impuestos.
type ComprobanteImpuestos struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteImpuestos) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"COMPIMPUESTOSC01", "Si existe el nodo impuestos entonces debe incluir el total de traslados y/o el total de retenciones, salvo que solo contenga traslados exentos"},
		{"COMPIMPUESTOSC02", "Si existe al menos un traslado que no es exento entonces debe existir el total de traslados"},
		{"COMPIMPUESTOSC03", "Si existe al menos una retención entonces debe existir el total de retenciones"},
		{"COMPIMPUESTOSC04", "Si todos los traslados son exentos entonces no debe existir el total de traslados"},
	})
	impuestos := c.SearchNode("cfdi:Impuestos")
	if impuestos == nil {
		return
	}
	existsTotalTrasladados := impuestos.HasAttr("TotalImpuestosTrasladados")
	existsTotalRetenidos := impuestos.HasAttr("TotalImpuestosRetenidos")

	traslados := impuestos.SearchNodes("cfdi:Traslados", "cfdi:Traslado")
	hasNoExentos := false
	for _, t := range traslados {
		if t.Attr("TipoFactor") != "Exento" {
			hasNoExentos = true
			break
		}
	}
	onlyExentos := len(traslados) > 0 && !hasNoExentos
	hasRetenciones := len(impuestos.SearchNodes("cfdi:Retenciones", "cfdi:Retencion")) > 0

	asserts.PutStatus("COMPIMPUESTOSC01", validate.StatusWhen(existsTotalTrasladados || existsTotalRetenidos || onlyExentos))
	asserts.PutStatus("COMPIMPUESTOSC02", validate.StatusWhen(!hasNoExentos || existsTotalTrasladados))
	asserts.PutStatus("COMPIMPUESTOSC03", validate.StatusWhen(!hasRetenciones || existsTotalRetenidos))
	if onlyExentos {
		asserts.PutStatus("COMPIMPUESTOSC04", validate.StatusWhen(!existsTotalTrasladados))
	}
}

// ---------------------------------------------------------------------------

var tipoCambioPattern = regexp.MustCompile(`^\d{1,18}(\.\d{1,6})?$`)

// ComprobanteTipoCambio valida el tipo de cambio según la moneda.
//
//   - TIPOCAMBIO01: La moneda exista y no tenga un valor vacío
//   - TIPOCAMBIO02: Si la moneda es "MXN", entonces el tipo de cambio debe tener el valor "1" o no debe existir
//   - TIPOCAMBIO03: Si la moneda es "XXX", entonces el tipo de cambio no debe existir
//   - TIPOCAMBIO04: Si la moneda no es "MXN" ni "XXX", entonces el tipo de cambio debe seguir el patrón [0-9]{1,18}(.[0-9]{1,6})?
type ComprobanteTipoCambio struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteTipoCambio) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"TIPOCAMBIO01", "La moneda exista y no tenga un valor vacío"},
		{"TIPOCAMBIO02", `Si la moneda es "MXN", entonces el tipo de cambio debe tener el valor "1" o no debe existir`},
		{"TIPOCAMBIO03", `Si la moneda es "XXX", entonces el tipo de cambio no debe existir`},
		{"TIPOCAMBIO04", `Si la moneda no es "MXN" ni "XXX", entonces el tipo de cambio debe seguir el patrón [0-9]{1,18}(.[0-9]{1,6})?`},
	})
	moneda := c.Attr("Moneda")
	tipoCambio, existsTipoCambio := c.LookupAttr("TipoCambio")
	asserts.PutStatus("TIPOCAMBIO01", validate.StatusWhen(moneda != ""))
	if moneda == "" {
		return
	}
	explanation := fmt.Sprintf("Moneda: %q, TipoCambio: %q", moneda, tipoCambio)
	switch moneda {
	case "MXN":
		asserts.PutStatus("TIPOCAMBIO02",
			validate.StatusWhen(!existsTipoCambio || math.Abs(toFloat(tipoCambio)-1) < 0.0000001), explanation)
	case "XXX":
		asserts.PutStatus("TIPOCAMBIO03", validate.StatusWhen(!existsTipoCambio), explanation)
	default:
		asserts.PutStatus("TIPOCAMBIO04", validate.StatusWhen(tipoCambioPattern.MatchString(tipoCambio)), explanation)
	}
}

// ---------------------------------------------------------------------------

// ComprobanteTipoDeComprobante valida reglas según el tipo de comprobante.
//
//   - TIPOCOMP01: Si el tipo de comprobante es T, P ó N, entonces no debe existir las condiciones de pago
//   - TIPOCOMP02: Si el tipo de comprobante es T, P ó N, entonces no debe existir la definición de impuestos
//   - TIPOCOMP03: Si el tipo de comprobante es T ó P, entonces no debe existir la forma de pago
//   - TIPOCOMP04: Si el tipo de comprobante es T ó P, entonces no debe existir el método de pago
//   - TIPOCOMP05: Si el tipo de comprobante es T ó P, entonces no debe existir el descuento del comprobante
//   - TIPOCOMP06: Si el tipo de comprobante es T ó P, entonces no debe existir el descuento de los conceptos
//   - TIPOCOMP07: Si el tipo de comprobante es T ó P, entonces el subtotal debe ser cero
//   - TIPOCOMP08: Si el tipo de comprobante es T ó P entonces el total debe ser cero
//   - TIPOCOMP09: Si el tipo de comprobante es I, E ó N, entonces el valor unitario de todos los conceptos debe ser mayor que cero
//   - TIPOCOMP10: Si el tipo de comprobante es N entonces, la moneda debe ser MXN
type ComprobanteTipoDeComprobante struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteTipoDeComprobante) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"TIPOCOMP01", "Si el tipo de comprobante es T, P ó N, entonces no debe existir las condiciones de pago"},
		{"TIPOCOMP02", "Si el tipo de comprobante es T, P ó N, entonces no debe existir la definición de impuestos"},
		{"TIPOCOMP03", "Si el tipo de comprobante es T ó P, entonces no debe existir la forma de pago"},
		{"TIPOCOMP04", "Si el tipo de comprobante es T ó P, entonces no debe existir el método de pago"},
		{"TIPOCOMP05", "Si el tipo de comprobante es T ó P, entonces no debe existir el descuento del comprobante"},
		{"TIPOCOMP06", "Si el tipo de comprobante es T ó P, entonces no debe existir el descuento de los conceptos"},
		{"TIPOCOMP07", "Si el tipo de comprobante es T ó P, entonces el subtotal debe ser cero"},
		{"TIPOCOMP08", "Si el tipo de comprobante es T ó P entonces el total debe ser cero"},
		{"TIPOCOMP09", "Si el tipo de comprobante es I, E ó N, entonces el valor unitario de todos los conceptos debe ser mayor que cero"},
		{"TIPOCOMP10", "Si el tipo de comprobante es N entonces, la moneda debe ser MXN"},
	})
	tipo := c.Attr("TipoDeComprobante")
	conceptos := c.SearchNodes("cfdi:Conceptos", "cfdi:Concepto")

	if tipo == "T" || tipo == "P" || tipo == "N" {
		asserts.PutStatus("TIPOCOMP01", validate.StatusWhen(!c.HasAttr("CondicionesDePago")))
		asserts.PutStatus("TIPOCOMP02", validate.StatusWhen(c.SearchNode("cfdi:Impuestos") == nil))
	}
	if tipo == "T" || tipo == "P" {
		asserts.PutStatus("TIPOCOMP03", validate.StatusWhen(!c.HasAttr("FormaPago")))
		asserts.PutStatus("TIPOCOMP04", validate.StatusWhen(!c.HasAttr("MetodoPago")))
		asserts.PutStatus("TIPOCOMP05", validate.StatusWhen(!c.HasAttr("Descuento")))
		withoutDescuento := true
		for _, concepto := range conceptos {
			if concepto.HasAttr("Descuento") {
				withoutDescuento = false
				break
			}
		}
		asserts.PutStatus("TIPOCOMP06", validate.StatusWhen(withoutDescuento))
		asserts.PutStatus("TIPOCOMP07", validate.StatusWhen(isZero(c.Attr("SubTotal"))), fmt.Sprintf("SubTotal: %q", c.Attr("SubTotal")))
		asserts.PutStatus("TIPOCOMP08", validate.StatusWhen(isZero(c.Attr("Total"))), fmt.Sprintf("Total: %q", c.Attr("Total")))
	}
	if tipo == "I" || tipo == "E" || tipo == "N" {
		status := validate.StatusOk
		for i, concepto := range conceptos {
			if !isGreaterThanZero(concepto.Attr("ValorUnitario")) {
				status = validate.StatusError
				asserts.Get("TIPOCOMP09").SetExplanation(fmt.Sprintf("Concepto #%d, ValorUnitario: %q", i+1, concepto.Attr("ValorUnitario")))
				break
			}
		}
		asserts.PutStatus("TIPOCOMP09", status)
	}
	if tipo == "N" {
		asserts.PutStatus("TIPOCOMP10", validate.StatusWhen(c.Attr("Moneda") == "MXN"), fmt.Sprintf("Moneda: %q", c.Attr("Moneda")))
	}
}

func isZero(value string) bool {
	f, ok := parseNumeric(value)
	return ok && math.Abs(f) < 0.0000001
}

func isGreaterThanZero(value string) bool {
	f, ok := parseNumeric(value)
	return ok && f > 0.0000001
}

// ---------------------------------------------------------------------------

var totalPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)

// ComprobanteTotal valida el formato del total del comprobante.
//
//   - TOTAL01: El atributo Total existe, no está vacío y cumple con el patrón [0-9]+(.[0-9]+)?
type ComprobanteTotal struct{ version40 }

// Validate implementa validate.Validator.
func (ComprobanteTotal) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	total := c.Attr("Total")
	asserts.Put("TOTAL01", "El atributo Total existe, no está vacío y cumple con el patrón [0-9]+(.[0-9]+)?").
		SetStatus(validate.StatusWhen(totalPattern.MatchString(total)), fmt.Sprintf("Total: %q", total))
}
