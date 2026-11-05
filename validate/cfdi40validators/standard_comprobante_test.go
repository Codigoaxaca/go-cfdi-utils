package cfdi40validators

import (
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

type statusCase struct {
	name     string
	attrs    string
	inner    string
	expected map[string]validate.Status
}

func runStatusCases(t *testing.T, v validate.Validator, cases []statusCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			asserts := runValidator(v, newComprobante(t, tc.attrs, tc.inner))
			expectStatuses(t, asserts, tc.expected)
		})
	}
}

func TestComprobanteDecimalesMoneda(t *testing.T) {
	expectCodes(t, runValidator(ComprobanteDecimalesMoneda{}, newComprobante(t, `Moneda="MXN"`, "")),
		"MONDEC01", "MONDEC02", "MONDEC03", "MONDEC04", "MONDEC05")

	impuestos := func(tras, ret string) string {
		return `<cfdi:Impuestos TotalImpuestosTrasladados="` + tras + `" TotalImpuestosRetenidos="` + ret + `"/>`
	}
	runStatusCases(t, ComprobanteDecimalesMoneda{}, []statusCase{
		{"mxn correcto", `Moneda="MXN" SubTotal="100.00" Descuento="1.5" Total="116"`, impuestos("16.00", "1.00"),
			map[string]validate.Status{"MONDEC01": stOk, "MONDEC02": stOk, "MONDEC03": stOk, "MONDEC04": stOk, "MONDEC05": stOk}},
		{"mxn exceso de decimales", `Moneda="MXN" SubTotal="100.001" Descuento="1.555" Total="116.0001"`, impuestos("16.001", "1.001"),
			map[string]validate.Status{"MONDEC01": stError, "MONDEC02": stError, "MONDEC03": stError, "MONDEC04": stError, "MONDEC05": stError}},
		{"usd y eur 2 decimales", `Moneda="USD" SubTotal="1.12" Total="1.123"`, "",
			map[string]validate.Status{"MONDEC01": stOk, "MONDEC02": stOk, "MONDEC03": stError, "MONDEC04": stNone, "MONDEC05": stNone}},
		{"eur", `Moneda="EUR" SubTotal="1.12" Total="1.12"`, "",
			map[string]validate.Status{"MONDEC01": stOk, "MONDEC03": stOk}},
		{"xxx sin decimales", `Moneda="XXX" SubTotal="0" Total="0.0"`, "",
			map[string]validate.Status{"MONDEC01": stOk, "MONDEC03": stError}},
		{"subtotal y total requeridos", `Moneda="MXN"`, "",
			map[string]validate.Status{"MONDEC01": stError, "MONDEC02": stOk, "MONDEC03": stError}},
		{"moneda desconocida", `Moneda="JPY" SubTotal="1.123456" Total="1"`, impuestos("1.123", "1"),
			map[string]validate.Status{"MONDEC01": stNone, "MONDEC02": stNone, "MONDEC03": stNone, "MONDEC04": stNone, "MONDEC05": stNone}},
	})
}

func TestComprobanteDescuento(t *testing.T) {
	expectCodes(t, runValidator(ComprobanteDescuento{}, newComprobante(t, "", "")), "DESCUENTO01")
	runStatusCases(t, ComprobanteDescuento{}, []statusCase{
		{"sin descuento", `SubTotal="100"`, "", map[string]validate.Status{"DESCUENTO01": stNone}},
		{"cero", `SubTotal="100" Descuento="0"`, "", map[string]validate.Status{"DESCUENTO01": stOk}},
		{"igual al subtotal", `SubTotal="100" Descuento="100.00"`, "", map[string]validate.Status{"DESCUENTO01": stOk}},
		{"mayor al subtotal", `SubTotal="100" Descuento="100.01"`, "", map[string]validate.Status{"DESCUENTO01": stError}},
		{"negativo", `SubTotal="100" Descuento="-1"`, "", map[string]validate.Status{"DESCUENTO01": stError}},
		{"vacío", `SubTotal="100" Descuento=""`, "", map[string]validate.Status{"DESCUENTO01": stError}},
	})
}

