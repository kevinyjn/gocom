package builtincontrollers

import (
	"fmt"
	"strconv"

	"github.com/kataras/iris"
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/microsvc/parameters"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/utils"
)

// UserSessionController interface of current user session
type UserSessionController interface {
	PushUID(ctx iris.Context, uid string)
	PushAppID(ctx iris.Context, appID string)
	CurrentUserID(ctx iris.Context) string
	CurrentUserAppID(ctx iris.Context) string
}

// userController processes with builtin RBAC user business
type userController struct {
	microsvc.AbstractController
	SerializationStrategy interface{} `serialization:"json"`
	Name                  string
	observers             []string
}

// var _userSessionManager UserSessionController

// GetCurrent : get current user
func (c *userController) GetCurrent(msg mqenv.MQConsumerMessage) (*builtinmodels.User, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *builtinmodels.User
	for {
		if "" == msg.UserID {
			logger.Warning.Printf("get current user failed while user not login")
			handlerErr = microsvc.NewHandlerError(results.Unauthorized, "Unauthorized")
			break
		}
		userID, err := strconv.ParseInt(msg.UserID, 10, 64)
		if nil != err {
			logger.Warning.Printf("get current user failed while giving invalid user id:%s error:%v", msg.UserID, err)
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, err.Error())
			break
		}
		user := builtinmodels.User{ID: userID}
		_, err = user.Fetch()
		if nil != err {
			logger.Error.Printf("get current user:%d while fetch user data failed with error:%v", user.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		response = &user
		break
	}
	return response, handlerErr
}

// // PostLogin : login user
// func (c *userController) PostLogin(param LoginParam) (*UserResponse, microsvc.HandlerError) {
// 	var handlerErr microsvc.HandlerError
// 	var response *UserResponse
// 	for {
// 		if "" == param.Name {
// 			logger.Warning.Printf("login user failed while giving empty user name")
// 			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
// 			break
// 		}
// 		if nil == _userSessionManager {
// 			logger.Error.Printf("login user:%s while user session controller not configured", param.Name)
// 			handlerErr = microsvc.NewHandlerError(results.InnerError, "User session manager not configured")
// 			break
// 		}

// 		user := builtinmodels.User{Name: param.Name}
// 		_, err := user.Fetch()
// 		if nil != err {
// 			logger.Error.Printf("login user:%s while fetch user data failed with error:%v", param.Name, err)
// 			handlerErr = microsvc.NewHandlerError(results.DataNotExists, "User does not exists")
// 			break
// 		}

// 		passwordHash, err := builtinmodels.GenerateHashedPassword(param.Password)
// 		if nil != err {
// 			logger.Error.Printf("login user:%s while process inputs password:%s failed with error:%v", param.Name, param.Password, err)
// 			handlerErr = microsvc.NewHandlerError(results.InvalidInput, err.Error())
// 			break
// 		}
// 		if false == user.VerifyPassword(passwordHash) {
// 			logger.Error.Printf("login user:%s while user password:%s were not valid", param.Name, param.Password)
// 			handlerErr = microsvc.NewHandlerError(results.InvalidPassword, err.Error())
// 			break
// 		}

// 		logger.Info.Printf("login user %s(%d) succeed", user.Name, user.ID)
// 		response = &UserResponse{Name: user.Name}
// 		break
// 	}
// 	return response, handlerErr
// }

// GetInfo : get user
func (c *userController) GetInfo(param parameters.IDParam) (*builtinmodels.User, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *builtinmodels.User
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("get user failed while giving empty user primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		user := builtinmodels.User{ID: param.ID}
		_, err := user.Fetch()
		if nil != err {
			logger.Error.Printf("get user:%d while fetch user data failed with error:%v", user.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		response = &user
		break
	}
	return response, handlerErr
}

// GetList : get user list
func (c *userController) GetList(param UserListQueryParam) (UserListQueryResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response = UserListQueryResponse{}
	for {
		user := builtinmodels.User{}
		if nil != param.Filters {
			utils.FormDataCopyFields(param.Filters, &user, "json")
		}
		count, err := rdbms.GetInstance().FetchRecordsAndCountTotal(&user, param.GetPageSize(), param.GetPageOffset(), &response.Items)
		if nil != err {
			logger.Error.Printf("get user list while fetch user records failed with error:%v", err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		response.Total = count
		if nil != response.Items {
			urr := builtinmodels.UserRoleRelation{}
			role := builtinmodels.Role{}
			roleRows, err := rdbms.GetInstance().QueryRelationData(rdbms.RelationQuery{
				Select:              fmt.Sprintf("%s.user_id AS uid, %s.id AS id, %s.name AS name", urr.TableName(), role.TableName(), role.TableName()),
				RelationTable:       &urr,
				TargetTable:         &role,
				SelfRelationField:   "user_id",
				TargetRelationField: "role_id",
				TargetPrimaryKey:    "id",
			}, "ID", "Roles", response.Items)
			if nil != err {
				logger.Error.Printf("find user ralated role failed with error:%v", err)
				break
			}

			roleNames := map[interface{}][]builtinmodels.NameInfo{}
			ok := false
			foundExists := false
			for _, row := range roleRows {
				uid := row["uid"]
				id := row["id"]
				name := utils.ToString(row["name"])
				_, ok = roleNames[uid]
				if ok {
					foundExists = false
					for _, o := range roleNames[uid] {
						if o.ID == id {
							foundExists = true
							break
						}
					}
					if false == foundExists {
						roleNames[uid] = append(roleNames[uid], builtinmodels.NameInfo{ID: id, Name: name})
					}
				} else {
					roleNames[uid] = []builtinmodels.NameInfo{builtinmodels.NameInfo{ID: id, Name: name}}
				}
			}

			for _, o := range response.Items {
				roles, ok := roleNames[o.ID]
				if ok {
					o.Roles = roles
				}
			}
		}

		break
	}
	return response, handlerErr
}

// PostAdd : add user
func (c *userController) PostAdd(param UserParam) (*UserResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *UserResponse
	for {
		user := builtinmodels.User{}
		affected := utils.FormDataCopyFields(param, &user, "json")
		if 0 >= affected {
			logger.Warning.Printf("add user failed while giving empty user data")
			handlerErr = microsvc.NewHandlerError(results.NothingToDo, "Invalid input data")
			break
		}

		passwordHash, err := builtinmodels.GenerateHashedPassword(param.Password)
		if nil != err {
			logger.Error.Printf("add user:%s while process inputs password:%s failed with error:%v", param.Name, param.Password, err)
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, err.Error())
			break
		}
		user.HashedPassword = passwordHash

		_, err = user.Save()
		if nil != err {
			logger.Error.Printf("add user while save user data %+v failed with error:%v", user, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("add user %s(%d) succeed", user.Name, user.ID)
		response = &UserResponse{Name: user.Name}
		break
	}
	return response, handlerErr
}

// PostEdit : edit user
func (c *userController) PostEdit(param UserParam) (*UserResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *UserResponse
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("edit user failed while giving empty user primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		user := builtinmodels.User{ID: param.ID}
		_, err := user.Fetch()
		if nil != err {
			logger.Error.Printf("edit user:%d while fetch user data failed with error:%v", user.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		affected := utils.FormDataCopyFields(param, &user, "json")
		passwordHash, err := builtinmodels.GenerateHashedPassword(param.Password)
		if nil != err {
			logger.Error.Printf("set user:%s while process inputs password:%s failed with error:%v", param.Name, param.Password, err)
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, err.Error())
			break
		}
		if passwordHash != user.HashedPassword {
			user.HashedPassword = passwordHash
			affected++
		}
		if 0 >= affected {
			logger.Warning.Printf("edit user failed while giving empty user data")
			handlerErr = microsvc.NewHandlerError(results.NothingToDo, "Nothing to do")
			break
		}

		_, err = user.Save()
		if nil != err {
			logger.Error.Printf("edit user while save user data %+v failed with error:%v", user, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("edit user %s(%d) succeed", user.Name, user.ID)
		response = &UserResponse{Name: user.Name}
		break
	}
	return response, handlerErr
}

// PostDelete : delete user
func (c *userController) PostDelete(param parameters.IDParam) (*UserResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *UserResponse
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("delete user failed while giving empty user primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		user := builtinmodels.User{ID: param.ID}
		_, err := user.Fetch()
		if nil != err {
			logger.Error.Printf("delete user:%d while fetch user data failed with error:%v", user.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		_, err = user.Delete()
		if nil != err {
			logger.Error.Printf("delete user:%d failed with error:%v", user.ID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("delete user %s(%d) succeed", user.Name, user.ID)
		response = &UserResponse{Name: user.Name}
		break
	}
	return response, handlerErr
}

// GetModules : get current user authed modules
func (c *userController) GetModules(msg mqenv.MQConsumerMessage) ([]builtinmodels.Module, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response []builtinmodels.Module
	for {
		if "" == msg.UserID {
			logger.Warning.Printf("get current user authorized modules while could not get current user id")
			handlerErr = microsvc.NewHandlerError(results.NotLogin, "User not login")
			break
		}

		userID, err := strconv.ParseInt(msg.UserID, 10, 64)
		if nil != err {
			logger.Error.Printf("get current user:%s authorized modules while the user id were not corrrect error:%v", msg.UserID, err)
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, err.Error())
			break
		}
		modules, err := builtinmodels.FindAuthorizedModules(userID)
		if nil != err {
			logger.Error.Printf("get current user:%s authorized modules while query authorized modules error:%v", msg.UserID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}
		response = modules
		break
	}
	return response, handlerErr
}

// PostRelationRoles : set user related role
func (c *userController) PostRelationRoles(param UserRoleRelationParam) (RelationsResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response RelationsResponse
	for {
		if 0 >= param.UserID {
			logger.Warning.Printf("set user role failed while giving empty user primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		user := builtinmodels.User{ID: param.UserID}
		ok, err := user.Exists()
		if false == ok {
			logger.Error.Printf("set user:%d roles:%+v while the user does not exists error:%v", param.UserID, param.RoleIDs, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, "User does not exists")
			break
		}

		userRoles := builtinmodels.UserRoleRelation{UserID: param.UserID}
		rows, err := rdbms.GetInstance().FetchAll(&userRoles)
		if nil != err {
			logger.Error.Printf("set user:%d roles:%+v while fetch exists relations failed with error:%v", param.UserID, param.RoleIDs, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		existsIDs := map[int64]*builtinmodels.UserRoleRelation{}
		if nil != rows {
			for _, row := range rows {
				ur := row.(*builtinmodels.UserRoleRelation)
				existsIDs[ur.RoleID] = ur
			}
		}
		urModel := builtinmodels.UserRoleRelation{}
		addRecords := []interface{}{}
		response.Adds = []int64{}
		response.Deletes = []int64{}
		if nil != param.RoleIDs {
			for _, id := range param.RoleIDs {
				if nil != existsIDs[id] {
					delete(existsIDs, id)
					continue
				}
				addRecords = append(addRecords, &builtinmodels.UserRoleRelation{
					UserID:   param.UserID,
					RoleID:   id,
					SystemID: param.SystemID,
				})
				response.Adds = append(response.Adds, id)
			}
		}

		for _, ur := range existsIDs {
			response.Deletes = append(response.Deletes, ur.ID)
		}
		if len(response.Deletes) > 0 {
			eng, err := rdbms.GetInstance().GetDbEngine(&urModel)
			if nil != err {
				logger.Error.Printf("set user:%d roles:%+v while get db engine failed with error:%v", param.UserID, param.RoleIDs, err)
				handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
				break
			}
			count, err := eng.In("id", response.Deletes).Delete(&urModel)
			if nil != err {
				logger.Error.Printf("set user:%d roles:%+v while delete old relations %+v failed with error:%v", param.UserID, param.RoleIDs, response.Deletes, err)
				handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
				break
			} else {
				logger.Info.Printf("set user:%d roles:%+v deleted old relations %+v affected %d rows", param.UserID, param.RoleIDs, response.Deletes, count)
			}
		}

		if len(addRecords) > 0 {
			count, err := urModel.InsertMany(addRecords)
			if nil != err {
				logger.Error.Printf("set user:%d roles:%+v while add new relations failed with error:%v", param.UserID, param.RoleIDs, err)
				handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
				break
			} else {
				logger.Info.Printf("set user:%d roles:%+v add new relations affected %d rows", param.UserID, param.RoleIDs, count)
			}
		}
		logger.Info.Printf("set user:%d roles:%+v succeed", param.UserID, param.RoleIDs)
		break
	}
	return response, handlerErr
}
