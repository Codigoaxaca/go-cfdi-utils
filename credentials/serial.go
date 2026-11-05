package credentials

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// SerialNumber representa el número de serie sin perder ceros iniciales.
type SerialNumber struct{ hex string }

// SerialNumberFromHex interpreta una representación hexadecimal sin prefijo.
func SerialNumberFromHex(s string) (SerialNumber, error) {
	if s == "" {
		return SerialNumber{}, fmt.Errorf("credentials: número hexadecimal vacío")
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return SerialNumber{}, fmt.Errorf("credentials: número hexadecimal inválido")
		}
	}
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return SerialNumber{strings.ToLower(s)}, nil
}

// SerialNumberFromDecimal interpreta un entero decimal no negativo.
func SerialNumberFromDecimal(s string) (SerialNumber, error) {
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(s) {
		return SerialNumber{}, fmt.Errorf("credentials: número decimal inválido")
	}
	n, _ := new(big.Int).SetString(s, 10)
	return SerialNumberFromHex(n.Text(16))
}

// SerialNumberFromAscii interpreta los bytes ASCII del número SAT.
func SerialNumberFromAscii(s string) SerialNumber { return SerialNumber{hex.EncodeToString([]byte(s))} }

// Hexadecimal devuelve la representación hexadecimal.
func (s SerialNumber) Hexadecimal() string { return s.hex }

// Ascii devuelve los bytes originales como texto.
func (s SerialNumber) Ascii() string { b, _ := hex.DecodeString(s.hex); return string(b) }

// Decimal devuelve la representación decimal.
func (s SerialNumber) Decimal() string {
	n, ok := new(big.Int).SetString(s.hex, 16)
	if !ok {
		return "0"
	}
	return n.String()
}

// String devuelve Ascii.
func (s SerialNumber) String() string { return s.Ascii() }

var certificateNumberPattern = regexp.MustCompile(`^[0-9]{20}$`)

// IsValidNoCertificado indica si hay exactamente veinte dígitos ASCII.
func IsValidNoCertificado(n string) bool { return certificateNumberPattern.MatchString(n) }

// SatCertificateURL construye la dirección del repositorio de certificados SAT.
func SatCertificateURL(n string) (string, error) {
	if !IsValidNoCertificado(n) {
		return "", fmt.Errorf("credentials: número de certificado inválido")
	}
	return fmt.Sprintf("https://rdc.sat.gob.mx/rccf/%s/%s/%s/%s/%s/%s.cer", n[:6], n[6:12], n[12:14], n[14:16], n[16:18], n), nil
}
