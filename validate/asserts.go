// Package validate contiene la infraestructura de validación de CFDI:
// estados (Status), aseguramientos (Assert), su colección (Asserts), la
// interfaz Validator y el ejecutor MultiValidator.
//
// Un Assert es un aseguramiento, un enunciado afirmativo como "El sello del
// comprobante está en base 64", con un estado: ERROR, WARN, OK o NONE. Al
// validar no solo se reportan los errores, también lo que fue correcto y lo
// que no se pudo comprobar.
//
// Los validadores de CFDI 4.0 están en el paquete validate/v40.
package validate

import (
	"fmt"
	"strings"
)

// Status es el estado de un Assert.
type Status int

const (
	// StatusNone indica que la prueba no se realizó.
	StatusNone Status = iota
	// StatusOk indica que se realizó la prueba y no se encontró fallo.
	StatusOk
	// StatusWarning indica que existe un fallo pero se desconoce si es correcto o incorrecto.
	StatusWarning
	// StatusError indica un fallo, el CFDI se debe considerar inválido.
	StatusError
)

// String devuelve ERROR, WARN, OK o NONE.
func (s Status) String() string {
	switch s {
	case StatusOk:
		return "OK"
	case StatusWarning:
		return "WARN"
	case StatusError:
		return "ERROR"
	default:
		return "NONE"
	}
}

// IsError informa si el estado es error.
func (s Status) IsError() bool { return s == StatusError }

// IsWarning informa si el estado es advertencia.
func (s Status) IsWarning() bool { return s == StatusWarning }

// IsOk informa si el estado es correcto.
func (s Status) IsOk() bool { return s == StatusOk }

// IsNone informa si la prueba no se realizó.
func (s Status) IsNone() bool { return s == StatusNone }

// StatusWhen devuelve StatusOk si la condición es verdadera, StatusError si no.
func StatusWhen(condition bool) Status {
	if condition {
		return StatusOk
	}
	return StatusError
}

// StatusWhenOrWarning devuelve StatusOk si la condición es verdadera, StatusWarning si no.
func StatusWhenOrWarning(condition bool) Status {
	if condition {
		return StatusOk
	}
	return StatusWarning
}

// Assert es un aseguramiento con código, título, estado y explicación.
type Assert struct {
	Code        string
	Title       string
	Status      Status
	Explanation string
}

// SetStatus establece el estado y, si no está vacía, la explicación.
func (a *Assert) SetStatus(status Status, explanation ...string) *Assert {
	a.Status = status
	if len(explanation) > 0 && explanation[0] != "" {
		a.Explanation = explanation[0]
	}
	return a
}

// SetExplanation establece la explicación.
func (a *Assert) SetExplanation(explanation string) *Assert {
	a.Explanation = explanation
	return a
}

// String devuelve una representación como "ERROR: SELLO08 - El sello ...".
func (a *Assert) String() string {
	return fmt.Sprintf("%s: %s - %s", a.Status, a.Code, a.Title)
}

// Asserts es una colección ordenada de Assert identificados por código.
// No pueden existir dos Assert con el mismo código; al escribir un código
// existente se actualiza el previo.
type Asserts struct {
	items    []*Assert
	index    map[string]*Assert
	mustStop bool
}

// NewAsserts crea una colección vacía.
func NewAsserts() *Asserts {
	return &Asserts{index: map[string]*Assert{}}
}

// Put crea (o actualiza el título de) un Assert con estado NONE si es nuevo.
func (as *Asserts) Put(code, title string) *Assert {
	if as.index == nil {
		as.index = map[string]*Assert{}
	}
	if a, ok := as.index[code]; ok {
		if title != "" {
			a.Title = title
		}
		return a
	}
	a := &Assert{Code: code, Title: title, Status: StatusNone}
	as.items = append(as.items, a)
	as.index[code] = a
	return a
}

// PutStatus establece el estado (y explicación opcional) de un Assert,
// creándolo si no existe.
func (as *Asserts) PutStatus(code string, status Status, explanation ...string) *Assert {
	return as.Put(code, "").SetStatus(status, explanation...)
}

// Add agrega (o reemplaza por código) un Assert.
func (as *Asserts) Add(a *Assert) {
	if existing, ok := as.index[a.Code]; ok {
		*existing = *a
		return
	}
	as.Put(a.Code, a.Title).SetStatus(a.Status, a.Explanation)
}

// Get devuelve el Assert con el código dado o nil.
func (as *Asserts) Get(code string) *Assert { return as.index[code] }

// Exists informa si existe un Assert con el código dado.
func (as *Asserts) Exists(code string) bool {
	_, ok := as.index[code]
	return ok
}

// Remove elimina el Assert con el código dado.
func (as *Asserts) Remove(code string) {
	if _, ok := as.index[code]; !ok {
		return
	}
	delete(as.index, code)
	for i, a := range as.items {
		if a.Code == code {
			as.items = append(as.items[:i], as.items[i+1:]...)
			break
		}
	}
}

// Import agrega todos los Assert de otra colección.
func (as *Asserts) Import(other *Asserts) {
	for _, a := range other.items {
		copied := *a
		as.Add(&copied)
	}
	if other.mustStop {
		as.mustStop = true
	}
}

// All devuelve todos los Assert en el orden en que se agregaron.
func (as *Asserts) All() []*Assert { return append([]*Assert(nil), as.items...) }

// Len devuelve la cantidad de Assert.
func (as *Asserts) Len() int { return len(as.items) }

// ByStatus devuelve los Assert con el estado dado.
func (as *Asserts) ByStatus(status Status) []*Assert {
	var out []*Assert
	for _, a := range as.items {
		if a.Status == status {
			out = append(out, a)
		}
	}
	return out
}

// Errors devuelve los Assert con estado ERROR.
func (as *Asserts) Errors() []*Assert { return as.ByStatus(StatusError) }

// Warnings devuelve los Assert con estado WARN.
func (as *Asserts) Warnings() []*Assert { return as.ByStatus(StatusWarning) }

// Oks devuelve los Assert con estado OK.
func (as *Asserts) Oks() []*Assert { return as.ByStatus(StatusOk) }

// Nones devuelve los Assert con estado NONE.
func (as *Asserts) Nones() []*Assert { return as.ByStatus(StatusNone) }

// HasStatus informa si existe algún Assert con el estado dado.
func (as *Asserts) HasStatus(status Status) bool {
	for _, a := range as.items {
		if a.Status == status {
			return true
		}
	}
	return false
}

// HasErrors informa si existe algún Assert con estado ERROR.
func (as *Asserts) HasErrors() bool { return as.HasStatus(StatusError) }

// HasWarnings informa si existe algún Assert con estado WARN.
func (as *Asserts) HasWarnings() bool { return as.HasStatus(StatusWarning) }

// FirstError devuelve el primer Assert con estado ERROR o nil.
func (as *Asserts) FirstError() *Assert {
	for _, a := range as.items {
		if a.Status == StatusError {
			return a
		}
	}
	return nil
}

// MustStop informa si un validador solicitó detener el resto de validaciones.
func (as *Asserts) MustStop() bool { return as.mustStop }

// SetMustStop establece si se deben detener las validaciones siguientes.
func (as *Asserts) SetMustStop(stop bool) { as.mustStop = stop }

// String genera un reporte de texto, una línea por Assert.
func (as *Asserts) String() string {
	var b strings.Builder
	for _, a := range as.items {
		fmt.Fprintf(&b, "%-5s %-12s %s", a.Status, a.Code, a.Title)
		if a.Explanation != "" {
			fmt.Fprintf(&b, " => %s", a.Explanation)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
