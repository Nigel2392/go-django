package codegen

import (
	"strings"

	"github.com/Nigel2392/go-django/cmd/go-django-definitions/internal/codegen/plugin"
)

func DataType(n *plugin.Identifier) string {
	if n.Schema != "" {
		return n.Schema + "." + n.Name
	} else {
		return n.Name
	}
}

func goInnerType(c *CodeGenerator, col *plugin.Column) string {
	//	columnType := DataType(col.Type)
	//	notNull := col.NotNull || col.IsArray
	//	// package overrides have a higher precedence
	//	for _, override := range c.opts.Overrides {
	//		oride := override.ShimOverride
	//		if oride.GoType.TypeName == "" {
	//			continue
	//		}
	//		if oride.DbType != "" && oride.DbType == columnType && oride.Nullable != notNull && oride.Unsigned == col.Unsigned {
	//			return oride.GoType.TypeName
	//		}
	//	}

	// TODO: Extend the engine interface to handle types
	switch c.opts.req.Settings.Engine {
	case "mysql":
		return mysqlType(c, col)
	case "postgresql":
		panic("PostgreSQL type not implemented")
	case "sqlite":
		return sqliteType(c, col)
	default:
		return "interface{}"
	}
}

func mysqlType(c *CodeGenerator, col *plugin.Column) string {
	columnType := DataType(col.Type)
	notNull := col.NotNull || col.IsArray
	unsigned := col.Unsigned
	switch columnType {

	case "varchar", "text", "char", "tinytext", "mediumtext", "longtext":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "tinyint":
		if col.Length == 1 {
			if notNull {
				return "bool"
			}
			return "sql.NullBool"
		} else {
			if notNull {
				if unsigned {
					return "uint8"
				}
				return "int8"
			}
			// The database/sql package does not have a sql.NullInt8 type, so we
			// use the smallest type they have which is NullInt16
			return "sql.NullInt16"
		}

	case "year":
		if notNull {
			return "int16"
		}
		return "sql.NullInt16"

	case "smallint":
		if notNull {
			if unsigned {
				return "uint16"
			}
			return "int16"
		}
		return "sql.NullInt16"

	case "int", "integer", "mediumint":
		if notNull {
			if unsigned {
				return "uint32"
			}
			return "int32"
		}
		return "sql.NullInt32"

	case "bigint":
		if notNull {
			if unsigned {
				return "uint64"
			}
			return "int64"
		}
		return "sql.NullInt64"

	case "blob", "binary", "varbinary", "tinyblob", "mediumblob", "longblob":
		if notNull {
			return "[]byte"
		}
		return "sql.NullString"

	case "double", "double precision", "real", "float":
		if notNull {
			return "float64"
		}
		return "sql.NullFloat64"

	case "decimal", "dec", "fixed":
		if notNull {
			return "string"
		}
		return "sql.NullString"

	case "enum":
		// TODO: Proper Enum support
		return "string"

	case "date", "timestamp", "datetime", "time":
		if notNull {
			return "time.Time"
		}
		return "sql.NullTime"

	case "boolean", "bool":
		if notNull {
			return "bool"
		}
		return "sql.NullBool"

	case "json":
		return "json.RawMessage"

	case "any":
		return "interface{}"

	default:
		for _, schema := range c.opts.req.Catalog.Schemas {
			for _, enum := range schema.Enums {
				if enum.Name == columnType {
					if notNull {
						if schema.Name == c.opts.req.Catalog.DefaultSchema {
							return c.opts.GoName(enum.Name)
						}
						return c.opts.GoName(schema.Name + "_" + enum.Name)
					} else {
						if schema.Name == c.opts.req.Catalog.DefaultSchema {
							return "Null" + c.opts.GoName(enum.Name)
						}
						return "Null" + c.opts.GoName(schema.Name+"_"+enum.Name)
					}
				}
			}
		}
		return "interface{}"
	}
}

func sqliteType(c *CodeGenerator, col *plugin.Column) string {
	dt := strings.ToLower(DataType(col.Type))
	notNull := col.NotNull || col.IsArray
	emitPointersForNull := c.opts.EmitPointersForNull

	switch dt {

	case "int", "integer", "tinyint", "smallint", "mediumint", "bigint", "unsignedbigint", "int2", "int8":
		if notNull {
			return "int64"
		}
		if emitPointersForNull {
			return "*int64"
		}
		return "sql.NullInt64"

	case "blob":
		return "[]byte"

	case "real", "double", "doubleprecision", "float":
		if notNull {
			return "float64"
		}
		if emitPointersForNull {
			return "*float64"
		}
		return "sql.NullFloat64"

	case "boolean", "bool":
		if notNull {
			return "bool"
		}
		if emitPointersForNull {
			return "*bool"
		}
		return "sql.NullBool"

	case "date", "datetime", "timestamp":
		if notNull {
			return "time.Time"
		}
		if emitPointersForNull {
			return "*time.Time"
		}
		return "sql.NullTime"

	case "any":
		return "interface{}"

	}

	switch {

	case strings.HasPrefix(dt, "character"),
		strings.HasPrefix(dt, "varchar"),
		strings.HasPrefix(dt, "varyingcharacter"),
		strings.HasPrefix(dt, "nchar"),
		strings.HasPrefix(dt, "nativecharacter"),
		strings.HasPrefix(dt, "nvarchar"),
		dt == "text",
		dt == "clob":
		if notNull {
			return "string"
		}
		if emitPointersForNull {
			return "*string"
		}
		return "sql.NullString"

	case strings.HasPrefix(dt, "decimal"), dt == "numeric":
		if notNull {
			return "float64"
		}
		if emitPointersForNull {
			return "*float64"
		}
		return "sql.NullFloat64"

	default:
		return "interface{}"

	}
}
