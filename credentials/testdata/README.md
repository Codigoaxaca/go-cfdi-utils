# Llaves de prueba

Estas llaves son variantes cifradas de la **llave pública de pruebas del SAT**
`../../testdata/certs/EKU9003173C9.key.pem`. No son credenciales de producción.
La contraseña de todos los archivos es `12345678a`.

Se generaron con OpenSSL 3.6.2, ejecutando desde la raíz del repositorio:

```sh
openssl pkcs8 -topk8 -in testdata/certs/EKU9003173C9.key.pem -passout pass:12345678a -outform DER -v2 aes-128-cbc -v2prf hmacWithSHA1 -out credentials/testdata/aes128-sha1.key
openssl pkcs8 -topk8 -in testdata/certs/EKU9003173C9.key.pem -passout pass:12345678a -outform DER -v2 aes-192-cbc -v2prf hmacWithSHA256 -out credentials/testdata/aes192-sha256.key
openssl pkcs8 -topk8 -in testdata/certs/EKU9003173C9.key.pem -passout pass:12345678a -outform DER -v2 aes-256-cbc -v2prf hmacWithSHA256 -out credentials/testdata/aes256-sha256.key
openssl pkcs8 -topk8 -in testdata/certs/EKU9003173C9.key.pem -passout pass:12345678a -outform DER -v2 des3 -out credentials/testdata/des3.key
openssl pkcs8 -topk8 -in testdata/certs/EKU9003173C9.key.pem -passout pass:12345678a -outform DER -provider default -provider legacy -v2 des -out credentials/testdata/des.key
openssl pkcs8 -topk8 -in testdata/certs/EKU9003173C9.key.pem -passout pass:12345678a -outform DER -v1 PBE-SHA1-3DES -out credentials/testdata/pkcs12-des3.key
openssl rsa -in testdata/certs/EKU9003173C9.key.pem -traditional -des3 -passout pass:12345678a -out credentials/testdata/legacy-des3.pem
```

Las pruebas leen estos archivos estáticos; no necesitan OpenSSL instalado.

También se generaron `legacy-des.pem`, `legacy-aes128.pem`, `legacy-aes192.pem`
y `legacy-aes256.pem` sustituyendo `-des3` en el comando PEM tradicional por
`-des` (con `-provider default -provider legacy`), `-aes128`, `-aes192` y
`-aes256`, respectivamente. Verifican todos los cifrados PEM tradicionales
que expone la biblioteca estándar.
