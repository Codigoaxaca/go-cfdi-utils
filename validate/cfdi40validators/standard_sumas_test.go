package cfdi40validators

import (
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// sumasConceptos son dos conceptos: 1000 con IVA 16% y retención de ISR 10%,
// con descuento de 100, y 500 exento de IVA.
const sumasConceptos = `<cfdi:Conceptos>
	<cfdi:Concepto Importe="1000" Descuento="100" ValorUnitario="1000" ObjetoImp="02">
		<cfdi:Impuestos>
			<cfdi:Traslados><cfdi:Traslado Base="900" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="144"/></cfdi:Traslados>
			<cfdi:Retenciones><cfdi:Retencion Base="900" Impuesto="001" TipoFactor="Tasa" TasaOCuota="0.100000" Importe="90"/></cfdi:Retenciones>
		</cfdi:Impuestos>
	</cfdi:Concepto>
	<cfdi:Concepto Importe="500" ValorUnitario="500" ObjetoImp="02">
		<cfdi:Impuestos>
			<cfdi:Traslados><cfdi:Traslado Base="500" Impuesto="002" TipoFactor="Exento"/></cfdi:Traslados>
		</cfdi:Impuestos>
	</cfdi:Concepto>
</cfdi:Conceptos>`

const (
	sumasTrasladoTasa   = `<cfdi:Traslado Base="900.00" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="144.00"/>`
	sumasTrasladoExento = `<cfdi:Traslado Base="500.00" Impuesto="002" TipoFactor="Exento"/>`
	sumasRetencion      = `<cfdi:Retencion Impuesto="001" Importe="90.00"/>`
	sumasAttrs          = `Moneda="MXN" SubTotal="1500.00" Descuento="100.00" Total="1454.00"`
)

func sumasImpuestos(totales string, traslados []string, retenciones []string) string {
	var b strings.Builder
	b.WriteString("<cfdi:Impuestos " + totales + ">")
	if len(traslados) > 0 {
		b.WriteString("<cfdi:Traslados>" + strings.Join(traslados, "") + "</cfdi:Traslados>")
	}
	if len(retenciones) > 0 {
		b.WriteString("<cfdi:Retenciones>" + strings.Join(retenciones, "") + "</cfdi:Retenciones>")
	}
	b.WriteString("</cfdi:Impuestos>")
	return b.String()
}

func TestSumasConceptosComprobanteImpuestosCodes(t *testing.T) {
	expectCodes(t, runValidator(SumasConceptosComprobanteImpuestos{}, newComprobante(t, "", "")),
		"SUMAS01", "SUMAS02", "SUMAS03", "SUMAS04", "SUMAS05", "SUMAS06", "SUMAS07",
		"SUMAS08", "SUMAS09", "SUMAS10", "SUMAS11", "SUMAS12", "SUMAS13")
}

func TestSumasConceptosComprobanteImpuestos(t *testing.T) {
	const totales = `TotalImpuestosTrasladados="144.00" TotalImpuestosRetenidos="90.00"`
	allOk := map[string]validate.Status{}
	for _, code := range []string{"SUMAS01", "SUMAS02", "SUMAS03", "SUMAS04", "SUMAS05", "SUMAS06", "SUMAS07",
		"SUMAS08", "SUMAS09", "SUMAS10", "SUMAS11", "SUMAS12", "SUMAS13",
		"SUMAS06:001", "SUMAS06:002", "SUMAS10:001"} {
		allOk[code] = stOk
	}
	with := func(changes map[string]validate.Status) map[string]validate.Status {
		m := map[string]validate.Status{}
		for k, v := range allOk {
			m[k] = v
		}
		for k, v := range changes {
			m[k] = v
		}
		return m
	}

	runStatusCases(t, SumasConceptosComprobanteImpuestos{}, []statusCase{
		{"correcto", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{sumasTrasladoTasa, sumasTrasladoExento}, []string{sumasRetencion}),
			allOk},
		{"base del comprobante con 6 decimales", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{
				`<cfdi:Traslado Base="900.000000" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="144.00"/>`,
				`<cfdi:Traslado Base="500.000000" Impuesto="002" TipoFactor="Exento"/>`,
			}, []string{sumasRetencion}),
			allOk},
		{"subtotal, descuento y total incorrectos", `Moneda="MXN" SubTotal="1500.01" Descuento="99" Total="1454.01"`,
			sumasConceptos + sumasImpuestos(totales, []string{sumasTrasladoTasa, sumasTrasladoExento}, []string{sumasRetencion}),
			with(map[string]validate.Status{"SUMAS01": stError, "SUMAS02": stError, "SUMAS03": stError})},
		{"totales de impuestos incorrectos", sumasAttrs,
			sumasConceptos + sumasImpuestos(`TotalImpuestosTrasladados="144.01" TotalImpuestosRetenidos="89.99"`,
				[]string{sumasTrasladoTasa, sumasTrasladoExento}, []string{sumasRetencion}),
			with(map[string]validate.Status{"SUMAS04": stError, "SUMAS08": stError})},
		{"importe de traslado y retención incorrectos", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{
				`<cfdi:Traslado Base="900.00" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="144.01"/>`,
				sumasTrasladoExento,
			}, []string{`<cfdi:Retencion Impuesto="001" Importe="90.01"/>`}),
			with(map[string]validate.Status{"SUMAS06": stError, "SUMAS06:001": stError, "SUMAS10": stError, "SUMAS10:001": stError})},
		{"base de exento incorrecta", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{
				sumasTrasladoTasa,
				`<cfdi:Traslado Base="500.01" Impuesto="002" TipoFactor="Exento"/>`,
			}, []string{sumasRetencion}),
			with(map[string]validate.Status{"SUMAS06": stError, "SUMAS06:002": stError})},
		{"base de traslado incorrecta", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{
				`<cfdi:Traslado Base="900.01" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="144.00"/>`,
				sumasTrasladoExento,
			}, []string{sumasRetencion}),
			with(map[string]validate.Status{"SUMAS13": stError})},
		{"traslado sin base", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{
				`<cfdi:Traslado Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="144.00"/>`,
				sumasTrasladoExento,
			}, []string{sumasRetencion}),
			with(map[string]validate.Status{"SUMAS13": stError})},
		{"faltan exento y retención", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{sumasTrasladoTasa}, nil),
			with(map[string]validate.Status{
				"SUMAS05": stError, "SUMAS06": stError, "SUMAS06:002": stError,
				"SUMAS09": stError, "SUMAS10": stError, "SUMAS10:001": stError,
			})},
		{"faltan todos los traslados", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, nil, []string{sumasRetencion}),
			with(map[string]validate.Status{
				"SUMAS05": stError, "SUMAS06": stError, "SUMAS06:001": stError, "SUMAS06:002": stError, "SUMAS13": stNone,
			})},
		{"impuestos de más", sumasAttrs,
			sumasConceptos + sumasImpuestos(totales, []string{
				sumasTrasladoTasa, sumasTrasladoExento,
				`<cfdi:Traslado Base="1" Impuesto="003" TipoFactor="Tasa" TasaOCuota="0.080000" Importe="0"/>`,
			}, []string{sumasRetencion, `<cfdi:Retencion Impuesto="002" Importe="0"/>`}),
			with(map[string]validate.Status{"SUMAS07": stError, "SUMAS11": stError})},
		{"descuento mayor al subtotal", `Moneda="MXN" SubTotal="10" Descuento="11" Total="0"`, "",
			map[string]validate.Status{"SUMAS01": stError, "SUMAS02": stError, "SUMAS03": stOk, "SUMAS12": stError,
				"SUMAS04": stOk, "SUMAS05": stOk, "SUMAS06": stOk, "SUMAS07": stOk,
				"SUMAS08": stOk, "SUMAS09": stOk, "SUMAS10": stOk, "SUMAS11": stOk, "SUMAS13": stNone}},
		{"pago sin conceptos con impuestos", `Moneda="XXX" SubTotal="0" Total="0"`,
			`<cfdi:Conceptos><cfdi:Concepto Importe="0" ObjetoImp="01"/></cfdi:Conceptos>`,
			map[string]validate.Status{"SUMAS01": stOk, "SUMAS02": stOk, "SUMAS03": stOk, "SUMAS04": stOk, "SUMAS05": stOk,
				"SUMAS06": stOk, "SUMAS07": stOk, "SUMAS08": stOk, "SUMAS09": stOk, "SUMAS10": stOk, "SUMAS11": stOk,
				"SUMAS12": stOk, "SUMAS13": stNone}},
	})
}

