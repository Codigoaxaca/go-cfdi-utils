package credentials

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"hash"
	"unicode/utf16"
)

func unmarshal(data []byte, v any) error {
	rest, e := asn1.Unmarshal(data, v)
	if e != nil {
		return e
	}
	if len(rest) != 0 {
		return fmt.Errorf("datos ASN.1 adicionales")
	}
	return nil
}
func decryptPKCS8(der []byte, password string) ([]byte, error) {
	var info struct {
		Algorithm pkix.AlgorithmIdentifier
		Data      []byte
	}
	if e := unmarshal(der, &info); e != nil {
		return nil, fmt.Errorf("credentials: formato de llave inválido: %w", e)
	}
	var block cipher.Block
	var iv []byte
	var e error
	switch info.Algorithm.Algorithm.String() {
	case "1.2.840.113549.1.5.13":
		var p struct{ KDF, Encryption pkix.AlgorithmIdentifier }
		if e = unmarshal(info.Algorithm.Parameters.FullBytes, &p); e != nil {
			break
		}
		if p.KDF.Algorithm.String() != "1.2.840.113549.1.5.12" {
			e = fmt.Errorf("KDF no soportado: %s", p.KDF.Algorithm)
			break
		}
		var kdf struct {
			Salt       []byte
			Iterations int
			Length     int                      `asn1:"optional"`
			PRF        pkix.AlgorithmIdentifier `asn1:"optional"`
		}
		if e = unmarshal(p.KDF.Parameters.FullBytes, &kdf); e != nil {
			break
		}
		var hf func() hash.Hash
		switch kdf.PRF.Algorithm.String() {
		case "", "1.2.840.113549.2.7":
			hf = sha1.New
		case "1.2.840.113549.2.9":
			hf = sha256.New
		default:
			e = fmt.Errorf("PRF no soportada: %s", kdf.PRF.Algorithm)
		}
		if e != nil {
			break
		}
		size := 0
		var makeBlock func([]byte) (cipher.Block, error)
		switch p.Encryption.Algorithm.String() {
		case "1.2.840.113549.3.7":
			size = 24
			makeBlock = des.NewTripleDESCipher
		case "1.3.14.3.2.7":
			size = 8
			makeBlock = des.NewCipher
		case "2.16.840.1.101.3.4.1.2":
			size = 16
			makeBlock = aes.NewCipher
		case "2.16.840.1.101.3.4.1.22":
			size = 24
			makeBlock = aes.NewCipher
		case "2.16.840.1.101.3.4.1.42":
			size = 32
			makeBlock = aes.NewCipher
		default:
			e = fmt.Errorf("cifrado no soportado: %s", p.Encryption.Algorithm)
		}
		if e != nil {
			break
		}
		if kdf.Length != 0 && kdf.Length != size {
			e = fmt.Errorf("longitud de llave incompatible")
			break
		}
		if kdf.Iterations < 1 || kdf.Iterations > 10000000 {
			e = fmt.Errorf("iteraciones PBKDF2 fuera de rango")
			break
		}
		var key []byte
		key, e = pbkdf2.Key(hf, password, kdf.Salt, kdf.Iterations, size)
		if e != nil {
			break
		}
		block, e = makeBlock(key)
		if e != nil {
			break
		}
		e = unmarshal(p.Encryption.Parameters.FullBytes, &iv)
	case "1.2.840.113549.1.12.1.3":
		var p struct {
			Salt       []byte
			Iterations int
		}
		if e = unmarshal(info.Algorithm.Parameters.FullBytes, &p); e != nil {
			break
		}
		if p.Iterations < 1 || p.Iterations > 10000000 {
			e = fmt.Errorf("iteraciones PKCS12 fuera de rango")
			break
		}
		key := pkcs12KDF(password, p.Salt, p.Iterations, 1, 24)
		iv = pkcs12KDF(password, p.Salt, p.Iterations, 2, 8)
		block, e = des.NewTripleDESCipher(key)
	default:
		e = fmt.Errorf("algoritmo de cifrado no soportado: %s", info.Algorithm.Algorithm)
	}
	if e != nil {
		return nil, fmt.Errorf("credentials: descifrar PKCS#8: %w", e)
	}
	if block == nil || len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("credentials: vector de inicialización inválido")
	}
	if len(info.Data) == 0 || len(info.Data)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("%w: tamaño de cifrado inválido", ErrInvalidPassword)
	}
	plain := make([]byte, len(info.Data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, info.Data)
	padding := int(plain[len(plain)-1])
	if padding < 1 || padding > block.BlockSize() || !bytes.Equal(plain[len(plain)-padding:], bytes.Repeat([]byte{byte(padding)}, padding)) {
		return nil, ErrInvalidPassword
	}
	return plain[:len(plain)-padding], nil
}

// PKCS#12 B.2: SHA-1, u=20, v=64 y contraseña UTF-16BE terminada en cero.
func pkcs12KDF(password string, salt []byte, iterations, id, size int) []byte {
	words := utf16.Encode([]rune(password))
	p := make([]byte, 2*(len(words)+1))
	for i, w := range words {
		p[2*i] = byte(w >> 8)
		p[2*i+1] = byte(w)
	}
	repeat := func(b []byte) []byte {
		if len(b) == 0 {
			return nil
		}
		out := make([]byte, 64*((len(b)+63)/64))
		for i := range out {
			out[i] = b[i%len(b)]
		}
		return out
	}
	state := append(repeat(salt), repeat(p)...)
	d := bytes.Repeat([]byte{byte(id)}, 64)
	out := []byte{}
	for len(out) < size {
		h := sha1.New()
		h.Write(d)
		h.Write(state)
		a := h.Sum(nil)
		for j := 1; j < iterations; j++ {
			sum := sha1.Sum(a)
			a = sum[:]
		}
		out = append(out, a...)
		b := make([]byte, 64)
		for i := range b {
			b[i] = a[i%20]
		}
		for off := 0; off < len(state); off += 64 {
			carry := 1
			for j := 63; j >= 0; j-- {
				n := int(state[off+j]) + int(b[j]) + carry
				state[off+j] = byte(n)
				carry = n >> 8
			}
		}
	}
	return out[:size]
}
