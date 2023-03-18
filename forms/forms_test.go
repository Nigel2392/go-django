package forms_test

import (
	"github.com/Nigel2392/go-django/forms"
	"testing"
)

type Structie struct {
	Name  string   `form:"label:Name:Name; placeholder:Name; required:true;"`
	Names []string `form:"label:Names:Names; placeholder:Names; required:true;"`
	Age   int      `form:"label:Age:Age; placeholder:Age; required:true;"`
	Male  bool     `form:"label:Male:Male; required:true;"`
	Cash  float64  `form:"label:Cash:Cash; placeholder:Cash; required:true;"`
}

func TestFormFromStruct(t *testing.T) {
	var s = Structie{
		Name:  "John",                  //text
		Names: []string{"John", "Doe"}, //select
		Age:   42,                      //number
		Male:  true,                    //checkbox
		Cash:  42.42,                   //number
	}

	fields, err := forms.GenerateFieldsFromStruct(s)
	if err != nil {
		panic(err)
	}
	if len(fields) != 5 {
		panic("Expected 5 fields")
	}
	if fields[0].String() != "<label for=\"Name\">Name</label>\r\n<input type=\"text\" id=\"Name\" name=\"Name\" placeholder=\"Name\" value=\"John\" required>\r\n" {
		t.Errorf("Expected \n%s\ngot \n%s", "<label for=\"Name\">Name</label>\r\n<input type=\"text\" id=\"Name\" name=\"Name\" placeholder=\"Name\" value=\"John\" required>\r\n", fields[0].String())
	}
	if fields[1].String() != "<label for=\"Names\">Names</label>\r\n<select type=\"select\" id=\"Names\" name=\"Names\" placeholder=\"Names\" required><option value=\"John\">John</option><option value=\"Doe\">Doe</option></select>" {
		t.Errorf("Expected \n%s\ngot \n%s", "<label for=\"Names\">Names</label>\r\n<select type=\"select\" id=\"Names\" name=\"Names\" placeholder=\"Names\" required><option value=\"John\">John</option><option value=\"Doe\">Doe</option></select>", fields[1].String())
	}
	if fields[2].String() != "<label for=\"Age\">Age</label>\r\n<input type=\"number\" id=\"Age\" name=\"Age\" placeholder=\"Age\" value=\"42\" required>\r\n" {
		t.Errorf("Expected \n%s\ngot \n%s", "<label for=\"Age\">Age</label>\r\n<input type=\"number\" id=\"Age\" name=\"Age\" placeholder=\"Age\" value=\"42\" required>\r\n", fields[2].String())
	}
	if fields[3].String() != "<label for=\"Male\">Male</label>\r\n<input type=\"checkbox\" id=\"Male\" name=\"Male\" placeholder=\"Male\" value=\"true\" required checked>\r\n" {
		t.Errorf("Expected \n%s\ngot \n%s", "<label for=\"Male\">Male</label>\r\n<input type=\"checkbox\" id=\"Male\" name=\"Male\" placeholder=\"Male\" value=\"true\" required checked>\r\n", fields[3].String())
	}
	if fields[4].String() != "<label for=\"Cash\">Cash</label>\r\n<input type=\"number\" id=\"Cash\" name=\"Cash\" placeholder=\"Cash\" value=\"42.420000\" required>\r\n" {
		t.Errorf("Expected \n%s\ngot \n%s", "<label for=\"Cash\">Cash</label>\r\n<input type=\"number\" id=\"Cash\" name=\"Cash\" placeholder=\"Cash\" value=\"42.420000\" required>\r\n", fields[4].String())
	}
	for _, field := range fields {
		if field.String() == "" {
			t.Errorf("Expected field to be not empty")
		}
		t.Log(field.String())
	}
}