func TestComprobanteFormaPago(t *testing.T) {
	const pagos20 = `<cfdi:Complemento><pago20:Pagos xmlns:pago20="http://www.sat.gob.mx/Pagos20" Version="2.0"/></cfdi:Complemento>`
	const pagos10 = `<cfdi:Complemento><pago10:Pagos xmlns:pago10="http://www.sat.gob.mx/Pagos" Version="1.0"/></cfdi:Complemento>`
	expectCodes(t, runValidator(ComprobanteFormaPago{}, newComprobante(t, "", "")), "FORMAPAGO01")
	runStatusCases(t, ComprobanteFormaPago{}, []statusCase{
		{"sin complemento de pagos", `FormaPago="01"`, "", map[string]validate.Status{"FORMAPAGO01": stNone}},
		{"pagos 2.0 sin forma de pago", ``, pagos20, map[string]validate.Status{"FORMAPAGO01": stOk}},
		{"pagos 2.0 con forma de pago", `FormaPago="01"`, pagos20, map[string]validate.Status{"FORMAPAGO01": stError}},
		{"pagos 2.0 con forma de pago vacía", `FormaPago=""`, pagos20, map[string]validate.Status{"FORMAPAGO01": stError}},
		{"pagos 1.0 con forma de pago", `FormaPago="01"`, pagos10, map[string]validate.Status{"FORMAPAGO01": stError}},
	})
}

func TestComprobanteImpuestos(t *testing.T) {
	expectCodes(t, runValidator(ComprobanteImpuestos{}, newComprobante(t, "", "")),
		"COMPIMPUESTOSC01", "COMPIMPUESTOSC02", "COMPIMPUESTOSC03", "COMPIMPUESTOSC04")

	const tasa = `<cfdi:Traslado Base="100" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="16"/>`
	const exento = `<cfdi:Traslado Base="100" Impuesto="002" TipoFactor="Exento"/>`
	const retencion = `<cfdi:Retenciones><cfdi:Retencion Impuesto="001" Importe="10"/></cfdi:Retenciones>`
	all := func(s01, s02, s03, s04 validate.Status) map[string]validate.Status {
		return map[string]validate.Status{"COMPIMPUESTOSC01": s01, "COMPIMPUESTOSC02": s02, "COMPIMPUESTOSC03": s03, "COMPIMPUESTOSC04": s04}
	}
	runStatusCases(t, ComprobanteImpuestos{}, []statusCase{
		{"sin nodo impuestos", "", "", all(stNone, stNone, stNone, stNone)},
		{"nodo vacío", "", `<cfdi:Impuestos/>`, all(stError, stOk, stOk, stNone)},
		{"traslados con total", "",
			`<cfdi:Impuestos TotalImpuestosTrasladados="16"><cfdi:Traslados>` + tasa + `</cfdi:Traslados></cfdi:Impuestos>`,
			all(stOk, stOk, stOk, stNone)},
		{"traslados sin total", "",
			`<cfdi:Impuestos><cfdi:Traslados>` + tasa + `</cfdi:Traslados></cfdi:Impuestos>`,
			all(stError, stError, stOk, stNone)},
		{"retenciones con total", "",
			`<cfdi:Impuestos TotalImpuestosRetenidos="10">` + retencion + `</cfdi:Impuestos>`,
			all(stOk, stOk, stOk, stNone)},
		{"retenciones sin total", "",
			`<cfdi:Impuestos TotalImpuestosTrasladados="0">` + retencion + `</cfdi:Impuestos>`,
			all(stOk, stOk, stError, stNone)},
		{"solo exentos sin total", "",
			`<cfdi:Impuestos><cfdi:Traslados>` + exento + `</cfdi:Traslados></cfdi:Impuestos>`,
			all(stOk, stOk, stOk, stOk)},
		{"solo exentos con total", "",
			`<cfdi:Impuestos TotalImpuestosTrasladados="0.00"><cfdi:Traslados>` + exento + `</cfdi:Traslados></cfdi:Impuestos>`,
			all(stOk, stOk, stOk, stError)},
		{"exentos y tasa sin total", "",
			`<cfdi:Impuestos><cfdi:Traslados>` + exento + tasa + `</cfdi:Traslados></cfdi:Impuestos>`,
			all(stError, stError, stOk, stNone)},
		{"exentos y tasa con total", "",
			`<cfdi:Impuestos TotalImpuestosTrasladados="16"><cfdi:Traslados>` + exento + tasa + `</cfdi:Traslados></cfdi:Impuestos>`,
			all(stOk, stOk, stOk, stNone)},
	})
}

