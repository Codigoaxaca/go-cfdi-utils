package cfdi40

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
	"github.com/codigoaxaca/go-cfdi-utils/validate/cfdi40validators"
)

// Validator valida CFDI 4.0 recibidos (ya creados, normalmente timbrados).
//
//	v := cfdi40.NewValidator()
//	asserts, err := v.ValidateXML(xmlBytes)
//	if asserts.HasErrors() { ... }
type Validator struct {
	config
	multi *validate.MultiValidator
}

// NewValidator crea un validador con las validaciones de CFDI 4.0 recibidos.
func NewValidator(opts ...Option) *Validator {
	return &Validator{config: newConfig(opts), multi: cfdi40validators.NewReceivedValidator()}
}

// MultiValidator devuelve el conjunto de validadores, permite agregar
// validadores propios con Add.
func (v *Validator) MultiValidator() *validate.MultiValidator { return v.multi }

// ValidateXML valida un contenido XML. Devuelve error solo si el XML no se
// puede interpretar; el resultado de las validaciones está en los Asserts.
func (v *Validator) ValidateXML(xml []byte) (*validate.Asserts, error) {
	return v.ValidateXMLContext(context.Background(), xml)
}

// ValidateXMLContext es como ValidateXML pero recibe un contexto.
func (v *Validator) ValidateXMLContext(ctx context.Context, xml []byte) (*validate.Asserts, error) {
	node, err := nodes.ParseXML(xml)
	if err != nil {
		return nil, fmt.Errorf("cfdi40: %w", err)
	}
	asserts := validate.NewAsserts()
	v.multi.Validate(ctx, node, v.environment(xml), asserts)
	return asserts, nil
}

// ValidateFile valida el archivo XML en la ruta dada.
func (v *Validator) ValidateFile(path string) (*validate.Asserts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cfdi40: %w", err)
	}
	return v.ValidateXML(data)
}

// ValidateXML valida un CFDI 4.0 recibido con la configuración predeterminada.
func ValidateXML(xml []byte) (*validate.Asserts, error) {
	return NewValidator().ValidateXML(xml)
}

func encodeBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
