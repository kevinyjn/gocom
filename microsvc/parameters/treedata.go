package parameters

// TreeDataInterface interface
type TreeDataInterface interface {
	GetKey() interface{}
	GetTitle() string
	SetChildren([]TreeDataInterface)
}

// TreeData structure
type TreeData struct {
	Key      interface{} `json:"key"`
	Title    string      `json:"title"`
	Children []TreeData  `json:"children"`
	BindData interface{} `json:"bindData,omitempty"`
}

// GetKey key field
func (td *TreeData) GetKey() interface{} {
	return td.Key
}

// GetTitle title field
func (td *TreeData) GetTitle() string {
	return td.Title
}

func (td *TreeData) SetChildren(children []TreeDataInterface) {
	if nil != children {
		td.Children = make([]TreeData, len(children))
		i := 0
		for _, child := range children {
			cd, ok := interface{}(child).(*TreeData)
			if ok {
				td.Children[i] = *cd
				i++
			}
		}
		if i < len(td.Children) {
			td.Children = td.Children[:i]
		}
	}
}

// BuiltinTreeDataBuilder builder of tree data list
type BuiltinTreeDataBuilder struct {
	data map[interface{}][]TreeData
}

// Push element
func (tb *BuiltinTreeDataBuilder) Push(treeElement TreeData, parentKey interface{}) {
	if nil == tb.data {
		tb.data = map[interface{}][]TreeData{}
	}
	children, ok := tb.data[parentKey]
	if ok {
		tb.data[parentKey] = append(children, treeElement)
	} else {
		tb.data[parentKey] = []TreeData{treeElement}
	}
}

// GetTreeData list
func (tb *BuiltinTreeDataBuilder) GetTreeData(parentKey interface{}) []TreeData {
	var treeData []TreeData
	if nil == tb.data {
		return treeData
	}
	children, ok := tb.data[parentKey]
	if ok {
		treeData = []TreeData{}
		for _, child := range children {
			childTreeData := tb.GetTreeData(child.GetKey())
			if len(childTreeData) > 0 {
				child.Children = childTreeData
			}
			treeData = append(treeData, child)
		}
	}
	return treeData
}

// TreeDataBuilder builder of tree data list
type TreeDataBuilder struct {
	data map[interface{}][]TreeDataInterface
}

// Push element
func (tb *TreeDataBuilder) Push(treeElement TreeDataInterface, parentKey interface{}) {
	if nil == tb.data {
		tb.data = map[interface{}][]TreeDataInterface{}
	}
	children, ok := tb.data[parentKey]
	if ok {
		tb.data[parentKey] = append(children, treeElement)
	} else {
		tb.data[parentKey] = []TreeDataInterface{treeElement}
	}
}

// GetTreeData list
func (tb *TreeDataBuilder) GetTreeData(parentKey interface{}) []TreeDataInterface {
	var treeData []TreeDataInterface
	if nil == tb.data {
		return treeData
	}
	children, ok := tb.data[parentKey]
	if ok {
		treeData = []TreeDataInterface{}
		for _, child := range children {
			childTreeData := tb.GetTreeData(child.GetKey())
			if len(childTreeData) > 0 {
				child.SetChildren(childTreeData)
			}
			treeData = append(treeData, child)
		}
	}
	return treeData
}
