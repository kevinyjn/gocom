package scaffolds

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/utils"
)

// ModelScaffolds scaffolds
type ModelScaffolds struct {
	Path       string       `json:"path" validate:"required"`
	Name       string       `json:"name" validate:"required"`
	TableName  string       `json:"tableName" validate:"required"`
	Fields     []ModelField `json:"fields" validate:"required"`
	Datasource string       `json:"datasource" validate:"required"`
}

// ModelField field structure
type ModelField struct {
	Name           string `json:"name"`
	Column         string `json:"column" validate:"required"`
	ColumnType     string `json:"columnType" validate:"required"`
	Type           string `json:"type" validate:"required"`
	Length         int    `json:"length" validate:"optional"`
	IsPrimaryKey   bool   `json:"isPrimaryKey" validate:"optional"`
	IsIndex        bool   `json:"isIndex"`
	AutoIncreament bool   `json:"autoIncreament"`
	IsRequired     bool   `json:"isRequired"`
}

// FieldTypeMappings map[golangType]dbType
var FieldTypeMappings = map[string]string{
	"int":        "Int",
	"int8":       "Int",
	"int16":      "Int",
	"int32":      "Int",
	"uint":       "Int",
	"uint8":      "Int",
	"uint16":     "Int",
	"uint32":     "Int",
	"int64":      "BigInt",
	"uint64":     "BigInt",
	"float32":    "Float",
	"float64":    "Double",
	"complex64":  "Varchar(64)",
	"complex128": "Varchar(64)",
	"[]uint8":    "Blob",
	"[]byte":     "Blob",
	"array":      "Text",
	"slice":      "Text",
	"map":        "Text",
	"bool":       "Bool",
	"string":     "Varchar(255)",
	"time.Time":  "DateTime",
}

// Encode model context
func (m *ModelScaffolds) Encode() string {
	pkgName := getGoSourcePackageName(m.Path, "models")
	structureName := m.GetStructName()
	tableName := m.TableName
	if "" == tableName {
		tableName = utils.SnakeCaseString(structureName)
	}
	fields := make([]string, len(m.Fields))
	fieldSpaces := 27
	typeSpaces := 3
	exImports := []string{}
	if nil != m.Fields {
		for _, field := range m.Fields {
			if len(field.Name) > fieldSpaces {
				fieldSpaces = len(field.Name)
			}
			fieldType := field.getFieldType()
			switch fieldType {
			case "time.Time":
				exImports = append(exImports, fmt.Sprintf("\t\"%s\"\n", "time"))
				break
			}
			if len(fieldType) > typeSpaces {
				typeSpaces = len(fieldType)
			}
		}
		for i, field := range m.Fields {
			fields[i] = field.Encode(fieldSpaces, typeSpaces)
		}
	}
	if len(exImports) > 0 {
		sort.Sort(sort.StringSlice(exImports))
		exImports = append(exImports, "\n")
	}
	lines := []string{
		fmt.Sprintf("package %s\n", pkgName),
		"import (",
		fmt.Sprintf("%s\t\"github.com/kevinyjn/gocom/orm/rdbms\"", strings.Join(exImports, "")),
		"\t\"github.com/kevinyjn/gocom/orm/rdbms/behaviors\"",
		")\n",
		fmt.Sprintf("// %s model", structureName),
		fmt.Sprintf("type %s struct {", structureName),
		strings.Join(fields, "\n"),
		fmt.Sprintf("\tbehaviors.ModifyingBehavior %s`xorm:\"extends\"`", strings.Repeat(" ", fieldSpaces-27)),
		fmt.Sprintf("\trdbms.Datasource %s`xorm:\"-\" datasource:\"default\"`", strings.Repeat(" ", fieldSpaces-16)),
		"}\n",
		"// TableName returns table name in database",
		fmt.Sprintf("func (m *%s) TableName() string {\n\treturn \"%s\"\n}\n", structureName, tableName),
		"// Fetch retrieve one record by self condition",
		fmt.Sprintf("func (m *%s) Fetch() (bool, error) {\n\treturn m.Datasource.Fetch(m)\n}\n", structureName),
		"// Save record to database",
		fmt.Sprintf("func (m *%s) Save() (bool, error) {\n\treturn m.Datasource.Save(m)\n}\n", structureName),
		"// Exists by record",
		fmt.Sprintf("func (m *%s) Exists() (bool, error) {\n\treturn m.Datasource.Exists(m)\n}\n", structureName),
		"// Count record",
		fmt.Sprintf("func (m *%s) Count() (int64, error) {\n\treturn m.Datasource.Count(m)\n}\n", structureName),
		"// Delete record",
		fmt.Sprintf("func (m *%s) Delete() (int64, error) {\n\treturn m.Datasource.Delete(m)\n}\n", structureName),
	}

	return strings.Join(lines, "\n")
}

// GetStructName converts name to camel string
func (m *ModelScaffolds) GetStructName() string {
	structureName := utils.PascalCaseString(m.Name)
	if "" == structureName {
		structureName = utils.PascalCaseString(m.TableName)
	}
	return structureName
}

// Encode as xorm field defination
func (f *ModelField) Encode(fieldPaces int, typeSpaces int) string {
	fieldName := utils.PascalCaseString(f.Name)
	if "" == fieldName {
		fieldName = utils.PascalCaseString(f.Column)
	}
	if "" == fieldName {
		return ""
	}
	jsonName := utils.CamelCaseString(fieldName)
	columnName := f.Column
	if "" == columnName {
		columnName = utils.SnakeCaseString(fieldName)
	}
	fieldType := f.getFieldType()
	columnType := f.getColumnType()

	return fmt.Sprintf("\t%s %s %s %s`xorm:\"'%s' %s%s\" json:\"%s\"`",
		fieldName, strings.Repeat(" ", fieldPaces-len(fieldName)),
		fieldType, strings.Repeat(" ", typeSpaces-len(fieldType)),
		columnName, columnType,
		f.serializeFieldSpecs(), jsonName)
}

func (f *ModelField) getFieldType() string {
	fieldType := f.Type
	if strings.ContainsAny(fieldType, "./[]") == false {
		fieldType = strings.ToLower(f.Type)
	}
	if FieldTypeMappings[fieldType] == "" {
		logger.Error.Printf("ModelField:%v get field type while the fieldType:%s not recognized, treat as string", f, f.Type)
		fieldType = "string"
	}
	return fieldType
}

func (f *ModelField) getColumnType() string {
	columnType := FieldTypeMappings[f.getFieldType()]
	if "" == columnType {
		logger.Error.Printf("ModelField:%v get field type while the fieldType:%s not recognized, treat as string", f, f.Type)
		columnType = FieldTypeMappings["string"]
	}
	if f.Length > 0 {
		r := regexp.MustCompile(`\((\d+)\)`)
		rs := r.FindAllStringSubmatch(columnType, 1)
		if len(rs) > 0 {
			columnType = r.ReplaceAllString(columnType, fmt.Sprintf("(%d)", f.Length))
		}
	}
	return columnType
}

func (f *ModelField) serializeFieldSpecs() string {
	fieldSpecs := []string{}
	if f.IsRequired {
		fieldSpecs = append(fieldSpecs, "notnull")
	}
	if f.IsPrimaryKey {
		fieldSpecs = append(fieldSpecs, "pk")
	} else if f.IsIndex {
		fieldSpecs = append(fieldSpecs, "index")
	}
	if f.AutoIncreament {
		fieldSpecs = append(fieldSpecs, "autoincr")
	}

	if len(fieldSpecs) > 0 {
		return " " + strings.Join(fieldSpecs, " ")
	}
	return ""
}
