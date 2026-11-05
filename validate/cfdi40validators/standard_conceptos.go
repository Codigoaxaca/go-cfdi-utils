package cfdi40validators

import (
	"context"
	"fmt"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// ConceptoDescuento valida el descuento de los conceptos.
//
//   - CONCEPDESC01: Si existe el atributo descuento en el concepto, entonces debe ser menor o igual que el importe y mayor o igual que cero
//
// Queda en NONE si no hay conceptos. La explicación señala el primer
// concepto con error (numerado desde 1).
type ConceptoDescuento struct{ version40 }

// Validate implementa validate.Validator.
func (ConceptoDescuento) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	a := asserts.Put("CONCEPDESC01", "Si existe el atributo descuento en el concepto, entonces debe ser menor o igual que el importe y mayor o igual que cero")
	conceptos := c.SearchNodes("cfdi:Conceptos", "cfdi:Concepto")
	for i, concepto := range conceptos {
		value, exists := concepto.LookupAttr("Descuento")
		if !exists {
			continue
		}
		descuento, importe := toFloat(value), toFloat(concepto.Attr("Importe"))
		if !(descuento >= 0 && descuento <= importe) {
			a.SetStatus(validate.StatusError, fmt.Sprintf("Concepto #%d, Descuento: %q, Importe: %q", i+1, value, concepto.Attr("Importe")))
			return
		}
	}
	if len(conceptos) > 0 {
		a.SetStatus(validate.StatusOk, fmt.Sprintf("Revisados %d conceptos", len(conceptos)))
	}
}

// ---------------------------------------------------------------------------

// ConceptoImpuestos valida los impuestos de los conceptos.
//
//   - CONCEPIMPC01: El nodo impuestos de un concepto debe incluir traslados y/o retenciones
//   - CONCEPIMPC02: Los traslados de los impuestos de un concepto deben tener una base y ser mayor a cero
//   - CONCEPIMPC03: No se debe registrar la tasa o cuota ni el importe cuando el tipo de factor de traslado es exento
//   - CONCEPIMPC04: Se debe registrar la tasa o cuota y el importe cuando el tipo de factor de traslado es tasa o cuota
//   - CONCEPIMPC05: Las retenciones de los impuestos de un concepto deben tener una base y ser mayor a cero
//   - CONCEPIMPC06: Las retenciones de los impuestos de un concepto deben tener un tipo de factor diferente de exento
//   - CONCEPIMPC07: Si el concepto es objeto de impuesto con desglose (ObjetoImp 02) entonces debe existir el nodo impuestos del concepto
//   - CONCEPIMPC08: Si el concepto no es objeto de impuesto, no está obligado al desglose o no causa impuesto (ObjetoImp 01, 03 o 04) entonces no debe existir el nodo impuestos del concepto
//
// Igual que en la versión PHP, CONCEPIMPC01 a CONCEPIMPC06 quedan en OK si no
// hay elementos que los incumplan. CONCEPIMPC07 y CONCEPIMPC08 (nuevas para
// CFDI 4.0, atributo ObjetoImp del catálogo c_ObjetoImp) quedan en NONE si
// ningún concepto tiene los valores de ObjetoImp correspondientes; las claves
// 05 en adelante no se validan.
type ConceptoImpuestos struct{ version40 }

