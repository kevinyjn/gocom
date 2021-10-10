package unittests

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/microsvc/acl"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/testingutil"
)

func TestBuiltinRBAC(t *testing.T) {
	initRDBMSTestingDB(t)
	testingutil.AssertNil(t, acl.InitBuiltinRBACModels(), "EnsureTableStructures error")
	user := builtinmodels.User{}
	role := builtinmodels.Role{}
	module := builtinmodels.Module{}
	roleModle := builtinmodels.RoleModuleRelation{}
	userRole := builtinmodels.UserRoleRelation{}
	passwdHash, err := builtinmodels.GeneratePassword("000000")
	testingutil.AssertNil(t, err, "GeneratePassword error")
	count, err := user.InsertMany([]interface{}{
		&builtinmodels.User{Name: "user1", HashedPassword: base64.StdEncoding.EncodeToString(passwdHash)},
		&builtinmodels.User{Name: "user2", HashedPassword: base64.StdEncoding.EncodeToString(passwdHash)},
	})
	testingutil.AssertEquals(t, int64(2), count, "user.InsertMany count")
	testingutil.AssertNil(t, err, "user.InsertMany error")
	count, err = role.InsertMany([]interface{}{
		&builtinmodels.Role{Name: "role1", Code: "001"},
		&builtinmodels.Role{Name: "role2", Code: "002"},
	})
	testingutil.AssertEquals(t, int64(2), count, "role.InsertMany count")
	testingutil.AssertNil(t, err, "role.InsertMany error")
	count, err = module.InsertMany([]interface{}{
		&builtinmodels.Module{Name: "hello", Code: "m001"},
		&builtinmodels.Module{Name: "view", Code: "m002", ParentID: 1},
		&builtinmodels.Module{Name: "edit", Code: "m003", ParentID: 1},
	})
	testingutil.AssertEquals(t, int64(3), count, "module.InsertMany count")
	testingutil.AssertNil(t, err, "module.InsertMany error")
	count, err = roleModle.InsertMany([]interface{}{
		&builtinmodels.RoleModuleRelation{ModuleID: 1, RoleID: 1},
		&builtinmodels.RoleModuleRelation{ModuleID: 2, RoleID: 1},
		&builtinmodels.RoleModuleRelation{ModuleID: 1, RoleID: 2},
		&builtinmodels.RoleModuleRelation{ModuleID: 3, RoleID: 2},
	})
	testingutil.AssertEquals(t, int64(4), count, "roleModle.InsertMany count")
	testingutil.AssertNil(t, err, "roleModle.InsertMany error")
	count, err = userRole.InsertMany([]interface{}{
		&builtinmodels.UserRoleRelation{UserID: 1, RoleID: 1},
		&builtinmodels.UserRoleRelation{UserID: 2, RoleID: 2},
	})
	testingutil.AssertEquals(t, int64(2), count, "userRole.InsertMany count")
	testingutil.AssertNil(t, err, "userRole.InsertMany error")

	modules, err := builtinmodels.FindAuthorizedModules(1)
	testingutil.AssertNil(t, err, "FindAuthorizedModules error")
	testingutil.AssertEquals(t, 2, len(modules), "FindAuthorizedModules count")
	expectedNames := []string{"hello", "view"}
	for i, mod := range modules {
		testingutil.AssertEquals(t, expectedNames[i], mod.Name, fmt.Sprintf("Module[%d].Name", i))
	}
}
