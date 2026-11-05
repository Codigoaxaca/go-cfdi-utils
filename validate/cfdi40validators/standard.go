package cfdi40validators

import (
	"strconv"
	"strings"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// StandardValidators devuelve las validaciones estándar adicionales de CFDI 4.0.
//
// Son la adaptación a CFDI 4.0 (Anexo 20 versión 4.0) de las validaciones
// estándar de CFDI 3.3 de eclipxe/CfdiUtils. Se devuelven en el orden en que
// la librería PHP las descubre (orden alfabético de sus nombres):
//
//   - ComprobanteDecimalesMoneda: MONDEC01..MONDEC05
//   - ComprobanteDescuento: DESCUENTO01
//   - ComprobanteFormaPago: FORMAPAGO01
//   - ComprobanteImpuestos: COMPIMPUESTOSC01..COMPIMPUESTOSC04
//   - ComprobanteTipoCambio: TIPOCAMBIO01..TIPOCAMBIO04
//   - ComprobanteTipoDeComprobante: TIPOCOMP01..TIPOCOMP10
//   - ComprobanteTotal: TOTAL01
//   - ConceptoDescuento: CONCEPDESC01
//   - ConceptoImpuestos: CONCEPIMPC01..CONCEPIMPC08
//   - EmisorRegimenFiscal: REGFIS01
//   - FechaComprobante: FECHA01, FECHA02
//   - ReceptorResidenciaFiscal: RESFISC01..RESFISC03
//   - SumasConceptosComprobanteImpuestos: SUMAS01..SUMAS13
func StandardValidators() []validate.Validator {
	return []validate.Validator{
		ComprobanteDecimalesMoneda{},
		ComprobanteDescuento{},
		ComprobanteFormaPago{},
		ComprobanteImpuestos{},
		ComprobanteTipoCambio{},
		ComprobanteTipoDeComprobante{},
		ComprobanteTotal{},
		ConceptoDescuento{},
		ConceptoImpuestos{},
		EmisorRegimenFiscal{},
		FechaComprobante{Tolerance: DefaultFechaTolerance},
		ReceptorResidenciaFiscal{},
		SumasConceptosComprobanteImpuestos{},
	}
}

// putTitles registra los Assert (con estado NONE) en el orden dado.
func putTitles(asserts *validate.Asserts, titles [][2]string) {
	for _, t := range titles {
		asserts.Put(t[0], t[1])
	}
}

// knownCurrencyDecimals devuelve los decimales de las monedas conocidas
// (port de CfdiUtils\Utils\CurrencyDecimals::knownCurrencyDecimals).
func knownCurrencyDecimals(currency string) (int, bool) {
	switch currency {
	case "MXN", "EUR", "USD":
		return 2, true
	case "XXX":
		return 0, true
	}
	return 0, false
}

// decimalsCount devuelve la cantidad de caracteres después del último punto
// (como pathinfo($value, PATHINFO_EXTENSION) en PHP).
func decimalsCount(value string) int {
	i := strings.LastIndex(value, ".")
	if i < 0 {
		return 0
	}
	return len(value) - i - 1
}

// parseNumeric interpreta un valor numérico (similar a is_numeric de PHP).
func parseNumeric(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// toFloat convierte como (float) de PHP en los casos comunes: vacío o
// inválido es cero.
func toFloat(value string) float64 {
	f, _ := parseNumeric(value)
	return f
}

// formatFloat expresa un número con la representación más corta.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// DefaultFechaTolerance es la tolerancia por defecto de FechaComprobante.
const DefaultFechaTolerance = 300 * time.Second
