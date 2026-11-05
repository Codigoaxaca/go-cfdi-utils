package credentials_test

import (
	"fmt"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
)

func ExampleOpenFiles() {
	cred, err := credentials.OpenFiles("../testdata/certs/EKU9003173C9.cer", "../testdata/certs/EKU9003173C9.key", "12345678a")
	if err != nil {
		panic(err)
	}
	fmt.Println(cred.Rfc())
	// Output: EKU9003173C9
}

func ExampleCertificate_NoCertificado() {
	cert, err := credentials.OpenCertificateFile("../testdata/certs/EKU9003173C9.cer")
	if err != nil {
		panic(err)
	}
	fmt.Println(cert.NoCertificado())
	// Output: 30001000000500003416
}

func ExampleCredential_Sign() {
	cred, err := credentials.OpenFiles("../testdata/certs/EKU9003173C9.cer", "../testdata/certs/EKU9003173C9.key", "12345678a")
	if err != nil {
		panic(err)
	}
	cadena := []byte("||4.0|...||")
	sello, err := cred.Sign(cadena)
	if err != nil {
		panic(err)
	}
	fmt.Println(cred.Verify(cadena, sello))
	// Output: true
}

func ExampleParseCertificate() {
	original, err := credentials.OpenCertificateFile("../testdata/certs/EKU9003173C9.cer")
	if err != nil {
		panic(err)
	}
	// En una factura, este texto proviene de Comprobante@Certificado.
	atributoCertificado := original.Base64()
	cert, err := credentials.ParseCertificate([]byte(atributoCertificado))
	if err != nil {
		panic(err)
	}
	fmt.Println(cert.Rfc())
	// Output: EKU9003173C9
}
