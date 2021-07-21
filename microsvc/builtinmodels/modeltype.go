package builtinmodels

// RecordModel interface
type RecordModel interface {
	Save() (bool, error)
}

type NameInfo struct {
	ID   interface{} `json:"id" xorm:"'id'"`
	Name string      `json:"name" xorm:"'name' VARCHAR"`
}
