package cfdi40validators

import (
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

func conceptosXML(conceptos ...string) string {
	return "<cfdi:Conceptos>" + strings.Join(conceptos, "") + "</cfdi:Conceptos>"
}

func TestConceptoDescuento(t *testing.T) {
	expectCodes(t, runValidator(ConceptoDescuento{}, newComprobante(t, "", "")), "CONCEPDESC01")
	runStatusCases(t, ConceptoDescuento{}, []statusCase{
		{"sin conceptos", "", "", map[string]validate.Status{"CONCEPDESC01": stNone}},
		{"conceptos sin descuento", "", conceptosXML(`<cfdi:Concepto Importe="10"/>`),
			map[string]validate.Status{"CONCEPDESC01": stOk}},
		{"descuentos válidos", "", conceptosXML(`<cfdi:Concepto Importe="10" Descuento="0"/>`, `<cfdi:Concepto Importe="10" Descuento="10.00"/>`),
			map[string]validate.Status{"CONCEPDESC01": stOk}},
		{"descuento mayor al importe", "", conceptosXML(`<cfdi:Concepto Importe="10" Descuento="1"/>`, `<cfdi:Concepto Importe="10" Descuento="10.01"/>`),
			map[string]validate.Status{"CONCEPDESC01": stError}},
		{"descuento negativo", "", conceptosXML(`<cfdi:Concepto Importe="10" Descuento="-0.01"/>`),
			map[string]validate.Status{"CONCEPDESC01": stError}},
	})

	asserts := runValidator(ConceptoDescuento{}, newComprobante(t, "",
		conceptosXML(`<cfdi:Concepto Importe="10"/>`, `<cfdi:Concepto Importe="10" Descuento="11"/>`)))
	if got := asserts.Get("CONCEPDESC01").Explanation; !strings.HasPrefix(got, "Concepto #2,") {
		t.Errorf("explicación inesperada: %q", got)
	}
}

func TestConceptoImpuestos(t *testing.T) {
	codes := []string{"CONCEPIMPC01", "CONCEPIMPC02", "CONCEPIMPC03", "CONCEPIMPC04",
		"CONCEPIMPC05", "CONCEPIMPC06", "CONCEPIMPC07", "CONCEPIMPC08"}
	expectCodes(t, runValidator(ConceptoImpuestos{}, newComprobante(t, "", "")), codes...)

	statuses := func(values ...validate.Status) map[string]validate.Status {
		m := map[string]validate.Status{}
		for i, v := range values {
			m[codes[i]] = v
		}
		return m
	}
	concepto := func(objetoImp, impuestos string) string {
		attr := ""
		if objetoImp != "" {
			attr = ` ObjetoImp="` + objetoImp + `"`
		}
		return `<cfdi:Concepto Importe="100"` + attr + `>` + impuestos + `</cfdi:Concepto>`
	}
	traslados := func(items ...string) string {
		return "<cfdi:Impuestos><cfdi:Traslados>" + strings.Join(items, "") + "</cfdi:Traslados></cfdi:Impuestos>"
	}
	retenciones := func(items ...string) string {
		return "<cfdi:Impuestos><cfdi:Retenciones>" + strings.Join(items, "") + "</cfdi:Retenciones></cfdi:Impuestos>"
	}
	const tasa = `<cfdi:Traslado Base="100" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="16"/>`
	const cuota = `<cfdi:Traslado Base="100" Impuesto="003" TipoFactor="Cuota" TasaOCuota="1.000000" Importe="100"/>`
	const exento = `<cfdi:Traslado Base="100" Impuesto="002" TipoFactor="Exento"/>`
	const retencion = `<cfdi:Retencion Base="100" Impuesto="001" TipoFactor="Tasa" TasaOCuota="0.100000" Importe="10"/>`

	runStatusCases(t, ConceptoImpuestos{}, []statusCase{
		{"sin conceptos", "", "", statuses(stOk, stOk, stOk, stOk, stOk, stOk, stNone, stNone)},
		{"correcto", "", conceptosXML(
			concepto("02", traslados(tasa, cuota, exento)),
			concepto("02", retenciones(retencion)),
			concepto("01", ""),
			concepto("03", ""),
			concepto("04", ""),
		), statuses(stOk, stOk, stOk, stOk, stOk, stOk, stOk, stOk)},
		{"nodo impuestos vacío", "", conceptosXML(concepto("02", `<cfdi:Impuestos/>`)),
			statuses(stError, stOk, stOk, stOk, stOk, stOk, stOk, stNone)},
		{"traslado sin base", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="0"/>`))),
			statuses(stOk, stError, stOk, stOk, stOk, stOk, stOk, stNone)},
		{"traslado con base cero", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Base="0" Impuesto="002" TipoFactor="Exento"/>`))),
			statuses(stOk, stError, stOk, stOk, stOk, stOk, stOk, stNone)},
		{"traslado con base no numérica", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Base="x" Impuesto="002" TipoFactor="Exento"/>`))),
			map[string]validate.Status{"CONCEPIMPC02": stError}},
		{"exento con tasa o cuota", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Base="1" Impuesto="002" TipoFactor="Exento" TasaOCuota="0.000000"/>`))),
			statuses(stOk, stOk, stError, stOk, stOk, stOk, stOk, stNone)},
		{"exento con importe", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Base="1" Impuesto="002" TipoFactor="Exento" Importe=""/>`))),
			statuses(stOk, stOk, stError, stOk, stOk, stOk, stOk, stNone)},
		{"tasa sin tasa o cuota", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Base="1" Impuesto="002" TipoFactor="Tasa" Importe="0.16"/>`))),
			statuses(stOk, stOk, stOk, stError, stOk, stOk, stOk, stNone)},
		{"cuota sin importe", "", conceptosXML(concepto("02", traslados(`<cfdi:Traslado Base="1" Impuesto="003" TipoFactor="Cuota" TasaOCuota="1.000000"/>`))),
			statuses(stOk, stOk, stOk, stError, stOk, stOk, stOk, stNone)},
		{"retención sin base", "", conceptosXML(concepto("02", retenciones(`<cfdi:Retencion Impuesto="001" TipoFactor="Tasa" TasaOCuota="0.100000" Importe="10"/>`))),
			statuses(stOk, stOk, stOk, stOk, stError, stOk, stOk, stNone)},
		{"retención exenta", "", conceptosXML(concepto("02", retenciones(`<cfdi:Retencion Base="1" Impuesto="001" TipoFactor="Exento"/>`))),
			statuses(stOk, stOk, stOk, stOk, stOk, stError, stOk, stNone)},
		{"objeto 02 sin impuestos", "", conceptosXML(concepto("02", traslados(tasa)), concepto("02", "")),
			statuses(stOk, stOk, stOk, stOk, stOk, stOk, stError, stNone)},
		{"objeto 01 con impuestos", "", conceptosXML(concepto("01", traslados(tasa))),
			statuses(stOk, stOk, stOk, stOk, stOk, stOk, stNone, stError)},
		{"objeto 03 con impuestos", "", conceptosXML(concepto("03", traslados(tasa))),
			map[string]validate.Status{"CONCEPIMPC07": stNone, "CONCEPIMPC08": stError}},
		{"objeto 04 con impuestos", "", conceptosXML(concepto("04", retenciones(retencion))),
			map[string]validate.Status{"CONCEPIMPC07": stNone, "CONCEPIMPC08": stError}},
		{"objeto 05 y sin objeto no se validan", "", conceptosXML(concepto("05", traslados(tasa)), concepto("05", ""), concepto("", "")),
			map[string]validate.Status{"CONCEPIMPC07": stNone, "CONCEPIMPC08": stNone}},
	})

	asserts := runValidator(ConceptoImpuestos{}, newComprobante(t, "", conceptosXML(
		concepto("02", traslados(tasa)),
		concepto("02", traslados(tasa, `<cfdi:Traslado Impuesto="002" TipoFactor="Exento"/>`)),
	)))
	if got := asserts.Get("CONCEPIMPC02").Explanation; got != "Concepto #2, Traslado #2" {
		t.Errorf("explicación inesperada de CONCEPIMPC02: %q", got)
	}
}