// Validate implementa validate.Validator.
func (ConceptoImpuestos) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"CONCEPIMPC01", "El nodo impuestos de un concepto debe incluir traslados y/o retenciones"},
		{"CONCEPIMPC02", "Los traslados de los impuestos de un concepto deben tener una base y ser mayor a cero"},
		{"CONCEPIMPC03", "No se debe registrar la tasa o cuota ni el importe cuando el tipo de factor de traslado es exento"},
		{"CONCEPIMPC04", "Se debe registrar la tasa o cuota y el importe cuando el tipo de factor de traslado es tasa o cuota"},
		{"CONCEPIMPC05", "Las retenciones de los impuestos de un concepto deben tener una base y ser mayor a cero"},
		{"CONCEPIMPC06", "Las retenciones de los impuestos de un concepto deben tener un tipo de factor diferente de exento"},
		{"CONCEPIMPC07", "Si el concepto es objeto de impuesto con desglose (ObjetoImp 02) entonces debe existir el nodo impuestos del concepto"},
		{"CONCEPIMPC08", "Si el concepto no es objeto de impuesto, no está obligado al desglose o no causa impuesto (ObjetoImp 01, 03 o 04) entonces no debe existir el nodo impuestos del concepto"},
	})

	codes := []string{"CONCEPIMPC01", "CONCEPIMPC02", "CONCEPIMPC03", "CONCEPIMPC04", "CONCEPIMPC05", "CONCEPIMPC06", "CONCEPIMPC07", "CONCEPIMPC08"}
	statuses := map[string]validate.Status{}
	for _, code := range codes[:6] {
		statuses[code] = validate.StatusOk
	}
	fail := func(code, explanation string) {
		if statuses[code] == validate.StatusError {
			return
		}
		statuses[code] = validate.StatusError
		asserts.Get(code).SetExplanation(explanation)
	}
	check := func(code string) {
		if _, ok := statuses[code]; !ok {
			statuses[code] = validate.StatusOk
		}
	}

	for i, concepto := range c.SearchNodes("cfdi:Conceptos", "cfdi:Concepto") {
		n := i + 1
		impuestos := concepto.SearchNode("cfdi:Impuestos")
		traslados := concepto.SearchNodes("cfdi:Impuestos", "cfdi:Traslados", "cfdi:Traslado")
		retenciones := concepto.SearchNodes("cfdi:Impuestos", "cfdi:Retenciones", "cfdi:Retencion")

		if impuestos != nil && len(traslados) == 0 && len(retenciones) == 0 {
			fail("CONCEPIMPC01", fmt.Sprintf("Concepto #%d", n))
		}
		for k, traslado := range traslados {
			where := fmt.Sprintf("Concepto #%d, Traslado #%d", n, k+1)
			if !impuestoHasBaseGreaterThanZero(traslado) {
				fail("CONCEPIMPC02", where)
			}
			switch traslado.Attr("TipoFactor") {
			case "Exento":
				if traslado.HasAttr("TasaOCuota") || traslado.HasAttr("Importe") {
					fail("CONCEPIMPC03", where)
				}
			case "Tasa", "Cuota":
				if traslado.Attr("TasaOCuota") == "" || traslado.Attr("Importe") == "" {
					fail("CONCEPIMPC04", where)
				}
			}
		}
		for k, retencion := range retenciones {
			where := fmt.Sprintf("Concepto #%d, Retención #%d", n, k+1)
			if !impuestoHasBaseGreaterThanZero(retencion) {
				fail("CONCEPIMPC05", where)
			}
			if retencion.Attr("TipoFactor") == "Exento" {
				fail("CONCEPIMPC06", where)
			}
		}

		objetoImp := concepto.Attr("ObjetoImp")
		switch objetoImp {
		case "02":
			check("CONCEPIMPC07")
			if impuestos == nil {
				fail("CONCEPIMPC07", fmt.Sprintf("Concepto #%d, ObjetoImp: %q", n, objetoImp))
			}
		case "01", "03", "04":
			check("CONCEPIMPC08")
			if impuestos != nil {
				fail("CONCEPIMPC08", fmt.Sprintf("Concepto #%d, ObjetoImp: %q", n, objetoImp))
			}
		}
	}

	for _, code := range codes {
		if status, ok := statuses[code]; ok {
			asserts.PutStatus(code, status)
		}
	}
}

func impuestoHasBaseGreaterThanZero(impuesto *nodes.Node) bool {
	base, ok := parseNumeric(impuesto.Attr("Base"))
	return ok && base >= 0.000001
}
