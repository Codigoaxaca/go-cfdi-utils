package credentials

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// ErrInvalidPassword indica contraseña incorrecta o contenido cifrado dañado.
var ErrInvalidPassword = errors.New("credentials: contraseña inválida o llave cifrada dañada")

// PrivateKey contiene una llave privada RSA del SAT.
type PrivateKey struct{ key *rsa.PrivateKey }

// OpenPrivateKeyFile abre una llave DER, PEM o base64, descifrándola si hace falta.
func OpenPrivateKeyFile(path, password string) (*PrivateKey, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, fmt.Errorf("credentials: leer llave: %w", e)
	}
	return ParsePrivateKey(b, password)
}

// ParsePrivateKey acepta PKCS#1, PKCS#8 y PKCS#8 cifrado PBES2 o PKCS#12
// SHA1/3DES, además del cifrado PEM tradicional. Sólo admite llaves RSA.
func ParsePrivateKey(data []byte, password string) (*PrivateKey, error) {
	der, block, e := decodeInput(data, "PRIVATE KEY", "ENCRYPTED PRIVATE KEY", "RSA PRIVATE KEY")
	if e != nil {
		return nil, e
	}
	encrypted := false
	if block != nil && x509.IsEncryptedPEMBlock(block) {
		encrypted = true
		der, e = x509.DecryptPEMBlock(block, []byte(password))
		if e != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidPassword, e)
		}
	}
	if key, e := parseRSA(der); e == nil {
		return &PrivateKey{key}, nil
	}
	if encrypted {
		return nil, ErrInvalidPassword
	}
	plain, e := decryptPKCS8(der, password)
	if e != nil {
		return nil, e
	}
	key, e := parseRSA(plain)
	if e != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPassword, e)
	}
	return &PrivateKey{key}, nil
}
func parseRSA(der []byte) (*rsa.PrivateKey, error) {
	if key, e := x509.ParsePKCS1PrivateKey(der); e == nil {
		return key, key.Validate()
	}
	k, e := x509.ParsePKCS8PrivateKey(der)
	if e != nil {
		return nil, e
	}
	key, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("credentials: la llave no es RSA")
	}
	return key, key.Validate()
}

// Sign firma los datos con RSA PKCS#1 v1.5 y SHA-256.
func (k *PrivateKey) Sign(data []byte) ([]byte, error) { return k.SignWithHash(crypto.SHA256, data) }

// SignWithHash firma los datos con el hash indicado.
func (k *PrivateKey) SignWithHash(h crypto.Hash, data []byte) ([]byte, error) {
	if k == nil || k.key == nil {
		return nil, fmt.Errorf("credentials: llave vacía")
	}
	if !h.Available() {
		return nil, fmt.Errorf("credentials: hash no disponible: %v", h)
	}
	d := h.New()
	d.Write(data)
	b, e := rsa.SignPKCS1v15(rand.Reader, k.key, h, d.Sum(nil))
	if e != nil {
		return nil, fmt.Errorf("credentials: firmar: %w", e)
	}
	return b, nil
}

// BelongsTo compara las llaves públicas del certificado y la llave privada.
func (k *PrivateKey) BelongsTo(c *Certificate) bool {
	if k == nil || k.key == nil || c == nil {
		return false
	}
	p, ok := c.public.(*rsa.PublicKey)
	return ok && k.key.PublicKey.Equal(p)
}

// PEM devuelve PKCS#8 sin cifrar; contiene material secreto.
func (k *PrivateKey) PEM() string {
	b, e := x509.MarshalPKCS8PrivateKey(k.key)
	if e != nil {
		return ""
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b}))
}

// RSA devuelve la llave RSA; el llamante no debe modificarla durante su uso.
func (k *PrivateKey) RSA() *rsa.PrivateKey { return k.key }