func TestComprobanteTipoCambio(t *testing.T) {
	expectCodes(t, runValidator(ComprobanteTipoCambio{}, newComprobante(t, "", "")),
		"TIPOCAMBIO01", "TIPOCAMBIO02", "TIPOCAMBIO03", "TIPOCAMBIO04")
	all := func(s01, s02, s03, s04 validate.Status) map[string]validate.Status {
		return map[string]validate.Status{"TIPOCAMBIO01": s01, "TIPOCAMBIO02": s02, "TIPOCAMBIO03": s03, "TIPOCAMBIO04": s04}
	}
	runStatusCases(t, ComprobanteTipoCambio{}, []statusCase{
		{"sin moneda", ``, "", all(stError, stNone, stNone, stNone)},
		{"moneda vacía", `Moneda=""`, "", all(stError, stNone, stNone, stNone)},
		{"mxn sin tipo de cambio", `Moneda="MXN"`, "", all(stOk, stOk, stNone, stNone)},
		{"mxn con 1", `Moneda="MXN" TipoCambio="1"`, "", all(stOk, stOk, stNone, stNone)},
		{"mxn con 1.000000", `Moneda="MXN" TipoCambio="1.000000"`, "", all(stOk, stOk, stNone, stNone)},
		{"mxn con 1.5", `Moneda="MXN" TipoCambio="1.5"`, "", all(stOk, stError, stNone, stNone)},
		{"mxn vacío", `Moneda="MXN" TipoCambio=""`, "", all(stOk, stError, stNone, stNone)},
		{"xxx sin tipo de cambio", `Moneda="XXX"`, "", all(stOk, stNone, stOk, stNone)},
		{"xxx con tipo de cambio", `Moneda="XXX" TipoCambio="1"`, "", all(stOk, stNone, stError, stNone)},
		{"usd correcto", `Moneda="USD" TipoCambio="18.9008"`, "", all(stOk, stNone, stNone, stOk)},
		{"usd entero", `Moneda="USD" TipoCambio="18"`, "", all(stOk, stNone, stNone, stOk)},
		{"usd siete decimales", `Moneda="USD" TipoCambio="18.1234567"`, "", all(stOk, stNone, stNone, stError)},
		{"usd sin tipo de cambio", `Moneda="USD"`, "", all(stOk, stNone, stNone, stError)},
		{"usd inválido", `Moneda="USD" TipoCambio="18."`, "", all(stOk, stNone, stNone, stError)},
	})
}

