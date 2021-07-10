package builtincontrollers

import (
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/utils"
)

// roleController processes with builtin RBAC role business
type roleController struct {
	microsvc.AbstractController
	SerializationStrategy interface{} `serialization:"json"`
	Name                  string
	observers             []string
}

// GetInfo : get role
func (c *roleController) GetInfo(param IDParam) (*builtinmodels.Role, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *builtinmodels.Role
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("get role failed while giving empty role primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		role := builtinmodels.Role{ID: param.ID}
		_, err := role.Fetch()
		if nil != err {
			logger.Error.Printf("get role:%d while fetch role data failed with error:%v", role.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		response = &role
		break
	}
	return response, handlerErr
}

// GetList : get role list
func (c *roleController) GetList(param RoleListQueryParam) (RoleListQueryResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response = RoleListQueryResponse{}
	for {
		role := builtinmodels.Role{}
		if nil != param.Filters {
			utils.FormDataCopyFields(param.Filters, &role, "json")
		}
		count, err := rdbms.GetInstance().FetchRecordsAndCountTotal(&role, param.Pagination.GetPageSize(), param.Pagination.GetPageOffset(), &response.Items)
		if nil != err {
			logger.Error.Printf("get role list while fetch role records failed with error:%v", err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		response.Total = count

		break
	}
	return response, handlerErr
}

// PostAdd : add role
func (c *roleController) PostAdd(param RoleParam) (*RoleResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *RoleResponse
	for {
		role := builtinmodels.Role{}
		affected := utils.FormDataCopyFields(param, &role, "json")
		if 0 >= affected {
			logger.Warning.Printf("add role failed while giving empty role data")
			handlerErr = microsvc.NewHandlerError(results.NothingToDo, "Invalid input data")
			break
		}

		_, err := role.Save()
		if nil != err {
			logger.Error.Printf("add role while save role data %+v failed with error:%v", role, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("add role %s(%s) succeed", role.Name, role.Code)
		response = &RoleResponse{Name: role.Name}
		break
	}
	return response, handlerErr
}

// PostEdit : edit role
func (c *roleController) PostEdit(param RoleParam) (*RoleResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *RoleResponse
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("edit role failed while giving empty role primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		role := builtinmodels.Role{ID: param.ID}
		_, err := role.Fetch()
		if nil != err {
			logger.Error.Printf("edit role:%d while fetch role data failed with error:%v", role.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		affected := utils.FormDataCopyFields(param, &role, "json")
		if 0 >= affected {
			logger.Warning.Printf("edit role failed while giving empty role data")
			handlerErr = microsvc.NewHandlerError(results.NothingToDo, "Nothing to do")
			break
		}

		_, err = role.Save()
		if nil != err {
			logger.Error.Printf("edit role while save role data %+v failed with error:%v", role, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("edit role %s(%s) succeed", role.Name, role.Code)
		response = &RoleResponse{Name: role.Name}
		break
	}
	return response, handlerErr
}

// PostDelete : delete role
func (c *roleController) PostDelete(param IDParam) (*RoleResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *RoleResponse
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("delete role failed while giving empty role primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		role := builtinmodels.Role{ID: param.ID}
		_, err := role.Fetch()
		if nil != err {
			logger.Error.Printf("delete role:%d while fetch role data failed with error:%v", role.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		roleModule := builtinmodels.RoleModuleRelation{RoleID: role.ID}
		count, err := roleModule.Count()
		if nil != err {
			logger.Error.Printf("delete role:%d while find role referenced modules count failed with error:%v", role.ID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}
		if 0 < count {
			logger.Error.Printf("delete role:%d while find role referenced modules were not empty", role.ID)
			handlerErr = microsvc.NewHandlerError(results.NotEmptyReferences, "Referenced modules were not empty")
			break
		}
		userRole := builtinmodels.UserRoleRelation{RoleID: role.ID}
		count, err = userRole.Count()
		if nil != err {
			logger.Error.Printf("delete role:%d while find role referenced user count failed with error:%v", role.ID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}
		if 0 < count {
			logger.Error.Printf("delete role:%d while find role referenced users were not empty", role.ID)
			handlerErr = microsvc.NewHandlerError(results.NotEmptyReferences, "Referenced users were not empty")
			break
		}

		_, err = role.Delete()
		if nil != err {
			logger.Error.Printf("delete role:%d failed with error:%v", role.ID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("delete role %s(%s) succeed", role.Name, role.Code)
		response = &RoleResponse{Name: role.Name}
		break
	}
	return response, handlerErr
}

// PostRelationModules : set role related modules
func (c *roleController) PostRelationModules(param RoleModuleRelationParam) (RelationsResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response RelationsResponse
	for {
		if 0 >= param.RoleID {
			logger.Warning.Printf("set role modules failed while giving empty role primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		user := builtinmodels.Role{ID: param.RoleID}
		ok, err := user.Exists()
		if false == ok {
			logger.Error.Printf("set role:%d modules:%+v while the role does not exists error:%v", param.RoleID, param.ModuleIDs, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, "User does not exists")
			break
		}

		userRoles := builtinmodels.RoleModuleRelation{RoleID: param.RoleID}
		rows, err := rdbms.GetInstance().FetchAll(&userRoles)
		if nil != err {
			logger.Error.Printf("set role:%d modules:%+v while fetch exists relations failed with error:%v", param.RoleID, param.ModuleIDs, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		existsIDs := map[int64]*builtinmodels.RoleModuleRelation{}
		if nil != rows {
			for _, row := range rows {
				ur := row.(*builtinmodels.RoleModuleRelation)
				existsIDs[ur.ModuleID] = ur
			}
		}
		rmModel := builtinmodels.RoleModuleRelation{}
		addRecords := []interface{}{}
		response.Adds = []int64{}
		response.Deletes = []int64{}
		if nil != param.ModuleIDs {
			for _, id := range param.ModuleIDs {
				if nil != existsIDs[id] {
					delete(existsIDs, id)
					continue
				}
				addRecords = append(addRecords, &builtinmodels.RoleModuleRelation{
					RoleID:   param.RoleID,
					ModuleID: id,
					SystemID: param.SystemID,
				})
				response.Adds = append(response.Adds, id)
			}
		}

		for _, rm := range existsIDs {
			response.Deletes = append(response.Deletes, rm.ID)
		}
		if len(response.Deletes) > 0 {
			eng, err := rdbms.GetInstance().GetDbEngine(&rmModel)
			if nil != err {
				logger.Error.Printf("set role:%d modules:%+v while get db engine failed with error:%v", param.RoleID, param.ModuleIDs, err)
				handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
				break
			}
			count, err := eng.In("id", response.Deletes).Delete(&rmModel)
			if nil != err {
				logger.Error.Printf("set role:%d modules:%+v while delete old relations %+v failed with error:%v", param.RoleID, param.ModuleIDs, response.Deletes, err)
				handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
				break
			} else {
				logger.Info.Printf("set role:%d modules:%+v deleted old relations %+v affected %d rows", param.RoleID, param.ModuleIDs, response.Deletes, count)
			}
		}

		if len(addRecords) > 0 {
			count, err := rmModel.InsertMany(addRecords)
			if nil != err {
				logger.Error.Printf("set role:%d modules:%+v while add new relations failed with error:%v", param.RoleID, param.ModuleIDs, err)
				handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
				break
			} else {
				logger.Info.Printf("set role:%d modules:%+v add new relations affected %d rows", param.RoleID, param.ModuleIDs, count)
			}
		}
		logger.Info.Printf("set role:%d modules:%+v succeed", param.RoleID, param.ModuleIDs)
		break
	}
	return response, handlerErr
}
