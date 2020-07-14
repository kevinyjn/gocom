package scaffolds

import (
	"fmt"
	"path"
	"strings"

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
	Name         string `json:"name"`
	Column       string `json:"column" validate:"required"`
	Type         string `json:"type" validate:"required"`
	Length       int    `json:"length" validate:"optional"`
	IsPrimaryKey bool   `json:"isPrimaryKey" validate:"optional"`
	IsIndex      bool   `json:"isIndex"`
	IsRequired   bool   `json:"isRequired"`
}

// Encode model context
func (m *ModelScaffolds) Encode() string {
	pkgName := path.Base(m.Path)
	if strings.HasSuffix(pkgName, ".go") {
		pkgName = pkgName[0 : len(pkgName)-3]
	}
	if strings.Contains(pkgName, ".") {
		pkgNames := strings.Split(pkgName, ".")
		pkgName = pkgNames[len(pkgNames)-1]
	}
	structureName := utils.CamelString(m.Name)
	if "" == structureName {
		structureName = utils.CamelString(m.TableName)
	}
	tableName := m.TableName
	if "" == tableName {
		tableName = utils.SnakeString(structureName)
	}
	fields := make([]string, len(m.Fields))
	fieldSpaces := 27
	if nil != m.Fields {
		for _, field := range m.Fields {
			if len(field.Name) > fieldSpaces {
				fieldSpaces = len(field.Name)
			}
		}
		for i, field := range m.Fields {
			fields[i] = field.Encode(fieldSpaces)
		}
	}
	lines := []string{
		fmt.Sprintf("package %s\n", pkgName),
		"import (",
		"\t\"github.com/kevinyjn/gocom/orm/rdbms/behaviors\"",
		"\t\"github.com/kevinyjn/gocom/orm/rdbms/dal\"",
		")\n",
		fmt.Sprintf("// %s model", structureName),
		fmt.Sprintf("type %s struct {", structureName),
		strings.Join(fields, "\n"),
		fmt.Sprintf("\tbehaviors.ModifyingBehavior %s`xorm:\"extends\"`", strings.Repeat(" ", fieldSpaces-27)),
		fmt.Sprintf("\tdal.Datasource %s`xorm:\"-\" datasource:\"default\"`", strings.Repeat(" ", fieldSpaces-14)),
		"}\n",
		"// TableName returns table name in database",
		fmt.Sprintf("func (m *%s) TableName() string {\n\treturn \"%s\"\n}\n", structureName, tableName),
		"// Fetch retrieve one record by self condition",
		fmt.Sprintf("func (m *%s) Fetch() error {\n\treturn m.Datasource.Fetch(m)\n}\n", structureName),
		"// Save record to database",
		fmt.Sprintf("func (m *%s) Save() error {\n\treturn m.Datasource.Save(m)\n}\n", structureName),
	}

	return strings.Join(lines, "\n")
}

// Encode as xorm field defination
func (f *ModelField) Encode(spaces int) string {
	fieldName := utils.CamelString(f.Name)
	if "" == fieldName {
		fieldName = utils.CamelString(f.Column)
	}
	if "" == fieldName {
		return ""
	}
	jsonName := strings.ToLower(fieldName[:1]) + fieldName[1:]
	columnName := f.Column
	if "" == columnName {
		columnName = utils.SnakeString(fieldName)
	}

	return fmt.Sprintf("\t%s %s`xorm:\"'%s' %s\" json:\"%s\"`",
		fieldName, strings.Repeat(" ", spaces-len(fieldName)), columnName, f.serializeFieldSpecs(), jsonName)
}

func (f *ModelField) serializeFieldSpecs() string {
	fieldType := f.Type
	if f.Length > 0 {
		if strings.Compare(strings.ToUpper(f.Type), "VARCHAR") == 0 {
			fieldType += fmt.Sprintf("(%d)", f.Length)
		}
	}
	fieldSpecs := []string{
		fieldType,
	}
	if f.IsRequired {
		fieldSpecs = append(fieldSpecs, "notnull")
	}
	if f.IsPrimaryKey {
		fieldSpecs = append(fieldSpecs, "pk")
	} else if f.IsIndex {
		fieldSpecs = append(fieldSpecs, "index")
	}

	return strings.Join(fieldSpecs, " ")
}
