package builtinmodels

// RecordModel interface
type RecordModel interface {
	Save() (bool, error)
}
