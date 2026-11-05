package credentials

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const assets = "../testdata/certs/"

func read(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func certificate(t *testing.T) *Certificate {
	t.Helper()
	c, e := OpenCertificateFile(assets + "EKU9003173C9.cer")
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestCertificates(t *testing.T) {
	paths, e := filepath.Glob(assets + "*.cer")
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			c, e := OpenCertificateFile(p)
			if e != nil {
				t.Fatal(e)
			}
			if c.NoCertificado() == "" || c.ValidFrom().IsZero() || c.ValidTo().IsZero() || c.PublicKeyPEM() == "" {
				t.Fatal("certificado incompleto")
			}
			t.Log(c.Rfc(), c.Name(), c.NoCertificado(), "fallback", c.X509() == nil)
		})
	}
	c := certificate(t)
	// Valores obtenidos con openssl x509 -inform DER -noout -subject -serial -dates.
	if c.Rfc() != "EKU9003173C9" || c.RfcRepresentante() != "VADA800927DJ3" || c.Curp() != "" || c.Name() != "ESCUELA KEMPER URGATE SA DE CV" || c.NoCertificado() != "30001000000500003416" {
		t.Fatal("sujeto inesperado", c)
	}
	if c.ValidFrom().Format(time.RFC3339) != "2023-05-18T11:43:51Z" || c.ValidTo().Format(time.RFC3339) != "2027-05-18T11:43:51Z" {
		t.Fatal("vigencia incorrecta")
	}
	if !c.IsValidAt(c.ValidFrom()) || !c.IsValidAt(c.ValidTo()) || c.IsValidAt(c.ValidFrom().Add(-time.Second)) || c.IsValidAt(c.ValidTo().Add(time.Second)) {
		t.Fatal("límites de vigencia")
	}
	if !c.IsCsd() || c.IsFiel() || !c.IsPersonaMoral() || c.LegalNameForCfdi40() != "ESCUELA KEMPER URGATE" || c.NameWithoutRegimenCapital() != "ESCUELA KEMPER URGATE" || c.Issuer() == "" {
		t.Fatal("perfil inesperado")
	}
	for _, data := range [][]byte{read(t, assets+"EKU9003173C9.cer.pem"), []byte(c.Base64()), []byte(" \n" + c.Base64()[:32] + "\n" + c.Base64()[32:]), []byte(c.PublicKeyPEM() + c.PEM() + c.PublicKeyPEM())} {
		other, e := ParseCertificate(data)
		if e != nil || !bytes.Equal(other.DER(), c.DER()) {
			t.Fatalf("formato: %v", e)
		}
	}
	if _, e := ParseCertificate(read(t, assets+"certificate-with-error.pem")); e == nil || !strings.Contains(e.Error(), "credentials:") {
		t.Fatal("aceptó certificado inválido")
	}
	for _, data := range [][]byte{nil, []byte("bad"), append(c.DER(), 0)} {
		if _, e := ParseCertificate(data); e == nil {
			t.Fatal("aceptó datos inválidos")
		}
	}
	path := filepath.Join(t.TempDir(), "cert.cer")
	if e := c.SaveFile(path); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(read(t, path), c.DER()) {
		t.Fatal("guardar")
	}
	b := c.DER()
	b[0] = 0
	if c.DER()[0] == 0 {
		t.Fatal("DER no es copia")
	}
}
func TestSerial(t *testing.T) {
	const h = "3330303031303030303030333030303233373038"
	const a = "30001000000300023708"
	const d = "292233162870206001759766198425879490508935868472"
	x, e := SerialNumberFromHex(h)
	if e != nil {
		t.Fatal(e)
	}
	y, e := SerialNumberFromDecimal(d)
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range []SerialNumber{x, y, SerialNumberFromAscii(a)} {
		if s.Hexadecimal() != h || s.Ascii() != a || s.Decimal() != d || s.String() != a {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"", "-1", "0x12", "zz"} {
		if _, e := SerialNumberFromHex(s); e == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"", "-1", "+1", "1a"} {
		if _, e := SerialNumberFromDecimal(s); e == nil {
			t.Fatal(s)
		}
	}
	z, e := SerialNumberFromHex("0001")
	if e != nil || z.Hexadecimal() != "0001" || z.Decimal() != "1" {
		t.Fatal(z, e)
	}
	if !IsValidNoCertificado(a) || IsValidNoCertificado(a+"\n") || IsValidNoCertificado("１２３４５６７８９０１２３４５６７８９０") {
		t.Fatal("validación de número")
	}
	url, e := SatCertificateURL(a)
	if e != nil || url != "https://rdc.sat.gob.mx/rccf/300010/000003/00/02/37/30001000000300023708.cer" {
		t.Fatal(url, e)
	}
	if _, e := SatCertificateURL("bad"); e == nil {
		t.Fatal("número inválido")
	}
}
func TestKeysAndSignatures(t *testing.T) {
	c := certificate(t)
	data := read(t, assets+"data-to-sign.txt")
	want := read(t, assets+"data-sha256.bin")
	other, e := OpenCertificateFile(assets + "00001000000403258748.cer")
	if e != nil {
		t.Fatal(e)
	}
	paths := []string{assets + "EKU9003173C9.key", assets + "EKU9003173C9.key.pem", assets + "EKU9003173C9_password.key.pem"}
	extra, _ := filepath.Glob("testdata/*.key")
	paths = append(paths, extra...)
	legacy, _ := filepath.Glob("testdata/legacy-*.pem")
	paths = append(paths, legacy...)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			key, e := OpenPrivateKeyFile(path, "12345678a")
			if e != nil {
				t.Fatal(e)
			}
			if !key.BelongsTo(c) || key.BelongsTo(other) {
				t.Fatal("correspondencia")
			}
			sig, e := key.Sign(data)
			if e != nil || !bytes.Equal(sig, want) {
				t.Fatal("firma OpenSSL diferente", e)
			}
			if !c.Verify(data, sig) || c.Verify(append(data, 0), sig) || c.Verify(data, []byte("invalid")) || c.VerifyWithHash(crypto.Hash(0), data, sig) {
				t.Fatal("verificación")
			}
			if _, e := key.SignWithHash(crypto.Hash(0), data); e == nil {
				t.Fatal("hash inválido")
			}
			for _, h := range []crypto.Hash{crypto.SHA1, crypto.SHA384, crypto.SHA512} {
				s, e := key.SignWithHash(h, data)
				if e != nil || !c.VerifyWithHash(h, data, s) {
					t.Fatal("hash", h, e)
				}
			}
			for _, b := range [][]byte{[]byte(key.PEM()), x509.MarshalPKCS1PrivateKey(key.RSA()), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key.RSA())})} {
				if _, e := ParsePrivateKey(b, ""); e != nil {
					t.Fatal(e)
				}
			}
			if path != assets+"EKU9003173C9.key.pem" {
				if _, e := OpenPrivateKeyFile(path, "wrong"); !errors.Is(e, ErrInvalidPassword) {
					t.Fatalf("contraseña incorrecta: %v", e)
				}
			}
		})
	}
	if _, e := ParsePrivateKey([]byte(base64.StdEncoding.EncodeToString(read(t, assets+"EKU9003173C9.key"))), "12345678a"); e != nil {
		t.Fatal(e)
	}
	cr, e := OpenFiles(assets+"EKU9003173C9.cer", assets+"EKU9003173C9.key", "12345678a")
	if e != nil {
		t.Fatal(e)
	}
	b64, e := cr.SignBase64(data)
	if e != nil || b64 != strings.TrimSpace(string(read(t, assets+"data-sha256.txt"))) { // openssl base64 may wrap lines
		expected := strings.Join(strings.Fields(string(read(t, assets+"data-sha256.txt"))), "")
		if e != nil || b64 != expected {
			t.Fatal("sello", e)
		}
	}
	if cr.Certificate() != nil && cr.PrivateKey() != nil {
		if cr.Rfc() != c.Rfc() || cr.Name() != c.Name() || cr.NoCertificado() != c.NoCertificado() || !cr.IsCsd() || cr.IsFiel() || !cr.Verify(data, want) {
			t.Fatal("delegación")
		}
	} else {
		t.Fatal("credencial vacía")
	}
	if _, e := NewCredential(other, cr.PrivateKey()); e == nil {
		t.Fatal("aceptó llave ajena")
	}
	if _, e := NewCredential(nil, nil); e == nil {
		t.Fatal("aceptó nil")
	}
	if _, e := ParseCredential(c.DER(), []byte(cr.PrivateKey().PEM()), ""); e != nil {
		t.Fatal(e)
	}
}
func TestRegimen(t *testing.T) {
	cases := map[string]string{"ESCUELA KEMPER URGATE SA DE CV": "ESCUELA KEMPER URGATE", "Empresa sa de cv": "Empresa", "Empresa   S DE RL DE CV": "Empresa", "Empresa SCL (LIMITADA)": "Empresa", "Empresa SOCIEDADES DE INVERSIÓN": "Empresa", "Empresa SAPI DE CV,SOFOM,ENR": "Empresa", "ROSA": "ROSA", "SA": "SA", "Empresa SA ": "Empresa SA ", "Empresa AC": "Empresa"}
	for in, want := range cases {
		if got := RemoveRegimenCapital(in); got != want {
			t.Errorf("%q: %q != %q", in, got, want)
		}
	}
}
func TestFallbackAndProfiles(t *testing.T) {
	// Extensión básica deliberadamente inválida: X509 la rechaza, ASN.1 conserva campos necesarios.
	key, e := rsa.GenerateKey(rand.Reader, 1024)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC().Truncate(time.Second)
	for _, usage := range []x509.KeyUsage{x509.KeyUsageDigitalSignature | x509.KeyUsageContentCommitment, x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment, 0} {
		template := &x509.Certificate{SerialNumber: big.NewInt(123), Subject: pkix.Name{CommonName: "Fallback SA DE CV", ExtraNames: []pkix.AttributeTypeAndValue{{Type: asn1.ObjectIdentifier{2, 5, 4, 45}, Value: "AAA010101AAA"}}}, NotBefore: now, NotAfter: now.Add(time.Hour), KeyUsage: usage, ExtraExtensions: []pkix.Extension{{Id: asn1.ObjectIdentifier{2, 5, 29, 19}, Value: []byte{1, 2, 3}}}}
		der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if e != nil {
			t.Fatal(e)
		}
		c, e := ParseCertificate(der)
		if e != nil {
			t.Fatal(e)
		}
		if c.X509() != nil || c.Rfc() != "AAA010101AAA" || c.LegalNameForCfdi40() != "Fallback" || !c.ValidFrom().Equal(now) || !(&PrivateKey{key}).BelongsTo(c) {
			t.Fatal("fallback incompleto")
		}
		if c.IsCsd() != (usage == 3) || c.IsFiel() != (usage&x509.KeyUsageDataEncipherment != 0) {
			t.Fatal("perfil")
		}
	}
}
