package builtinmodels

import (
	"fmt"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
)

// Role 内置RBAC-角色模型。
type Role struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id" form:"id"`
	Name                        string `xorm:"'name' VARCHAR(64) notnull index" json:"name" form:"name"`
	Code                        string `xorm:"'code' VARCHAR(45) notnull index" json:"code" form:"code"`
	AppID                       string `xorm:"'app_id' VARCHAR(45) null index" json:"app_id" form:"app_id"`
	SystemID                    string `xorm:"'system_id' VARCHAR(45) null index" json:"system_id" form:"system_id"`
	Avatar                      string `xorm:"'avatar' VARCHAR(64) null" json:"avatar" form:"avatar"`
	Remark                      string `xorm:"'remark' VARCHAR(255) null" json:"remark" form:"remark"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// IsValid 可以做一些非常简单的“低级”数据验证
func (m *Role) IsValid() bool {
	return m.ID > 0
}

// TableName table name
func (m *Role) TableName() string {
	return "sys_role"
}

// Fetch retrieve one record by self condition
func (m *Role) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *Role) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *Role) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *Role) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *Role) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// GetID primary id
func (m *Role) GetID() int64 {
	return m.ID
}

// InsertMany records
func (m *Role) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}

// Module 内置RBAC-模块模型。
type Module struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id" form:"id"`
	Code                        string `xorm:"'code' VARCHAR(45) notnull index" json:"code" form:"code"`
	Name                        string `xorm:"'name' VARCHAR(64) notnull index" json:"name" form:"name"`
	Path                        string `xorm:"'path' VARCHAR(128) notnull index" json:"path" form:"path"`
	SystemID                    string `xorm:"'system_id' VARCHAR(45) null index" json:"system_id" form:"system_id"`
	ParentID                    int64  `xorm:"'parent_id' BigInt index" json:"parent_id" form:"parent_id"`
	Icon                        string `xorm:"'icon' VARCHAR(45) null" json:"icon" form:"icon"`
	ViewOrder                   int    `xorm:"'view_order' Int default(1)" json:"view_order" form:"view_order"`
	Flag                        int    `xorm:"'flag' Int index" json:"flag" form:"flag"`
	Remark                      string `xorm:"'remark' VARCHAR(255) null" json:"remark" form:"remark"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// IsValid 可以做一些非常简单的“低级”数据验证
func (m *Module) IsValid() bool {
	return m.ID > 0
}

// TableName table name
func (m *Module) TableName() string {
	return "sys_module"
}

// Fetch retrieve one record by self condition
func (m *Module) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *Module) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *Module) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *Module) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *Module) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// InsertMany records
func (m *Module) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}

// FindAuthorizedModules by user id
func FindAuthorizedModules(userID int64) ([]Module, error) {
	modules := []Module{}
	module := Module{}
	engine, err := rdbms.GetInstance().GetDbEngine(&module)
	if nil != err {
		logger.Error.Printf("Find authorized modules by user id %d while get db engine failed with error:%+v", userID, err)
		return modules, err
	}
	roleModule := RoleModuleRelation{}
	userRole := UserRoleRelation{}
	// "select * from module join role_module on role_module.module_id = module.id join user_role on user_role.role_id = role_module.role_id where user_role.user_id = :uid"
	err = engine.Table(&module).Join(
		"INNER", &roleModule, fmt.Sprintf("%s.module_id = %s.id", roleModule.TableName(), module.TableName()),
	).Join(
		"INNER", &userRole, fmt.Sprintf("%s.role_id = %s.role_id", roleModule.TableName(), userRole.TableName()),
	).In(fmt.Sprintf("%s.user_id", userRole.TableName()), userID).Find(&modules)
	if nil != err {
		logger.Error.Printf("Find authorized modules by user id %d while query sql failed with error:%+v", userID, err)
	}
	return modules, err
}

// RoleModuleRelation 内置RBAC-角色与被授权模块关联图模型。
type RoleModuleRelation struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id" form:"id"`
	ModuleID                    int64  `xorm:"'module_id' BigInt notnull index unique(uniq_role_module_idx)" json:"module_id" form:"module_id"`
	RoleID                      int64  `xorm:"'role_id' BigInt notnull index unique(uniq_role_module_idx)" json:"role_id" form:"role_id"`
	SystemID                    string `xorm:"'system_id' VARCHAR(45) null index" json:"system_id" form:"system_id"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// IsValid 可以做一些非常简单的“低级”数据验证
func (m *RoleModuleRelation) IsValid() bool {
	return m.ID > 0
}

// TableName table name
func (m *RoleModuleRelation) TableName() string {
	return "sys_role_module"
}

// Fetch retrieve one record by self condition
func (m *RoleModuleRelation) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *RoleModuleRelation) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *RoleModuleRelation) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *RoleModuleRelation) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *RoleModuleRelation) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// InsertMany records
func (m *RoleModuleRelation) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}

// UserRoleRelation 内置RBAC-角色与被授权模块关联图模型。
type UserRoleRelation struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id" form:"id"`
	RoleID                      int64  `xorm:"'role_id' BigInt notnull index unique(uniq_user_role_idx)" json:"role_id" form:"role_id"`
	UserID                      int64  `xorm:"'user_id' BigInt notnull index unique(uniq_user_role_idx)" json:"user_id" form:"user_id"`
	SystemID                    string `xorm:"'system_id' VARCHAR(45) null index" json:"system_id" form:"system_id"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
}

// IsValid 可以做一些非常简单的“低级”数据验证
func (m *UserRoleRelation) IsValid() bool {
	return m.ID > 0
}

// TableName table name
func (m *UserRoleRelation) TableName() string {
	return "sys_user_role"
}

// Fetch retrieve one record by self condition
func (m *UserRoleRelation) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *UserRoleRelation) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *UserRoleRelation) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *UserRoleRelation) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *UserRoleRelation) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// InsertMany records
func (m *UserRoleRelation) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}

// GetRBACModels all RBAC related models
func GetRBACModels() []interface{} {
	return []interface{}{
		&User{},
		&Role{},
		&Module{},
		&RoleModuleRelation{},
		&UserRoleRelation{},
	}
}
