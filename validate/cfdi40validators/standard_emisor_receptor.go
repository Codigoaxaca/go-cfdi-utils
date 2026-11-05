package cfdi40validators

import (
	"context"
	"fmt"
	"slices"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// Claves de c_RegimenFiscal de CFDI 4.0 (catCFDI_V_4, columnas "Física" y
// "Moral") que aplican a personas morales (RFC de 12 posiciones) y a personas
// físicas (RFC de 13 posiciones).
//
// Respecto del catálogo de CFDI 3.3 se eliminó la clave 609 (Consolidación),
// que no forma parte del catálogo de CFDI 4.0. Las claves 628 (Hidrocarburos,
// moral), 629 (De los Regímenes Fiscales Preferentes y de las Empresas
// Multinacionales, física) y 630 (Enajenación de acciones en bolsa de
// valores, física) se reincorporaron al catálogo de CFDI 4.0 con vigencia a
// partir del 2024-01-01. Las claves 610 (Residentes en el Extranjero sin
// Establecimiento Permanente en México) y 626 (Régimen Simplificado de
// Confianza) aplican a ambos tipos de persona.
var (
	regimenFiscalPersonaMoral = []string{
		"601", "603", "610", "620", "622", "623", "624", "626", "628",
	}
	regimenFiscalPersonaFisica = []string{
		"605", "606", "607", "608", "610", "611", "612", "614", "615", "616", "621", "625", "626", "629", "630",
	}
)

// EmisorRegimenFiscal valida el régimen fiscal del emisor según su tipo de RFC.
//
//   - REGFIS01: El régimen fiscal contenga un valor apropiado según el tipo de RFC emisor
//
// No valida que el RFC sea válido, esa responsabilidad no es de este validador.
// Ver regimenFiscalPersonaMoral y regimenFiscalPersonaFisica para las claves
// del catálogo c_RegimenFiscal de CFDI 4.0 aceptadas.
type EmisorRegimenFiscal struct{ version40 }

// Validate implementa validate.Validator.
func (EmisorRegimenFiscal) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	regimenFiscal := c.SearchAttribute("cfdi:Emisor", "RegimenFiscal")
	rfc := c.SearchAttribute("cfdi:Emisor", "Rfc")
	var validCodes []string
	switch len([]rune(rfc)) {
	case 12:
		validCodes = regimenFiscalPersonaMoral
	case 13:
		validCodes = regimenFiscalPersonaFisica
	}
	asserts.Put("REGFIS01", "El régimen fiscal contenga un valor apropiado según el tipo de RFC emisor").
		SetStatus(validate.StatusWhen(slices.Contains(validCodes, regimenFiscal)),
			fmt.Sprintf("Rfc: %q, Regimen Fiscal: %q", rfc, regimenFiscal))
}

// ---------------------------------------------------------------------------

// rfcGenericoExtranjero es el RFC genérico para residentes en el extranjero.
const rfcGenericoExtranjero = "XEXX010101000"

// ReceptorResidenciaFiscal valida la residencia fiscal del receptor.
//
//   - RESFISC01: Si el RFC no es XEXX010101000, entonces la residencia fiscal no debe existir
//   - RESFISC02: Si el RFC sí es XEXX010101000 y existe el complemento de comercio exterior, entonces la residencia fiscal debe establecerse y no puede ser "MEX"
//   - RESFISC03: Si el RFC sí es XEXX010101000 y se registró el número de registro de identificación fiscal, entonces la residencia fiscal debe establecerse y no puede ser "MEX"
//
// El complemento de comercio exterior se reconoce como cce20:ComercioExterior
// (versión 2.0, la vigente con CFDI 4.0) o cce11:ComercioExterior.
type ReceptorResidenciaFiscal struct{ version40 }

// Validate implementa validate.Validator.
func (ReceptorResidenciaFiscal) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	putTitles(asserts, [][2]string{
		{"RESFISC01", "Si el RFC no es " + rfcGenericoExtranjero + ", entonces la residencia fiscal no debe existir"},
		{"RESFISC02", "Si el RFC sí es " + rfcGenericoExtranjero + " y existe el complemento de comercio exterior, entonces la residencia fiscal debe establecerse y no puede ser \"MEX\""},
		{"RESFISC03", "Si el RFC sí es " + rfcGenericoExtranjero + " y se registró el número de registro de identificación fiscal, entonces la residencia fiscal debe establecerse y no puede ser \"MEX\""},
	})
	receptor := c.SearchNode("cfdi:Receptor")
	if receptor == nil {
		receptor = nodes.New("cfdi:Receptor", nil)
	}
	residencia, existsResidencia := receptor.LookupAttr("ResidenciaFiscal")
	explanation := fmt.Sprintf("Rfc: %q, ResidenciaFiscal: %q", receptor.Attr("Rfc"), residencia)
	if receptor.Attr("Rfc") != rfcGenericoExtranjero {
		asserts.PutStatus("RESFISC01", validate.StatusWhen(!existsResidencia), explanation)
		return
	}
	isValidResidencia := residencia != "" && residencia != "MEX"
	if c.SearchNode("cfdi:Complemento", "cce20:ComercioExterior") != nil ||
		c.SearchNode("cfdi:Complemento", "cce11:ComercioExterior") != nil {
		asserts.PutStatus("RESFISC02", validate.StatusWhen(isValidResidencia), explanation)
	}
	if receptor.HasAttr("NumRegIdTrib") {
		asserts.PutStatus("RESFISC03", validate.StatusWhen(isValidResidencia), explanation)
	}
}
