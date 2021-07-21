package builtincontrollers

import (
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/microsvc/parameters"
)

// RoleParam form data
type RoleParam struct {
	ID       int64  `json:"id" form:"id" validate:"optional" label:"ID"`
	Name     string `json:"name" form:"name" validate:"required" label:"名称"`
	Code     string `json:"code" form:"code" validate:"required" label:"编码"`
	SystemID string `json:"system_id" form:"system_id" validate:"optional" label:"所属系统"`
	Remark   string `json:"remark" form:"remark" validate:"optional" label:"备注"`
}

// ModuleParam form data
type ModuleParam struct {
	ID        int64  `json:"id" form:"id" validate:"optional" label:"ID"`
	Name      string `json:"name" form:"name" validate:"required" label:"名称"`
	Code      string `json:"code" form:"code" validate:"required" label:"编码"`
	Path      string `json:"path" form:"path" validate:"required" label:"模块路径"`
	SystemID  string `json:"system_id" form:"system_id" validate:"optional" label:"所属系统"`
	ParentID  int64  `json:"parent_id" form:"parent_id" validate:"optional" label:"父级ID"`
	Icon      string `json:"icon" form:"icon" validate:"optional" label:"图标"`
	ViewOrder int    `json:"view_order" form:"view_order" validate:"optional" label:"显示顺序"`
	Flag      int    `json:"flag" form:"flag" validate:"optional" label:"标志位"`
	Remark    string `json:"remark" form:"remark" validate:"optional" label:"备注"`
}

// UserParam form data
type UserParam struct {
	ID        int64  `json:"id" form:"id" validate:"optional" label:"ID"`
	Name      string `json:"name" form:"name" validate:"required" label:"名称"`
	Telephone string `json:"telephone" form:"telephone" validate:"optional" label:"电话号码"`
	Email     string `json:"email" form:"email" validate:"optional" label:"电子邮箱"`
	Avatar    string `json:"avatar" form:"avatar" validate:"optional" label:"头像"`
	Password  string `json:"password" form:"password" validate:"optional" label:"密码"`
}

// RoleResponse response parameter
type RoleResponse struct {
	Name string `json:"name" validate:"required" label:"名称"`
}

// RoleListQueryParam role list query
type RoleListQueryParam struct {
	parameters.ListQueryParam
	RoleQueryParam
}

// RoleQueryParam query filter fields of role
type RoleQueryParam struct {
	ID       int64  `json:"id" form:"id" validate:"optional" label:"ID"`
	Name     string `json:"name" form:"name" validate:"optional" label:"名称"`
	Code     string `json:"code" form:"code" validate:"optional" label:"编码"`
	SystemID string `json:"system_id" form:"system_id" validate:"optional" label:"所属系统"`
}

// RoleListQueryResponse role list query response data
type RoleListQueryResponse struct {
	parameters.ListQueryResponse
	Items []*builtinmodels.Role `json:"items" label:"数据"`
}

// ModuleResponse response parameter
type ModuleResponse struct {
	Name string `json:"name" validate:"required" label:"名称"`
}

// ModuleListQueryParam role list query
type ModuleListQueryParam struct {
	parameters.ListQueryParam
	ModuleQueryParam
}

// ModuleQueryParam query filter fields of role
type ModuleQueryParam struct {
	ID       int64  `json:"id" form:"id" validate:"optional" label:"ID"`
	Name     string `json:"name" form:"name" validate:"optional" label:"名称"`
	Code     string `json:"code" form:"code" validate:"optional" label:"编码"`
	SystemID string `json:"system_id" form:"system_id" validate:"optional" label:"所属系统"`
	ParentID int64  `json:"parent_id" form:"parent_id" validate:"optional" label:"父级节点"`
}

// ModuleListQueryResponse module list query response data
type ModuleListQueryResponse struct {
	parameters.ListQueryResponse
	Items []*builtinmodels.Module `json:"items" label:"数据"`
}

// ModuleTreeDataResponse module tree data response data
type ModuleTreeDataResponse struct {
	Items []parameters.TreeData `json:"items" label:"数据"`
}

// UserResponse response parameter
type UserResponse struct {
	Name string `json:"name" validate:"required" label:"名称"`
}

// UserListQueryParam role list query
type UserListQueryParam struct {
	parameters.ListQueryParam
	UserQueryParam
}

// UserQueryParam query filter fields of role
type UserQueryParam struct {
	ID       int64  `json:"id" form:"id" validate:"optional" label:"ID"`
	Name     string `json:"name" form:"name" validate:"optional" label:"名称"`
	Code     string `json:"code" form:"code" validate:"optional" label:"编码"`
	SystemID string `json:"system_id" form:"system_id" validate:"optional" label:"所属系统"`
}

// UserListQueryResponse role list query response data
type UserListQueryResponse struct {
	parameters.ListQueryResponse
	Items []*builtinmodels.User `json:"items" label:"数据"`
}

// LoginParam form data
type LoginParam struct {
	Name     string `json:"name" form:"name" validate:"required" label:"名称"`
	Password string `json:"password" form:"password" validate:"required" label:"密码"`
}

// UserRoleRelationParam form data
type UserRoleRelationParam struct {
	UserID   int64   `json:"user_id" form:"user_id" validate:"required" label:"用户ID"`
	RoleIDs  []int64 `json:"role_ids" form:"role_ids" validate:"optional" label:"角色ID列表"`
	SystemID string  `json:"system_id" form:"system_id" validate:"optional" label:"系统编号"`
}

// RoleModuleRelationParam form data
type RoleModuleRelationParam struct {
	RoleID    int64   `json:"role_id" form:"role_id" validate:"required" label:"角色ID"`
	ModuleIDs []int64 `json:"module_ids" form:"module_ids" validate:"optional" label:"模块ID列表"`
	SystemID  string  `json:"system_id" form:"system_id" validate:"optional" label:"系统编号"`
}

// RelationsResponse response
type RelationsResponse struct {
	Adds    []int64 `json:"adds" label:"新增的ID"`
	Deletes []int64 `json:"deletes" label:"移除的ID"`
}
