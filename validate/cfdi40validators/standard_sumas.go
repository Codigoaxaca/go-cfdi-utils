package cfdi40validators

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/codigoaxaca/go-cfdi-utils/internal/decimal"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/sumasconceptos"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// SumasConceptosComprobanteImpuestos calcula el subtotal, descuento, total e
// impuestos a partir de los conceptos (paquete sumasconceptos) y los compara
// con los valores del comprobante y de su nodo de impuestos.
//
//   - SUMAS01: La suma de los importes de conceptos es igual al subtotal del comprobante
//   - SUMAS02: La suma de los descuentos es igual al descuento del comprobante
//   - SUMAS03: El cálculo del total es igual al total del comprobante
//   - SUMAS04: El cálculo de impuestos trasladados es igual al total de impuestos trasladados
//   - SUMAS05: Todos los impuestos trasladados existen en el comprobante
//   - SUMAS06: Todos los valores de los impuestos trasladados coinciden con el comprobante
//   - SUMAS07: No existen más nodos de impuestos trasladados en el comprobante de los que se han calculado
//   - SUMAS08: El cálculo de impuestos retenidos es igual al total de impuestos retenidos
//   - SUMAS09: Todos los impuestos retenidos existen en el comprobante
//   - SUMAS10: Todos los valores de los impuestos retenidos coinciden con el comprobante
//   - SUMAS11: No existen más nodos de impuestos retenidos en el comprobante de los que se han calculado
//   - SUMAS12: El cálculo del descuento debe ser menor o igual al cálculo del subtotal
//   - SUMAS13: Todas las bases de los impuestos trasladados coinciden con el comprobante
//
// Adicionalmente, para SUMAS06 y SUMAS10 se generan los Assert SUMASxx:yyy,
// donde yyy es el consecutivo del impuesto calculado, que validan que el
// importe (o la base, en el caso de los traslados exentos) del impuesto
// calculado es igual al del nodo del comprobante.
//
// En CFDI 4.0 los traslados exentos se registran en el nodo de impuestos del
// comprobante con su Base, por eso forman parte de SUMAS05 a SUMAS07
// comparando la base. SUMAS13 (nuevo en CFDI 4.0, donde Base es requerido en
// los traslados del comprobante) compara la base de los traslados tasa o
// cuota y queda en NONE si no existe ninguno en el comprobante.
//
// La precisión de las sumas es la de la moneda (2 si no se conoce).
type SumasConceptosComprobanteImpuestos struct{ version40 }

// Validate implementa validate.Validator.
func (SumasConceptosComprobanteImpuestos) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"SUMAS01", "La suma de los importes de conceptos es igual al subtotal del comprobante"},
		{"SUMAS02", "La suma de los descuentos es igual al descuento del comprobante"},
		{"SUMAS03", "El cálculo del total es igual al total del comprobante"},
		{"SUMAS04", "El cálculo de impuestos trasladados es igual al total de impuestos trasladados"},
		{"SUMAS05", "Todos los impuestos trasladados existen en el comprobante"},
		{"SUMAS06", "Todos los valores de los impuestos trasladados coinciden con el comprobante"},
		{"SUMAS07", "No existen más nodos de impuestos trasladados en el comprobante de los que se han calculado"},
		{"SUMAS08", "El cálculo de impuestos retenidos es igual al total de impuestos retenidos"},
		{"SUMAS09", "Todos los impuestos retenidos existen en el comprobante"},
		{"SUMAS10", "Todos los valores de los impuestos retenidos coinciden con el comprobante"},
		{"SUMAS11", "No existen más nodos de impuestos retenidos en el comprobante de los que se han calculado"},
		{"SUMAS12", "El cálculo del descuento debe ser menor o igual al cálculo del subtotal"},
		{"SUMAS13", "Todas las bases de los impuestos trasladados coinciden con el comprobante"},
	})

	precision, ok := knownCurrencyDecimals(c.Attr("Moneda"))
	if !ok {
		precision = 2
	}
	sumas := sumasconceptos.New(c, precision)

	putValues(asserts, "SUMAS01", sumas.SubTotal, toFloat(c.Attr("SubTotal")), "Comprobante")
	putValues(asserts, "SUMAS02", sumas.Descuento, toFloat(c.Attr("Descuento")), "Comprobante")
	putValues(asserts, "SUMAS03", sumas.Total, toFloat(c.Attr("Total")), "Comprobante")
	putValues(asserts, "SUMAS04", sumas.ImpuestosTrasladados, toFloat(c.SearchAttribute("cfdi:Impuestos", "TotalImpuestosTrasladados")), "Comprobante")

	// traslados: tasa o cuota (por importe) y exentos (por base)
	var expectedTraslados []expectedImpuesto
	for _, t := range sumas.Traslados {
		expectedTraslados = append(expectedTraslados, expectedImpuesto{
			key:   sumasconceptos.ImpuestoKey(t.Impuesto, t.TipoFactor, t.TasaOCuota),
			label: fmt.Sprintf("Grupo traslado Impuesto %s Tipo factor %s Tasa o cuota %s", t.Impuesto, t.TipoFactor, t.TasaOCuota),
			value: t.Importe,
			base:  t.Base,
		})
	}
	for _, e := range sumas.Exentos {
		expectedTraslados = append(expectedTraslados, expectedImpuesto{
			key:    sumasconceptos.ImpuestoKey(e.Impuesto, e.TipoFactor, ""),
			label:  fmt.Sprintf("Grupo traslado Impuesto %s Tipo factor %s", e.Impuesto, e.TipoFactor),
			value:  e.Base,
			exento: true,
		})
	}
	foundTraslados := validateImpuestosMatch(asserts, 5, expectedTraslados,
		c.SearchNodes("cfdi:Impuestos", "cfdi:Traslados", "cfdi:Traslado"), precision)

	putValues(asserts, "SUMAS08", sumas.ImpuestosRetenidos, toFloat(c.SearchAttribute("cfdi:Impuestos", "TotalImpuestosRetenidos")), "Comprobante")

	var expectedRetenciones []expectedImpuesto
	for _, r := range sumas.Retenciones {
		expectedRetenciones = append(expectedRetenciones, expectedImpuesto{
			key:   sumasconceptos.ImpuestoKey(r.Impuesto, "", ""),
			label: fmt.Sprintf("Grupo retención Impuesto %s", r.Impuesto),
			value: r.Importe,
		})
	}
	validateImpuestosMatch(asserts, 9, expectedRetenciones,
		c.SearchNodes("cfdi:Impuestos", "cfdi:Retenciones", "cfdi:Retencion"), precision)

	// SUMAS12
	subtotal, descuento := toFloat(c.Attr("SubTotal")), toFloat(c.Attr("Descuento"))
	asserts.PutStatus("SUMAS12", validate.StatusWhen(subtotal >= descuento),
		fmt.Sprintf("SubTotal: %s, Descuento: %s", c.Attr("SubTotal"), c.Attr("Descuento")))

	// SUMAS13
	var mismatches []string
	checked := 0
	for _, expected := range expectedTraslados {
		node, ok := foundTraslados[expected.key]
		if expected.exento || !ok {
			continue
		}
		checked++
		found := toFloat(node.Attr("Base"))
		if !decimal.Equals(expected.base, found, precision) {
			mismatches = append(mismatches, fmt.Sprintf("%s: Calculado %s, Encontrado %q",
				expected.label, formatFloat(expected.base), node.Attr("Base")))
		}
	}
	if checked > 0 {
		explanation := fmt.Sprintf("Revisados %d traslados", checked)
		if len(mismatches) > 0 {
			explanation = strings.Join(mismatches, "; ")
		}
		asserts.PutStatus("SUMAS13", validate.StatusWhen(len(mismatches) == 0), explanation)
	}
}

