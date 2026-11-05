package pagos20

import (
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/internal/testresources"
)

// Las pruebas usan los recursos del SAT del respaldo incluido, sin red.
func TestMain(m *testing.M) { testresources.Main(m) }
