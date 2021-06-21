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
