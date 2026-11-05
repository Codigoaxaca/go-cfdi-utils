// Package credentials abre certificados y llaves del SAT y firma cadenas originales.
// No verifica cadenas de confianza ni revocación; IsValidAt sólo revisa las fechas.
package credentials

import (
	"crypto"
	"crypto/rsa"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
)

// Certificate contiene un certificado CSD, e.firma o SAT.
type Certificate struct {
	der             []byte
	cert            *x509.Certificate
	subject, issuer pkix.Name
	serial          SerialNumber
	from, to        time.Time
	public          any
	usage           x509.KeyUsage
}

// OpenCertificateFile abre un certificado DER, PEM o base64.
func OpenCertificateFile(path string) (*Certificate, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, fmt.Errorf("credentials: leer certificado: %w", e)
	}
	return ParseCertificate(b)
}
func decodeInput(data []byte, types ...string) ([]byte, *pem.Block, error) {
	if strings.Contains(string(data), "-----BEGIN ") {
		rest := data
		for {
			b, r := pem.Decode(rest)
			if b == nil {
				break
			}
			for _, t := range types {
				if b.Type == t {
					return b.Bytes, b, nil
				}
			}
			rest = r
		}
		return nil, nil, fmt.Errorf("credentials: bloque PEM requerido no encontrado")
	}
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, string(data))
	if b, e := base64.StdEncoding.DecodeString(compact); e == nil && len(b) > 0 {
		return b, nil, nil
	}
	return data, nil, nil
}

// ParseCertificate acepta DER, PEM (incluso con otros bloques) o el atributo Certificado en base64.
func ParseCertificate(data []byte) (*Certificate, error) {
	der, _, e := decodeInput(data, "CERTIFICATE")
	if e != nil {
		return nil, e
	}
	c := &Certificate{der: append([]byte(nil), der...)}
	x, e := x509.ParseCertificate(der)
	if e == nil {
		c.cert = x
		c.subject = x.Subject
		c.issuer = x.Issuer
		c.from = x.NotBefore
		c.to = x.NotAfter
		c.public = x.PublicKey
		c.usage = x.KeyUsage
		c.serial, _ = SerialNumberFromHex(x.SerialNumber.Text(16))
		return c, nil
	}
	if err := c.parseFallback(); err != nil {
		return nil, fmt.Errorf("credentials: certificado inválido: %w (X509: %v)", err, e)
	}
	return c, nil
}

