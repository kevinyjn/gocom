package models

// AllModelStructures get all model structures
func AllModelStructures() []interface{} {
	defs := []interface{}{
		&ExampleModel{},
	} // ends of models array. NOTE: do not remove or change this comment!
	return defs
}
