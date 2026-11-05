package cfdi40validators

import (
	"context"
	"fmt"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

// FechaComprobanteMinima es la fecha mínima de un CFDI 4.0: 2022-01-01
// 00:00:00 hora del centro de México (inicio de vigencia de la versión 4.0).
var FechaComprobanteMinima = time.Date(2022, time.January, 1, 0, 0, 0, 0, MexicoLocation)

// FechaComprobante valida la fecha del comprobante.
//
//   - FECHA01: La fecha del comprobante cumple con el formato
//   - FECHA02: La fecha existe en el comprobante y es mayor que 2022-01-01 y menor que el futuro
//
// La fecha se interpreta en hora del centro de México (MexicoLocation). La
// fecha máxima ("el futuro") es MaximumDate si no es cero; en otro caso es el
// momento de la validación más Tolerance. FECHA02 queda en NONE si la fecha
// no cumple con el formato.
type FechaComprobante struct {
	version40
	// MaximumDate es la fecha máxima permitida. Si es cero se usa el momento
	// de la validación más Tolerance.
	MaximumDate time.Time
	// Tolerance es la tolerancia que se suma al momento de la validación
	// para obtener la fecha máxima. Si es cero o negativa se usa
	// DefaultFechaTolerance (300 segundos).
	Tolerance time.Duration
}

// MaximumDateAt devuelve la fecha máxima permitida tomando now como el
// momento de la validación.
func (v FechaComprobante) MaximumDateAt(now time.Time) time.Time {
	if !v.MaximumDate.IsZero() {
		return v.MaximumDate
	}
	tolerance := v.Tolerance
	if tolerance <= 0 {
		tolerance = DefaultFechaTolerance
	}
	return now.Add(tolerance)
}

// Validate implementa validate.Validator.
func (v FechaComprobante) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	value, exists := c.LookupAttr("Fecha")
	fecha, hasFormat := parseFechaStrict(value)
	asserts.Put("FECHA01", "La fecha del comprobante cumple con el formato").
		SetStatus(validate.StatusWhen(hasFormat), fmt.Sprintf("Contenido del campo: %q", value))
	a := asserts.Put("FECHA02", "La fecha existe en el comprobante y es mayor que 2022-01-01 y menor que el futuro")
	if !hasFormat {
		return
	}
	maximum := v.MaximumDateAt(time.Now())
	existsText := "No existe"
	if exists {
		existsText = "Existe"
	}
	a.SetStatus(validate.StatusWhen(!fecha.Before(FechaComprobanteMinima) && !fecha.After(maximum)),
		fmt.Sprintf("Fecha: %q (%s), Máxima: %s", value, existsText, maximum.In(MexicoLocation).Format("2006-01-02 15:04:05")))
}

// parseFechaStrict interpreta una fecha AAAA-MM-DDThh:mm:ss y exige que el
// texto sea exactamente la representación de la fecha (sin espacios ni
// valores fuera de rango).
func parseFechaStrict(value string) (time.Time, bool) {
	fecha, ok := ParseFecha(value)
	if !ok || fecha.Format("2006-01-02T15:04:05") != value {
		return time.Time{}, false
	}
	return fecha, true
}
