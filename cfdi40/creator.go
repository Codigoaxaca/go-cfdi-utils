// Package cfdi40 permite crear, sellar y validar CFDI versión 4.0.
//
// La forma más sencilla de crear un CFDI es con Creator, que trabaja sobre el
// elemento raíz Comprobante y sabe cómo establecer el certificado, calcular
// las sumas de los conceptos, generar el sello, validar y exportar el XML:
//
//	csd, err := credentials.OpenFiles("csd.cer", "csd.key", "contraseña")
//	creator := cfdi40.NewCreator(cfdi40.Attrs{"Serie": "A", "Folio": "1", ...})
//	creator.PutCertificado(csd.Certificate(), true)
//	comprobante := creator.Comprobante()
//	comprobante.AddEmisor(cfdi40.Attrs{"RegimenFiscal": "601"})
//	comprobante.AddReceptor(cfdi40.Attrs{...})
//	comprobante.AddConcepto(cfdi40.Attrs{...}).AddTraslado(cfdi40.Attrs{...})
//	creator.AddSumasConceptos(nil, 2)
//	err = creator.AddSello(csd.PrivateKey())
//	asserts := creator.Validate()
//	xml := creator.AsXML()
//
// Los elementos (Comprobante, Emisor, Concepto, ...) conocen el orden en que
// el SAT exige sus nodos hijos, por lo que no es necesario preocuparse por el
// orden en que se agregan.
package cfdi40

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/codigoaxaca/go-cfdi-utils/cadenaorigen"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/sumasconceptos"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
	"github.com/codigoaxaca/go-cfdi-utils/validate/cfdi40validators"
	"github.com/codigoaxaca/go-cfdi-utils/xmlresolver"
	"github.com/codigoaxaca/go-cfdi-utils/xmlschema"
)

// ErrKeyDoesNotBelongToCertificate se devuelve al sellar con una llave
// privada que no corresponde al certificado establecido.
var ErrKeyDoesNotBelongToCertificate = errors.New("cfdi40: la llave privada no corresponde al certificado")

// config contiene las dependencias compartidas por Creator y Validator.
type config struct {
	cadenaOrigen    cadenaorigen.Builder
	resolver        *xmlresolver.Resolver
	schemaValidator validate.SchemaValidator
	schemaSet       bool
}

// Option configura un Creator o un Validator.
type Option func(*config)

// WithCadenaOrigenBuilder establece el generador de cadena original. Por
// omisión se usa cadenaorigen.NewBuilder() que no requiere herramientas externas.
func WithCadenaOrigenBuilder(b cadenaorigen.Builder) Option {
	return func(c *config) { c.cadenaOrigen = b }
}

// WithResolver establece el resolvedor de recursos del SAT (certificados y
// esquemas). Por omisión se usa xmlresolver.New() con caché en disco. Usar nil
// desactiva las validaciones que requieren descargar recursos.
func WithResolver(r *xmlresolver.Resolver) Option {
	return func(c *config) { c.resolver = r }
}

// WithSchemaValidator establece el validador de esquemas XSD. Por omisión se
// usa xmllint si está disponible en el sistema. Usar nil desactiva la
// validación de esquemas (XSD01 quedará con estado NONE).
func WithSchemaValidator(v validate.SchemaValidator) Option {
	return func(c *config) { c.schemaValidator = v; c.schemaSet = true }
}

func newConfig(opts []Option) config {
	c := config{cadenaOrigen: cadenaorigen.NewBuilder(), resolver: xmlresolver.New()}
	for _, opt := range opts {
		opt(&c)
	}
	if !c.schemaSet && c.resolver != nil {
		if sv, err := xmlschema.NewXmllint(c.resolver); err == nil {
			c.schemaValidator = sv
		}
	}
	return c
}

func (c config) environment(xml []byte) *validate.Environment {
	return &validate.Environment{
		XML:             xml,
		CadenaOrigen:    c.cadenaOrigen,
		Resolver:        c.resolver,
		SchemaValidator: c.schemaValidator,
	}
}

// Creator ayuda a crear un CFDI 4.0: contiene el Comprobante y métodos para
// establecer el certificado, calcular sumas, sellar, validar y exportar.
type Creator struct {
	config
	comprobante *Comprobante
	certificado *credentials.Certificate
}

// NewCreator crea un Creator con los atributos dados para el Comprobante.
func NewCreator(attrs Attrs, opts ...Option) *Creator {
	return &Creator{config: newConfig(opts), comprobante: NewComprobante(attrs)}
}

// NewCreatorFromNode crea un Creator a partir de un nodo cfdi:Comprobante
// existente, copiando sus atributos e hijos.
func NewCreatorFromNode(node *nodes.Node, opts ...Option) *Creator {
	c := NewCreator(nil, opts...)
	c.comprobante.SetAttrs(node.AttrsMap())
	for _, child := range node.Children() {
		c.comprobante.AddChild(child)
	}
	return c
}

// NewCreatorFromXML crea un Creator a partir de un contenido XML de CFDI 4.0,
// por ejemplo para modificar un precfdi y volverlo a sellar.
func NewCreatorFromXML(xml []byte, opts ...Option) (*Creator, error) {
	node, err := nodes.ParseXML(xml)
	if err != nil {
		return nil, err
	}
	return NewCreatorFromNode(node, opts...), nil
}