type expectedImpuesto struct {
	key    string
	label  string
	value  float64 // importe o, si es exento, la base
	base   float64
	exento bool
}

// validateImpuestosMatch compara los impuestos calculados con los nodos del
// comprobante y establece SUMAS{offset}, SUMAS{offset+1} (con sus
// SUMAS{offset+1}:yyy) y SUMAS{offset+2}. Devuelve los nodos del comprobante
// por llave.
func validateImpuestosMatch(asserts *validate.Asserts, offset int, expected []expectedImpuesto, extracted []*nodes.Node, precision int) map[string]*nodes.Node {
	byKey := map[string]*nodes.Node{}
	var keys []string
	for _, node := range extracted {
		key := sumasconceptos.ImpuestoKey(node.Attr("Impuesto"), node.Attr("TipoFactor"), node.Attr("TasaOCuota"))
		if _, ok := byKey[key]; !ok {
			keys = append(keys, key)
		}
		byKey[key] = node // como en PHP, el último con la misma llave prevalece
	}

	found := map[string]bool{}
	allFound, allMatch := true, true
	for i, item := range expected {
		code := fmt.Sprintf("SUMAS%02d:%03d", offset+1, i+1)
		node, ok := byKey[item.key]
		var compare float64
		if ok {
			found[item.key] = true
			if item.exento {
				compare = toFloat(node.Attr("Base"))
			} else {
				compare = toFloat(node.Attr("Importe"))
			}
		} else {
			allFound = false
		}
		var match bool
		if item.exento {
			asserts.Put(code, fmt.Sprintf("La base del impuesto %s es igual a la base del nodo", item.label))
			match = ok && decimal.Equals(item.value, compare, precision)
		} else {
			asserts.Put(code, fmt.Sprintf("El importe del impuesto %s es igual al importe del nodo", item.label))
			match = importesAreEqual(item.value, compare)
		}
		asserts.PutStatus(code, validate.StatusWhen(match),
			fmt.Sprintf("Calculado: %s, Encontrado: %s", formatFloat(item.value), formatFloat(compare)))
		allMatch = allMatch && match
	}

	var notFound []string
	for _, key := range keys {
		if found[key] {
			continue
		}
		node := byKey[key]
		var parts []string
		for _, attr := range []string{"Impuesto", "TipoFactor", "TasaOCuota", "Base", "Importe"} {
			if v, exists := node.LookupAttr(attr); exists {
				parts = append(parts, fmt.Sprintf("%s: %s", attr, v))
			}
		}
		notFound = append(notFound, strings.Join(parts, ", "))
	}

	asserts.PutStatus(fmt.Sprintf("SUMAS%02d", offset), validate.StatusWhen(allFound))
	asserts.PutStatus(fmt.Sprintf("SUMAS%02d", offset+1), validate.StatusWhen(allMatch))
	explanation := "No encontrados: 0 impuestos."
	if len(notFound) > 0 {
		explanation = fmt.Sprintf("No encontrados: %d impuestos. %s.", len(notFound), strings.Join(notFound, "; "))
	}
	asserts.PutStatus(fmt.Sprintf("SUMAS%02d", offset+2), validate.StatusWhen(len(notFound) == 0), explanation)
	return byKey
}

func putValues(asserts *validate.Asserts, code string, expected, compare float64, compareLabel string) {
	asserts.PutStatus(code, validate.StatusWhen(importesAreEqual(expected, compare)),
		fmt.Sprintf("Calculado: %s, %s: %s", formatFloat(expected), compareLabel, formatFloat(compare)))
}

func importesAreEqual(a, b float64) bool {
	return math.Abs(a-b) <= 0.000001
}
