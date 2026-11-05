package validate

import (
	"context"

	"github.com/codigoaxaca/go-cfdi-utils/cadenaorigen"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/xmlresolver"
)

// Environment contiene los recursos que los validadores pueden necesitar.
type Environment struct {
	// XML es el contenido XML original del comprobante, necesario para
	// generar la cadena original tal como fue recibida.
	XML []byte
	// CadenaOrigen genera la cadena original. Si es nil la validación del
	// sello no se puede realizar.
	CadenaOrigen cadenaorigen.Builder
	// Resolver obtiene los certificados del SAT para validar el sello del
	// Timbre Fiscal Digital. Si es nil esa validación no se realiza.
	Resolver *xmlresolver.Resolver
	// SchemaValidator valida el XML contra sus esquemas XSD. Si es nil la
	// validación de esquemas no se realiza.
	SchemaValidator SchemaValidator
}

// SchemaValidator valida un documento XML contra los esquemas XSD declarados
// en xsi:schemaLocation.
type SchemaValidator interface {
	ValidateSchemas(ctx context.Context, xml []byte) error
}

// Validator es un validador que agrega uno o más Assert a la colección.
type Validator interface {
	// Validate valida el comprobante (nodo cfdi:Comprobante) y escribe los
	// resultados en asserts.
	Validate(ctx context.Context, comprobante *nodes.Node, env *Environment, asserts *Asserts)
	// CanValidateCfdiVersion informa si el validador aplica para la versión dada.
	CanValidateCfdiVersion(version string) bool
}

// ValidatorFunc permite usar una función como Validator para cualquier versión.
type ValidatorFunc func(ctx context.Context, comprobante *nodes.Node, env *Environment, asserts *Asserts)

// Validate ejecuta la función.
func (f ValidatorFunc) Validate(ctx context.Context, comprobante *nodes.Node, env *Environment, asserts *Asserts) {
	f(ctx, comprobante, env, asserts)
}

// CanValidateCfdiVersion siempre devuelve verdadero.
func (f ValidatorFunc) CanValidateCfdiVersion(string) bool { return true }

// MultiValidator ejecuta una lista de validadores para una versión de CFDI.
type MultiValidator struct {
	Version    string
	Validators []Validator
}

// NewMultiValidator crea un MultiValidator para la versión dada.
func NewMultiValidator(version string, validators ...Validator) *MultiValidator {
	return &MultiValidator{Version: version, Validators: validators}
}

// Add agrega validadores.
func (m *MultiValidator) Add(validators ...Validator) *MultiValidator {
	m.Validators = append(m.Validators, validators...)
	return m
}

// CanValidateCfdiVersion informa si la versión es la del MultiValidator.
func (m *MultiValidator) CanValidateCfdiVersion(version string) bool { return m.Version == version }

// Validate ejecuta cada validador compatible con la versión. Si algún
// validador establece MustStop el resto ya no se ejecuta.
func (m *MultiValidator) Validate(ctx context.Context, comprobante *nodes.Node, env *Environment, asserts *Asserts) {
	if env == nil {
		env = &Environment{}
	}
	for _, v := range m.Validators {
		if asserts.MustStop() {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if !v.CanValidateCfdiVersion(m.Version) {
			continue
		}
		v.Validate(ctx, comprobante, env, asserts)
	}
}