// Comprobante devuelve el nodo raíz Comprobante.
func (c *Creator) Comprobante() *Comprobante { return c.comprobante }

// Certificado devuelve el certificado establecido con PutCertificado o nil.
func (c *Creator) Certificado() *credentials.Certificate { return c.certificado }

// PutCertificado establece los atributos NoCertificado y Certificado del
// comprobante. Si putEmisorRfcNombre es verdadero también establece Rfc y
// Nombre del Emisor; para personas morales se elimina el régimen de capital
// del nombre (por ejemplo " SA DE CV") como lo exige CFDI 4.0.
func (c *Creator) PutCertificado(cert *credentials.Certificate, putEmisorRfcNombre bool) {
	c.certificado = cert
	c.comprobante.SetAttr("NoCertificado", cert.NoCertificado())
	c.comprobante.SetAttr("Certificado", cert.Base64())
	if putEmisorRfcNombre {
		c.comprobante.AddEmisor(Attrs{
			"Rfc":    cert.Rfc(),
			"Nombre": cert.LegalNameForCfdi40(),
		})
	}
}

// BuildCadenaDeOrigen genera la cadena original del comprobante en su estado actual.
func (c *Creator) BuildCadenaDeOrigen() (string, error) {
	if c.cadenaOrigen == nil {
		return "", errors.New("cfdi40: no se puede generar la cadena de origen porque no hay un generador")
	}
	cadena, err := c.cadenaOrigen.Build([]byte(c.AsXML()), cadenaorigen.XsltCfdi40)
	if err != nil {
		return "", fmt.Errorf("cfdi40: no se pudo generar la cadena de origen: %w", err)
	}
	return cadena, nil
}

// BuildSumasConceptos calcula las sumas de los conceptos con la precisión dada.
func (c *Creator) BuildSumasConceptos(precision int) *sumasconceptos.SumasConceptos {
	return sumasconceptos.New(c.comprobante, precision)
}

// AddSumasConceptos escribe en el comprobante SubTotal, Descuento, Total, el
// nodo de Impuestos y los totales de impuestos locales. Si sumas es nil se
// calculan con BuildSumasConceptos(precision).
func (c *Creator) AddSumasConceptos(sumas *sumasconceptos.SumasConceptos, precision int) {
	if sumas == nil {
		sumas = c.BuildSumasConceptos(precision)
	}
	WriteSumasConceptos(c.comprobante, sumas, precision)
}

// AddSello genera la cadena original, la firma con la llave privada
// (RSA-SHA256) y guarda el resultado en base64 en el atributo Sello.
//
// Si se estableció un certificado con PutCertificado se verifica que la
// llave le pertenezca.
func (c *Creator) AddSello(key *credentials.PrivateKey) error {
	if key == nil {
		return errors.New("cfdi40: la llave privada es nula")
	}
	if c.certificado != nil && !key.BelongsTo(c.certificado) {
		return ErrKeyDoesNotBelongToCertificate
	}
	cadena, err := c.BuildCadenaDeOrigen()
	if err != nil {
		return err
	}
	signature, err := key.Sign([]byte(cadena))
	if err != nil {
		return fmt.Errorf("cfdi40: no se pudo firmar la cadena de origen: %w", err)
	}
	c.comprobante.SetAttr("Sello", encodeBase64(signature))
	return nil
}

// Sellar establece el certificado de la credencial (incluyendo Rfc y Nombre
// del emisor si putEmisorRfcNombre es verdadero) y genera el sello.
func (c *Creator) Sellar(csd *credentials.Credential, putEmisorRfcNombre bool) error {
	c.PutCertificado(csd.Certificate(), putEmisorRfcNombre)
	return c.AddSello(csd.PrivateKey())
}

// MoveSatDefinitionsToComprobante mueve las declaraciones de espacios de
// nombres (xmlns:*) y ubicaciones de esquemas (xsi:schemaLocation) del SAT
// al nodo raíz, como lo indica la documentación técnica del SAT.
func (c *Creator) MoveSatDefinitionsToComprobante() {
	nodes.MoveSatDefinitionsToRoot(c.comprobante.Node)
}

// Validate valida el comprobante con las validaciones de CFDI 4.0 creados.
func (c *Creator) Validate() *validate.Asserts {
	return c.ValidateContext(context.Background())
}

// ValidateContext es como Validate pero recibe un contexto (las validaciones
// pueden descargar recursos del SAT).
func (c *Creator) ValidateContext(ctx context.Context) *validate.Asserts {
	asserts := validate.NewAsserts()
	cfdi40validators.NewCreatedValidator().Validate(ctx, c.comprobante.Node, c.environment([]byte(c.AsXML())), asserts)
	return asserts
}

// AsXML genera el contenido XML del comprobante.
func (c *Creator) AsXML() string { return c.comprobante.XML() }

// SaveXML genera el XML y lo guarda en un archivo.
func (c *Creator) SaveXML(path string) error {
	if err := os.WriteFile(path, []byte(c.AsXML()), 0o644); err != nil {
		return fmt.Errorf("cfdi40: %w", err)
	}
	return nil
}

// String devuelve el XML del comprobante.
func (c *Creator) String() string { return c.AsXML() }