func TestSumasConceptosComprobanteImpuestosSoloExentos(t *testing.T) {
	// En CFDI 4.0 un comprobante con solo traslados exentos no registra
	// TotalImpuestosTrasladados pero sí el traslado exento con su base.
	const conceptos = `<cfdi:Conceptos><cfdi:Concepto Importe="500" ObjetoImp="02"><cfdi:Impuestos><cfdi:Traslados>` +
		`<cfdi:Traslado Base="500" Impuesto="002" TipoFactor="Exento"/></cfdi:Traslados></cfdi:Impuestos></cfdi:Concepto></cfdi:Conceptos>`
	c := newComprobante(t, `Moneda="MXN" SubTotal="500" Total="500"`,
		conceptos+sumasImpuestos("", []string{sumasTrasladoExento}, nil))
	asserts := runValidator(SumasConceptosComprobanteImpuestos{}, c)
	if asserts.HasErrors() {
		t.Errorf("no se esperaban errores:\n%s", asserts)
	}
	expectStatuses(t, asserts, map[string]validate.Status{"SUMAS05": stOk, "SUMAS06:001": stOk, "SUMAS07": stOk, "SUMAS13": stNone})
	if title := asserts.Get("SUMAS06:001").Title; !strings.Contains(title, "base") || !strings.Contains(title, "Exento") {
		t.Errorf("título inesperado: %q", title)
	}

	// y el resto de validadores estándar tampoco reportan errores
	all := validate.NewAsserts()
	for _, v := range StandardValidators() {
		v.Validate(t.Context(), c, &validate.Environment{}, all)
	}
	for _, code := range []string{"COMPIMPUESTOSC01", "COMPIMPUESTOSC02", "COMPIMPUESTOSC04"} {
		if a := all.Get(code); a.Status != stOk {
			t.Errorf("%s: se esperaba OK, se obtuvo %s", code, a.Status)
		}
	}
}