func TestComprobanteTipoDeComprobante(t *testing.T) {
	codes := []string{"TIPOCOMP01", "TIPOCOMP02", "TIPOCOMP03", "TIPOCOMP04", "TIPOCOMP05",
		"TIPOCOMP06", "TIPOCOMP07", "TIPOCOMP08", "TIPOCOMP09", "TIPOCOMP10"}
	expectCodes(t, runValidator(ComprobanteTipoDeComprobante{}, newComprobante(t, "", "")), codes...)

	statuses := func(values ...validate.Status) map[string]validate.Status {
		m := map[string]validate.Status{}
		for i, v := range values {
			m[codes[i]] = v
		}
		return m
	}
	const conceptoCero = `<cfdi:Conceptos><cfdi:Concepto ValorUnitario="0" Importe="0"/></cfdi:Conceptos>`
	const conceptoDescuento = `<cfdi:Conceptos><cfdi:Concepto ValorUnitario="1" Importe="1" Descuento="0"/></cfdi:Conceptos>`
	const conceptos = `<cfdi:Conceptos><cfdi:Concepto ValorUnitario="10"/><cfdi:Concepto ValorUnitario="0.01"/></cfdi:Conceptos>`
	const conceptosConCero = `<cfdi:Conceptos><cfdi:Concepto ValorUnitario="10"/><cfdi:Concepto ValorUnitario="0.00"/></cfdi:Conceptos>`
	const conceptosNegativo = `<cfdi:Conceptos><cfdi:Concepto ValorUnitario="-10"/></cfdi:Conceptos>`

	runStatusCases(t, ComprobanteTipoDeComprobante{}, []statusCase{
		{"sin tipo", ``, "", statuses(stNone, stNone, stNone, stNone, stNone, stNone, stNone, stNone, stNone, stNone)},
		{"P correcto", `TipoDeComprobante="P" SubTotal="0" Total="0" Moneda="XXX"`, conceptoCero,
			statuses(stOk, stOk, stOk, stOk, stOk, stOk, stOk, stOk, stNone, stNone)},
		{"T correcto", `TipoDeComprobante="T" SubTotal="0.00" Total="0.00" Moneda="MXN"`, conceptoCero,
			statuses(stOk, stOk, stOk, stOk, stOk, stOk, stOk, stOk, stNone, stNone)},
		{"P incorrecto", `TipoDeComprobante="P" CondicionesDePago="x" FormaPago="01" MetodoPago="PUE" Descuento="0" SubTotal="1" Total=""`,
			`<cfdi:Impuestos/>` + conceptoDescuento,
			statuses(stError, stError, stError, stError, stError, stError, stError, stError, stNone, stNone)},
		{"T subtotal no numérico", `TipoDeComprobante="T" SubTotal="cero" Total="0"`, "",
			map[string]validate.Status{"TIPOCOMP07": stError, "TIPOCOMP08": stOk}},
		{"N correcto", `TipoDeComprobante="N" Moneda="MXN" FormaPago="99" MetodoPago="PUE"`, conceptos,
			statuses(stOk, stOk, stNone, stNone, stNone, stNone, stNone, stNone, stOk, stOk)},
		{"N incorrecto", `TipoDeComprobante="N" Moneda="USD" CondicionesDePago="x"`, `<cfdi:Impuestos/>` + conceptosConCero,
			statuses(stError, stError, stNone, stNone, stNone, stNone, stNone, stNone, stError, stError)},
		{"I correcto", `TipoDeComprobante="I" CondicionesDePago="x" FormaPago="01"`, `<cfdi:Impuestos/>` + conceptos,
			statuses(stNone, stNone, stNone, stNone, stNone, stNone, stNone, stNone, stOk, stNone)},
		{"E valor unitario cero", `TipoDeComprobante="E"`, conceptosConCero,
			map[string]validate.Status{"TIPOCOMP09": stError}},
		{"I valor unitario negativo", `TipoDeComprobante="I"`, conceptosNegativo,
			map[string]validate.Status{"TIPOCOMP09": stError}},
		{"I sin conceptos", `TipoDeComprobante="I"`, "",
			map[string]validate.Status{"TIPOCOMP09": stOk}},
	})
}

func TestComprobanteTotal(t *testing.T) {
	expectCodes(t, runValidator(ComprobanteTotal{}, newComprobante(t, "", "")), "TOTAL01")
	cases := map[string]validate.Status{
		`Total="0"`:        stOk,
		`Total="123.45"`:   stOk,
		`Total="0.000001"`: stOk,
		``:                 stError,
		`Total=""`:         stError,
		`Total="-1"`:       stError,
		`Total="1."`:       stError,
		`Total=".5"`:       stError,
		`Total="1,000.00"`: stError,
		`Total=" 1"`:       stError,
	}
	for attrs, expected := range cases {
		t.Run(attrs, func(t *testing.T) {
			expectStatuses(t, runValidator(ComprobanteTotal{}, newComprobante(t, attrs, "")),
				map[string]validate.Status{"TOTAL01": expected})
		})
	}
}
