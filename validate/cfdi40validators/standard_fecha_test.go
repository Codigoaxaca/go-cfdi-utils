package cfdi40validators

import (
	"testing"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

func TestFechaComprobante(t *testing.T) {
	maximum := time.Date(2025, time.June, 30, 12, 0, 0, 0, MexicoLocation)
	v := FechaComprobante{MaximumDate: maximum}
	expectCodes(t, runValidator(v, newComprobante(t, "", "")), "FECHA01", "FECHA02")

	cases := []struct {
		name      string
		attrs     string
		fecha01   validate.Status
		fecha02   validate.Status
		validator FechaComprobante
	}{
		{"sin fecha", ``, stError, stNone, v},
		{"vacía", `Fecha=""`, stError, stNone, v},
		{"sin hora", `Fecha="2023-01-01"`, stError, stNone, v},
		{"con zona horaria", `Fecha="2023-01-01T00:00:00-06:00"`, stError, stNone, v},
		{"con milisegundos", `Fecha="2023-01-01T00:00:00.000"`, stError, stNone, v},
		{"con espacios", `Fecha=" 2023-01-01T00:00:00"`, stError, stNone, v},
		{"día inválido", `Fecha="2023-02-30T00:00:00"`, stError, stNone, v},
		{"hora inválida", `Fecha="2023-01-01T24:00:00"`, stError, stNone, v},
		{"mínima", `Fecha="2022-01-01T00:00:00"`, stOk, stOk, v},
		{"antes de la mínima", `Fecha="2021-12-31T23:59:59"`, stOk, stError, v},
		{"fecha de CFDI 3.3", `Fecha="2018-01-01T00:00:00"`, stOk, stError, v},
		{"máxima", `Fecha="2025-06-30T12:00:00"`, stOk, stOk, v},
		{"después de la máxima", `Fecha="2025-06-30T12:00:01"`, stOk, stError, v},
		{"fecha pasada con máxima por defecto", `Fecha="2023-06-30T12:00:01"`, stOk, stOk, FechaComprobante{}},
		{"futuro lejano con máxima por defecto", `Fecha="2099-01-01T00:00:00"`, stOk, stError, FechaComprobante{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectStatuses(t, runValidator(tc.validator, newComprobante(t, tc.attrs, "")),
				map[string]validate.Status{"FECHA01": tc.fecha01, "FECHA02": tc.fecha02})
		})
	}
}

func TestFechaComprobanteMaximumDate(t *testing.T) {
	now := time.Date(2024, time.March, 1, 10, 0, 0, 0, MexicoLocation)
	fixed := time.Date(2030, time.January, 1, 0, 0, 0, 0, MexicoLocation)
	cases := []struct {
		name     string
		v        FechaComprobante
		expected time.Time
	}{
		{"zero value usa la tolerancia por defecto", FechaComprobante{}, now.Add(300 * time.Second)},
		{"tolerancia por defecto de StandardValidators", FechaComprobante{Tolerance: DefaultFechaTolerance}, now.Add(5 * time.Minute)},
		{"tolerancia configurada", FechaComprobante{Tolerance: time.Hour}, now.Add(time.Hour)},
		{"tolerancia negativa usa la de por defecto", FechaComprobante{Tolerance: -time.Hour}, now.Add(DefaultFechaTolerance)},
		{"máxima configurada ignora tolerancia", FechaComprobante{MaximumDate: fixed, Tolerance: time.Hour}, fixed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.MaximumDateAt(now); !got.Equal(tc.expected) {
				t.Errorf("se esperaba %s y se obtuvo %s", tc.expected, got)
			}
		})
	}

	// con la tolerancia por defecto, una fecha de hace un momento (en hora de
	// México) es válida y una de dentro de una hora no
	inMexico := time.Now().In(MexicoLocation)
	for _, tc := range []struct {
		fecha    time.Time
		expected validate.Status
	}{
		{inMexico.Add(-time.Minute), stOk},
		{inMexico.Add(2 * time.Minute), stOk},
		{inMexico.Add(time.Hour), stError},
	} {
		c := newComprobante(t, `Fecha="`+tc.fecha.Format("2006-01-02T15:04:05")+`"`, "")
		expectStatuses(t, runValidator(FechaComprobante{}, c), map[string]validate.Status{"FECHA02": tc.expected})
	}
}