// parseFallback tolera extensiones antiguas que crypto/x509 rechaza, sin validar confianza.
func (c *Certificate) parseFallback() error {
	var outer struct {
		TBS       asn1.RawValue
		Algorithm pkix.AlgorithmIdentifier
		Signature asn1.BitString
	}
	rest, e := asn1.Unmarshal(c.der, &outer)
	if e != nil {
		return e
	}
	if len(rest) != 0 {
		return fmt.Errorf("datos adicionales")
	}
	var t struct {
		Version    int `asn1:"optional,explicit,tag:0,default:0"`
		Serial     asn1.RawValue
		Algorithm  pkix.AlgorithmIdentifier
		Issuer     asn1.RawValue
		Validity   struct{ From, To time.Time }
		Subject    asn1.RawValue
		Public     asn1.RawValue
		IssuerID   asn1.BitString   `asn1:"optional,tag:1"`
		SubjectID  asn1.BitString   `asn1:"optional,tag:2"`
		Extensions []pkix.Extension `asn1:"optional,explicit,tag:3"`
	}
	rest, e = asn1.Unmarshal(outer.TBS.FullBytes, &t)
	if e != nil {
		return e
	}
	if len(rest) > 0 || t.Serial.Tag != asn1.TagInteger || len(t.Serial.Bytes) == 0 {
		return fmt.Errorf("estructura X509 inválida")
	}
	for _, pair := range []struct {
		raw  asn1.RawValue
		name *pkix.Name
	}{{t.Subject, &c.subject}, {t.Issuer, &c.issuer}} {
		var seq pkix.RDNSequence
		if _, e = asn1.Unmarshal(pair.raw.FullBytes, &seq); e != nil {
			return e
		}
		pair.name.FillFromRDNSequence(&seq)
	}
	c.public, e = x509.ParsePKIXPublicKey(t.Public.FullBytes)
	if e != nil {
		return e
	}
	b := t.Serial.Bytes
	if len(b) > 1 && b[0] == 0 {
		b = b[1:]
	}
	c.serial = SerialNumberFromAscii(string(b))
	c.from = t.Validity.From
	c.to = t.Validity.To
	for _, ext := range t.Extensions {
		if ext.Id.Equal(asn1.ObjectIdentifier{2, 5, 29, 15}) {
			var bits asn1.BitString
			if _, e = asn1.Unmarshal(ext.Value, &bits); e != nil {
				return e
			}
			for i := 0; i < 9; i++ {
				if bits.At(i) != 0 {
					c.usage |= 1 << i
				}
			}
		}
	}
	return nil
}
func attribute(n pkix.Name, oid string) string {
	for _, a := range n.Names {
		if a.Type.String() == oid {
			if s, ok := a.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}
func token(s string, i int) string {
	parts := strings.Split(s, " / ")
	if i >= len(parts) {
		return ""
	}
	return strings.TrimSpace(parts[i])
}

// Rfc devuelve el RFC del titular.
func (c *Certificate) Rfc() string { return token(attribute(c.subject, "2.5.4.45"), 0) }

// RfcRepresentante devuelve el RFC del representante, si existe.
func (c *Certificate) RfcRepresentante() string { return token(attribute(c.subject, "2.5.4.45"), 1) }

// Curp devuelve el primer componente de serialNumber del sujeto.
func (c *Certificate) Curp() string { return token(attribute(c.subject, "2.5.4.5"), 0) }

// Name devuelve name, o en su ausencia O y CN.
func (c *Certificate) Name() string {
	if s := attribute(c.subject, "2.5.4.41"); s != "" {
		return s
	}
	if len(c.subject.Organization) > 0 {
		return c.subject.Organization[0]
	}
	return c.subject.CommonName
}

// NameWithoutRegimenCapital devuelve el nombre sin sufijo societario.
func (c *Certificate) NameWithoutRegimenCapital() string { return RemoveRegimenCapital(c.Name()) }

// LegalNameForCfdi40 devuelve el nombre requerido en Emisor@Nombre.
func (c *Certificate) LegalNameForCfdi40() string {
	if c.IsPersonaMoral() {
		return c.NameWithoutRegimenCapital()
	}
	return c.Name()
}

// IsPersonaMoral identifica RFC de doce caracteres.
func (c *Certificate) IsPersonaMoral() bool { return len(c.Rfc()) == 12 }

// SerialNumber devuelve el número de serie.
func (c *Certificate) SerialNumber() SerialNumber { return c.serial }

// NoCertificado devuelve la representación ASCII del número de serie.
func (c *Certificate) NoCertificado() string { return c.serial.Ascii() }

// ValidFrom devuelve el inicio de vigencia.
func (c *Certificate) ValidFrom() time.Time { return c.from }

// ValidTo devuelve el final de vigencia.
func (c *Certificate) ValidTo() time.Time { return c.to }

// IsValidAt comprueba los límites inclusivos de vigencia, sin validar confianza.
func (c *Certificate) IsValidAt(t time.Time) bool { return !t.Before(c.from) && !t.After(c.to) }

// IsCsd identifica el perfil SAT con digitalSignature y nonRepudiation, sin
// dataEncipherment ni keyAgreement. No acredita autorización del SAT.
func (c *Certificate) IsCsd() bool {
	return c.usage&(x509.KeyUsageDigitalSignature|x509.KeyUsageContentCommitment) == (x509.KeyUsageDigitalSignature|x509.KeyUsageContentCommitment) && c.usage&(x509.KeyUsageDataEncipherment|x509.KeyUsageKeyAgreement) == 0
}

// IsFiel identifica el perfil de e.firma con dataEncipherment o keyAgreement.
// Un certificado ajeno a ambos perfiles puede devolver false en ambos métodos.
func (c *Certificate) IsFiel() bool {
	return c.usage&(x509.KeyUsageDataEncipherment|x509.KeyUsageKeyAgreement) != 0
}

// Issuer devuelve el CN del emisor, o su nombre completo si no tiene CN.
func (c *Certificate) Issuer() string {
	if c.issuer.CommonName != "" {
		return c.issuer.CommonName
	}
	return c.issuer.String()
}

// DER devuelve una copia del certificado binario.
func (c *Certificate) DER() []byte { return append([]byte(nil), c.der...) }

// PEM devuelve un bloque CERTIFICATE con líneas de 64 columnas.
func (c *Certificate) PEM() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.der}))
}

// Base64 devuelve el atributo Certificado sin saltos de línea.
func (c *Certificate) Base64() string { return base64.StdEncoding.EncodeToString(c.der) }

// PublicKeyPEM devuelve la llave pública en formato PKIX PEM.
func (c *Certificate) PublicKeyPEM() string {
	b, e := x509.MarshalPKIXPublicKey(c.public)
	if e != nil {
		return ""
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: b}))
}

// X509 devuelve el certificado estándar; es nil si se usó el lector ASN.1 tolerante.
func (c *Certificate) X509() *x509.Certificate { return c.cert }

// Verify verifica RSA PKCS#1 v1.5 con SHA-256.
func (c *Certificate) Verify(data, signature []byte) bool {
	return c.VerifyWithHash(crypto.SHA256, data, signature)
}

// VerifyWithHash verifica RSA PKCS#1 v1.5 con el hash indicado.
func (c *Certificate) VerifyWithHash(h crypto.Hash, data, signature []byte) bool {
	key, ok := c.public.(*rsa.PublicKey)
	if !ok || !h.Available() {
		return false
	}
	digest := h.New()
	digest.Write(data)
	return rsa.VerifyPKCS1v15(key, h, digest.Sum(nil), signature) == nil
}

// SaveFile guarda el certificado DER con permisos 0644.
func (c *Certificate) SaveFile(path string) error {
	if e := os.WriteFile(path, c.der, 0644); e != nil {
		return fmt.Errorf("credentials: guardar certificado: %w", e)
	}
	return nil
}
