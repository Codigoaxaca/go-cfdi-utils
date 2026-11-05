package credentials

import (
	"encoding/base64"
	"fmt"
)

// Credential reúne un certificado y su llave privada correspondiente.
type Credential struct {
	cert *Certificate
	key  *PrivateKey
}

// OpenFiles abre los archivos SAT directamente, sin conversiones externas.
func OpenFiles(cerPath, keyPath, password string) (*Credential, error) {
	c, e := OpenCertificateFile(cerPath)
	if e != nil {
		return nil, e
	}
	k, e := OpenPrivateKeyFile(keyPath, password)
	if e != nil {
		return nil, e
	}
	return NewCredential(c, k)
}

// NewCredential valida que la llave corresponda al certificado.
func NewCredential(cert *Certificate, key *PrivateKey) (*Credential, error) {
	if !key.BelongsTo(cert) {
		return nil, fmt.Errorf("credentials: la llave no corresponde al certificado")
	}
	return &Credential{cert, key}, nil
}

// ParseCredential abre certificado y llave desde memoria y valida su correspondencia.
func ParseCredential(cerData, keyData []byte, password string) (*Credential, error) {
	c, e := ParseCertificate(cerData)
	if e != nil {
		return nil, e
	}
	k, e := ParsePrivateKey(keyData, password)
	if e != nil {
		return nil, e
	}
	return NewCredential(c, k)
}

// Certificate devuelve el certificado.
func (c *Credential) Certificate() *Certificate { return c.cert }

// PrivateKey devuelve la llave privada.
func (c *Credential) PrivateKey() *PrivateKey { return c.key }

// Rfc devuelve el RFC del titular.
func (c *Credential) Rfc() string { return c.cert.Rfc() }

// Name devuelve el nombre legal del titular.
func (c *Credential) Name() string { return c.cert.Name() }

// NoCertificado devuelve el número SAT en ASCII.
func (c *Credential) NoCertificado() string { return c.cert.NoCertificado() }

// Sign firma una cadena original con SHA-256 y RSA PKCS#1 v1.5.
func (c *Credential) Sign(data []byte) ([]byte, error) { return c.key.Sign(data) }

// SignBase64 devuelve el sello listo para Comprobante@Sello.
func (c *Credential) SignBase64(data []byte) (string, error) {
	b, e := c.Sign(data)
	if e != nil {
		return "", e
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// Verify verifica una firma RSA SHA-256.
func (c *Credential) Verify(data, signature []byte) bool { return c.cert.Verify(data, signature) }

// IsCsd identifica el perfil de usos de llave CSD.
func (c *Credential) IsCsd() bool { return c.cert.IsCsd() }

// IsFiel identifica el perfil de usos de llave e.firma.
func (c *Credential) IsFiel() bool { return c.cert.IsFiel() }