func TestSumasConceptosComprobanteImpuestosLocales(t *testing.T) {
	const conceptos = `<cfdi:Conceptos><cfdi:Concepto Importe="100" ObjetoImp="01"/></cfdi:Conceptos>`
	const implocal = `<cfdi:Complemento><implocal:ImpuestosLocales xmlns:implocal="http://www.sat.gob.mx/implocal" version="1.0" TotaldeRetenciones="0" TotaldeTraslados="3.00">` +
		`<implocal:TrasladosLocales ImpLocTrasladado="ISH" TasadeTraslado="3.00" Importe="3.00"/></implocal:ImpuestosLocales></cfdi:Complemento>`
	for total, expected := range map[string]validate.Status{"103.00": stOk, "100.00": stError} {
		c := newComprobante(t, `Moneda="MXN" SubTotal="100" Total="`+total+`"`, conceptos+implocal)
		expectStatuses(t, runValidator(SumasConceptosComprobanteImpuestos{}, c), map[string]validate.Status{"SUMAS03": expected})
	}
}

func TestSumasConceptosComprobanteImpuestosExplanations(t *testing.T) {
	c := newComprobante(t, sumasAttrs, sumasConceptos+sumasImpuestos(
		`TotalImpuestosTrasladados="144.00" TotalImpuestosRetenidos="90.00"`,
		[]string{sumasTrasladoTasa, sumasTrasladoExento, `<cfdi:Traslado Base="1" Impuesto="003" TipoFactor="Tasa" TasaOCuota="0.080000" Importe="0.08"/>`},
		[]string{sumasRetencion}))
	asserts := runValidator(SumasConceptosComprobanteImpuestos{}, c)
	if got := asserts.Get("SUMAS01").Explanation; got != "Calculado: 1500, Comprobante: 1500" {
		t.Errorf("explicación SUMAS01: %q", got)
	}
	if got := asserts.Get("SUMAS07").Explanation; !strings.Contains(got, "No encontrados: 1 impuestos.") || !strings.Contains(got, "Impuesto: 003") {
		t.Errorf("explicación SUMAS07: %q", got)
	}
	if got := asserts.Get("SUMAS11").Explanation; got != "No encontrados: 0 impuestos." {
		t.Errorf("explicación SUMAS11: %q", got)
	}
	if got := asserts.Get("SUMAS06:001").Title; got != "El importe del impuesto Grupo traslado Impuesto 002 Tipo factor Tasa Tasa o cuota 0.160000 es igual al importe del nodo" {
		t.Errorf("título SUMAS06:001: %q", got)
	}
	if got := asserts.Get("SUMAS10:001").Title; got != "El importe del impuesto Grupo retención Impuesto 001 es igual al importe del nodo" {
		t.Errorf("título SUMAS10:001: %q", got)
	}
}
