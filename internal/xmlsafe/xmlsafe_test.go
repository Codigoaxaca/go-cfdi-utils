package xmlsafe

import (
	"encoding/xml"
	"errors"
	"testing"
)

func TestCheckToken(t *testing.T) {
	for name, tc := range map[string]struct {
		tok  xml.Token
		want error
	}{
		"doctype":            {xml.Directive("DOCTYPE a"), ErrDTD},
		"entity":             {xml.Directive(`ENTITY x "y"`), ErrDTD},
		"declaración":        {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="UTF-8"`)}, nil},
		"declaración rara":   {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" x="y"`)}, ErrDeclaration},
		"utf7 con espacios":  {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding = "UTF-7"`)}, ErrDeclaration},
		"utf7 espacio antes": {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding ="UTF-7"`)}, ErrDeclaration},
		"utf7 espacio luego": {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding= "UTF-7"`)}, ErrDeclaration},
		"utf8 con espacios":  {xml.ProcInst{Target: "xml", Inst: []byte(`version = "1.0" encoding = "utf-8"`)}, nil},
		"latin1 sin espacio": {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding="ISO-8859-1"`)}, nil},
		"latin1 con espacio": {xml.ProcInst{Target: "xml", Inst: []byte(`version="1.0" encoding ="ISO-8859-1"`)}, ErrDeclaration},
		"instrucción":        {xml.ProcInst{Target: "xml-stylesheet", Inst: []byte(`encoding = "UTF-7"`)}, nil},
		"elemento":           {xml.StartElement{Name: xml.Name{Local: "a"}}, nil},
		"texto":              {xml.CharData("<!DOCTYPE a>"), nil},
		"comentario":         {xml.Comment("<!DOCTYPE a>"), nil},
	} {
		if err := CheckToken(tc.tok); !errors.Is(err, tc.want) {
			t.Errorf("%s: se esperaba %v, se obtuvo %v", name, tc.want, err)
		}
	}
}
