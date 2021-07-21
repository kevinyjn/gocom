package builtinmodels

import (
	"encoding/base64"
	"strconv"

	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/orm/rdbms/behaviors"
	"golang.org/x/crypto/bcrypt"
)

// UserModel interface
type UserModel interface {
	GetUserID() string
	SetUserID(string) error
	GetName() string
	SetName(string)
	GetPasswordHash() []byte
	SetPasswordHash([]byte)
	Save() (bool, error)
	NewRecord(name string, passwordHash []byte) UserModel
	FindByName(name string) (UserModel, error)
	VerifyPassword(passwordOrHash string) bool
}

// User 内置用户模型
type User struct {
	ID                          int64  `xorm:"'id' BigInt pk autoincr" json:"id" form:"id"`
	Name                        string `xorm:"'name' VARCHAR(64) notnull index" json:"name" form:"name"`
	Telephone                   string `xorm:"'telephone' VARCHAR(64) null index" json:"telephone" form:"telephone"`
	Email                       string `xorm:"'email' VARCHAR(64) null index" json:"email" form:"email"`
	Avatar                      string `xorm:"'avatar' VARCHAR(64) null index" json:"avatar" form:"avatar"`
	HashedPassword              string `xorm:"'passwd_hash' VARCHAR(64)" json:"-" form:"-"`
	behaviors.ModifyingBehavior `xorm:"extends"`
	rdbms.Datasource            `xorm:"-" datasource:"default"`
	Roles                       []NameInfo `xorm:"-" json:"roles"`
}

// IsValid 可以做一些非常简单的“低级”数据验证
func (m *User) IsValid() bool {
	return m.ID > 0
}

// TableName table name
func (m *User) TableName() string {
	return "sys_user"
}

// Fetch retrieve one record by self condition
func (m *User) Fetch() (bool, error) {
	return m.Datasource.Fetch(m)
}

// Save record to database
func (m *User) Save() (bool, error) {
	return m.Datasource.Save(m)
}

// Exists by record
func (m *User) Exists() (bool, error) {
	return m.Datasource.Exists(m)
}

// Count record
func (m *User) Count() (int64, error) {
	return m.Datasource.Count(m)
}

// Delete record
func (m *User) Delete() (int64, error) {
	return m.Datasource.Delete(m)
}

// GetID primary id
func (m *User) GetID() int64 {
	return m.ID
}

// GetName user name
func (m *User) GetName() string {
	return m.Name
}

// SetName user name
func (m *User) SetName(name string) {
	m.Name = name
}

// GetUserID primary id
func (m *User) GetUserID() string {
	return strconv.FormatInt(m.ID, 10)
}

// SetUserID primary id
func (m *User) SetUserID(id string) error {
	i64, err := strconv.ParseInt(id, 10, 64)
	if nil != err {
		return err
	}
	m.ID = i64
	return nil
}

// GetPasswordHash user password
func (m *User) GetPasswordHash() []byte {
	hashed, _ := base64.StdEncoding.DecodeString(m.HashedPassword)
	return hashed
}

// SetPasswordHash user password
func (m *User) SetPasswordHash(passwdHash []byte) {
	m.HashedPassword = base64.StdEncoding.EncodeToString(passwdHash)
}

// InsertMany records
func (m *User) InsertMany(records []interface{}) (int64, error) {
	return m.Datasource.Insert(records...)
}

// NewRecord of user model
func (m *User) NewRecord(name string, passwordHash []byte) UserModel {
	u := User{
		Name:           name,
		HashedPassword: base64.StdEncoding.EncodeToString(passwordHash),
	}
	return &u
}

// FindByName of user model
func (m *User) FindByName(name string) (UserModel, error) {
	user := &User{Name: name}
	_, err := m.Datasource.Fetch(user)
	return user, err
}

// VerifyPassword to user password
func (m *User) VerifyPassword(passwordOrHash string) bool {
	err := bcrypt.CompareHashAndPassword(m.GetPasswordHash(), []byte(passwordOrHash))
	if nil == err {
		return true
	}
	// if passwordOrHash == m.HashedPassword {
	// 	return true
	// }
	// passwordHash, err := GenerateHashedPassword(passwordOrHash)
	// if nil == err && passwordHash == m.HashedPassword {
	// 	return true
	// }
	return false
}

// GeneratePassword 将根据输入密码生成哈希密码
// 用户的输入
func GeneratePassword(userPassword string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
}

// GenerateHashedPassword 将根据输入密码生成哈希密码
// 用户的输入
func GenerateHashedPassword(userPassword string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	if nil != err {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(passwordHash), nil
}
