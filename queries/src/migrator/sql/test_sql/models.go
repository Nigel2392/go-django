package testsql

import (
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	ExtendedDefinitions        = false
	ExtendedDefinitionsUser    = false
	ExtendedDefinitionsTodo    = false
	ExtendedDefinitionsProfile = false
)

type User struct {
	ID        int64  `attrs:"primary"`
	Name      string `attrs:"max_length=255"`
	Email     string `attrs:"max_length=255"`
	Age       int32  `attrs:"min_value=0;max_value=120"`
	IsActive  bool   `attrs:"-"`
	FirstName string `attrs:"-"`
	LastName  string `attrs:"-"`
}

func (m *User) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	var fields = fieldDefs.Fields()
	if ExtendedDefinitions {
		fields = append(fields, attrs.NewField(m, "FirstName", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "LastName", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitionsUser {
		fields = append(fields, attrs.NewField(m, "IsActive", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitions || ExtendedDefinitionsUser {
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type Profile struct {
	ID        int64  `attrs:"primary"`
	User      *User  `attrs:"o2o=test_sql.User;column=user_id"`
	Image     string `attrs:"-"`
	Biography string `attrs:"-"`
	Website   string `attrs:"-"`
}

func (m *Profile) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	var fields = fieldDefs.Fields()
	if ExtendedDefinitions {
		fields = append(fields, attrs.NewField(m, "Biography", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "Website", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitionsProfile {
		fields = append(fields, attrs.NewField(m, "Image", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitions || ExtendedDefinitionsProfile {
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type Todo struct {
	ID          int64     `attrs:"primary"`
	Title       string    `attrs:"max_length=255"`
	Completed   bool      `attrs:"default=false"`
	User        *User     `attrs:"fk=test_sql.User;column=user_id"`
	Description string    `attrs:"-"`
	CreatedAt   time.Time `attrs:"-"`
	UpdatedAt   time.Time `attrs:"-"`
}

func (m *Todo) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	var fields = fieldDefs.Fields()
	if ExtendedDefinitions {
		fields = append(fields, attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "UpdatedAt", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitionsTodo {
		fields = append(fields, attrs.NewField(m, "Description", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitions || ExtendedDefinitionsTodo {
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type BlogPost struct {
	ID        int64     `attrs:"primary"`
	Title     string    `attrs:"max_length=255"`
	Body      string    `attrs:"max_length=255"`
	Author    *User     `attrs:"fk=test_sql.User;column=author_id"`
	Published bool      `attrs:"-"`
	CreatedAt time.Time `attrs:"-"`
	UpdatedAt time.Time `attrs:"-"`
}

func (m *BlogPost) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	if ExtendedDefinitions {
		var fields = fieldDefs.Fields()
		fields = append(fields, attrs.NewField(m, "Published", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "UpdatedAt", &attrs.FieldConfig{}))
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type BlogComment struct {
	ID        int64     `attrs:"primary"`
	Body      string    `attrs:"max_length=255"`
	Author    *User     `attrs:"fk=test_sql.User;column=author_id"`
	Post      *BlogPost `attrs:"fk=test_sql.BlogPost;column=post_id"`
	CreatedAt time.Time `attrs:"-"`
	UpdatedAt time.Time `attrs:"-"`
}

func (m *BlogComment) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	if ExtendedDefinitions {
		var fields = fieldDefs.Fields()
		fields = append(fields, attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "UpdatedAt", &attrs.FieldConfig{}))
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}
