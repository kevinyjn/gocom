package params

// HelloParam parameter
type HelloParam struct {
	Name    string `json:"name" validate:"required" label:"名称"`
	Version string `json:"version" validate:"optional" label:"版本号"`
}

// HelloResponse response parameter
type HelloResponse struct {
	Name  string `json:"name" validate:"required" label:"名称"`
	Title string `json:"title" validate:"required" label:"标题"`
}
