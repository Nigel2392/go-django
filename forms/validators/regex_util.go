package validators

import "strings"

// This type of regex parsing is also used in github.com/Nigel2392/routevars.Match()
// It is used to parse a string and replace variables with regex
// For example: "user/<<id:int>>" will be parsed to "user/(?P<id>[0-9]+)"

const (
	VAR_PREFIX = "<<"
	VAR_SUFFIX = ">>"
	NameInt    = "int"
	NameString = "string"
	NameSlug   = "slug"
	NameUUID   = "uuid"
	NameAny    = "any"
	NameHex    = "hex"
	NameEmail  = "email"
	NamePhone  = "phone"
	NameBool   = "bool"
	NameFloat  = "float"
)

const (
	// Match any character
	REGEX_ANY = ".+"
	// Match any number
	REGEX_NUM = "[0-9]+"
	// Match any string
	REGEX_STR = "[a-zA-Z]+"
	// Match any hex number
	REGEX_HEX = "[0-9a-fA-F]+"
	// Match any UUID
	REGEX_UUID = "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
	// Match any alphanumeric string
	REGEX_ALPHANUMERIC = "[0-9a-zA-Z_-]+"
	// Match any email address
	REGEX_EMAIL = `(?:[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+)*|"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\[(?:(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9]))\.){3}(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9])|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\])`
	// Match any phone number
	REGEX_PHONE = `^[\+]?[(]?[0-9]{3}[)]?[-\s\.]?[0-9]{3}[-\s\.]?[0-9]{4,6}$`
	// Match any boolean
	REGEX_BOOL = "true|false|True|False|TRUE|FALSE|1|0"
	// Match any float
	REGEX_FLOAT = `\d+\.\d+|\d+,\d+`
)

var tokens = map[string]string{
	NameInt:    REGEX_NUM,
	NameString: REGEX_STR,
	NameSlug:   REGEX_ALPHANUMERIC,
	NameUUID:   REGEX_UUID,
	NameAny:    REGEX_ANY,
	NameHex:    REGEX_HEX,
	NameEmail:  REGEX_EMAIL,
	NamePhone:  REGEX_PHONE,
	NameBool:   REGEX_BOOL,
	NameFloat:  REGEX_FLOAT,
}

// Convert a string to a regex string with a capture group.
func toRegex(str string) string {
	if !strings.HasPrefix(str, VAR_PREFIX) || !strings.HasSuffix(str, VAR_SUFFIX) {
		return str
	}
	str = strings.TrimPrefix(str, VAR_PREFIX)
	str = strings.TrimSuffix(str, VAR_SUFFIX)
	return typToRegx(str)
}

// Convert a type (string) to a regex for use in capture groups.
func typToRegx(typ string) string {
	// regex for raw is: raw(REGEX)
	var hasRaw string = strings.ToLower(typ)
	if strings.HasPrefix(hasRaw, "raw(") && strings.HasSuffix(hasRaw, ")") {
		return hasRaw[4 : len(hasRaw)-1]
	}
	if val, ok := tokens[typ]; ok {
		return val
	} else {
		return typ
	}
}
