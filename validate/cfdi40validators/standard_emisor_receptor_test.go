package cfdi40validators

import (
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

func TestEmisorRegimenFiscal(t *testing.T) {
	expectCodes(t, runValidator(EmisorRegimenFiscal{}, newComprobante(t, "", "")), "REGFIS01")

	const moral = "EKU9003173C9"
	const fisica = "COSC8001137NA"
	cases := []struct {
		rfc, regimen string
		expected     validate.Status
	}{
		{moral, "601", stOk},
		{moral, "603", stOk},
		{moral, "610", stOk},
		{moral, "620", stOk},
		{moral, "622", stOk},
		{moral, "623", stOk},
		{moral, "624", stOk},
		{moral, "626", stOk},
		{moral, "628", stOk},
		{moral, "609", stError}, // no existe en el catálogo de CFDI 4.0
		{moral, "605", stError},
		{moral, "612", stError},
		{moral, "", stError},
		{"AÑB0101011A1", "601", stOk}, // RFC con Ñ, 12 caracteres
		{fisica, "605", stOk},
		{fisica, "606", stOk},
		{fisica, "607", stOk},
		{fisica, "608", stOk},
		{fisica, "610", stOk},
		{fisica, "611", stOk},
		{fisica, "612", stOk},
		{fisica, "614", stOk},
		{fisica, "615", stOk},
		{fisica, "616", stOk},
		{fisica, "621", stOk},
		{fisica, "625", stOk},
		{fisica, "626", stOk},
		{fisica, "629", stOk},
		{fisica, "630", stOk},
		{fisica, "601", stError},
		{fisica, "628", stError},
		{"", "601", stError},
		{"ABC", "601", stError},
	}
	for _, tc := range cases {
		t.Run(tc.rfc+"/"+tc.regimen, func(t *testing.T) {
			c := newComprobante(t, "", `<cfdi:Emisor Rfc="`+tc.rfc+`" RegimenFiscal="`+tc.regimen+`"/>`)
			expectStatuses(t, runValidator(EmisorRegimenFiscal{}, c), map[string]validate.Status{"REGFIS01": tc.expected})
		})
	}

	// sin emisor
	expectStatuses(t, runValidator(EmisorRegimenFiscal{}, newComprobante(t, "", "")),
		map[string]validate.Status{"REGFIS01": stError})
}

func TestReceptorResidenciaFiscal(t *testing.T) {
	expectCodes(t, runValidator(ReceptorResidenciaFiscal{}, newComprobante(t, "", "")),
		"RESFISC01", "RESFISC02", "RESFISC03")

	const cce20 = `<cfdi:Complemento><cce20:ComercioExterior xmlns:cce20="http://www.sat.gob.mx/ComercioExterior20" Version="2.0"/></cfdi:Complemento>`
	const cce11 = `<cfdi:Complemento><cce11:ComercioExterior xmlns:cce11="http://www.sat.gob.mx/ComercioExterior11" Version="1.1"/></cfdi:Complemento>`
	all := func(s01, s02, s03 validate.Status) map[string]validate.Status {
		return map[string]validate.Status{"RESFISC01": s01, "RESFISC02": s02, "RESFISC03": s03}
	}
	runStatusCases(t, ReceptorResidenciaFiscal{}, []statusCase{
		{"sin receptor", "", "", all(stOk, stNone, stNone)},
		{"nacional sin residencia", "", `<cfdi:Receptor Rfc="COSC8001137NA"/>`, all(stOk, stNone, stNone)},
		{"nacional con residencia", "", `<cfdi:Receptor Rfc="COSC8001137NA" ResidenciaFiscal="USA"/>`, all(stError, stNone, stNone)},
		{"genérico nacional con residencia vacía", "", `<cfdi:Receptor Rfc="XAXX010101000" ResidenciaFiscal=""/>`, all(stError, stNone, stNone)},
		{"extranjero sin complemento ni registro", "", `<cfdi:Receptor Rfc="XEXX010101000"/>`, all(stNone, stNone, stNone)},
		{"extranjero cce20 con residencia", "", `<cfdi:Receptor Rfc="XEXX010101000" ResidenciaFiscal="USA"/>` + cce20, all(stNone, stOk, stNone)},
		{"extranjero cce20 sin residencia", "", `<cfdi:Receptor Rfc="XEXX010101000"/>` + cce20, all(stNone, stError, stNone)},
		{"extranjero cce20 con MEX", "", `<cfdi:Receptor Rfc="XEXX010101000" ResidenciaFiscal="MEX"/>` + cce20, all(stNone, stError, stNone)},
		{"extranjero cce11 sin residencia", "", `<cfdi:Receptor Rfc="XEXX010101000"/>` + cce11, all(stNone, stError, stNone)},
		{"extranjero registro con residencia", "", `<cfdi:Receptor Rfc="XEXX010101000" NumRegIdTrib="123" ResidenciaFiscal="USA"/>`, all(stNone, stNone, stOk)},
		{"extranjero registro sin residencia", "", `<cfdi:Receptor Rfc="XEXX010101000" NumRegIdTrib="123"/>`, all(stNone, stNone, stError)},
		{"extranjero registro con MEX", "", `<cfdi:Receptor Rfc="XEXX010101000" NumRegIdTrib="123" ResidenciaFiscal="MEX"/>`, all(stNone, stNone, stError)},
		{"extranjero completo", "", `<cfdi:Receptor Rfc="XEXX010101000" NumRegIdTrib="123" ResidenciaFiscal="CAN"/>` + cce20, all(stNone, stOk, stOk)},
	})
}
