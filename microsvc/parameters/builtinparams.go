package parameters

// IDParam form data
type IDParam struct {
	ID int64 `json:"id" form:"id" validate:"optional" label:"ID"`
}

// PaginationParam pagination parameter
type PaginationParam struct {
	Page     int `json:"page" form:"page" validate:"optional" label:"页码"`
	PageSize int `json:"pageSize" form:"pageSize" validate:"required" default:"20" label:"分页条数"`
}

// GetPageOffset offset for db offset query
func (p *PaginationParam) GetPageOffset() int {
	if p.Page > 0 {
		return p.Page - 1
	}
	return 0
}

// GetPageSize page size for db limit query
func (p *PaginationParam) GetPageSize() int {
	if p.PageSize > 1000 {
		return 1000
	} else if p.PageSize <= 0 {
		return 10
	}
	return p.PageSize
}

// SortingParam sorting parameter
type SortingParam struct {
	Sort  string `json:"sort" form:"sort" validate:"optional" label:"排序字段"`
	Order string `json:"sortOrder" form:"sortOrder" validate:"optional" label:"排序顺序"`
}

// ListQueryParam list query parameter
type ListQueryParam struct {
	PaginationParam
	SortingParam
	Filters map[string]string `json:"filter" form:"filter" validate:"optional" label:"筛选"`
}

// ListQueryResponse list query response data
type ListQueryResponse struct {
	Total int64       `json:"total" validate:"required" label:"总计"`
	Items interface{} `json:"items" label:"数据"`
}
