package builtincontrollers

import (
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/utils"
)

// moduleController processes with builtin RBAC module business
type moduleController struct {
	microsvc.AbstractController
	SerializationStrategy interface{} `serialization:"json"`
	Name                  string
	observers             []string
}

// GetInfo : get module
func (c *moduleController) GetInfo(param IDParam) (*builtinmodels.Module, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *builtinmodels.Module
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("get module failed while giving empty module primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		module := builtinmodels.Module{ID: param.ID}
		_, err := module.Fetch()
		if nil != err {
			logger.Error.Printf("get module:%d while fetch module data failed with error:%v", module.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		response = &module
		break
	}
	return response, handlerErr
}

// GetList : get module list
func (c *moduleController) GetList(param ModuleListQueryParam) (ModuleListQueryResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response = ModuleListQueryResponse{}
	for {
		module := builtinmodels.Module{}
		if nil != param.Filters {
			utils.FormDataCopyFields(param.Filters, &module, "json")
		}
		count, err := rdbms.GetInstance().FetchRecordsAndCountTotal(&module, param.Pagination.GetPageSize(), param.Pagination.GetPageOffset(), &response.Items)
		if nil != err {
			logger.Error.Printf("get module list while fetch module records failed with error:%v", err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		response.Total = count

		break
	}
	return response, handlerErr
}

// PostAdd : add module
func (c *moduleController) PostAdd(param ModuleParam) (*ModuleResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *ModuleResponse
	for {
		module := builtinmodels.Module{}
		affected := utils.FormDataCopyFields(param, &module, "json")
		if 0 >= affected {
			logger.Warning.Printf("add module failed while giving empty module data")
			handlerErr = microsvc.NewHandlerError(results.NothingToDo, "Invalid input data")
			break
		}

		_, err := module.Save()
		if nil != err {
			logger.Error.Printf("add module while save module data %+v failed with error:%v", module, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("add module %s(%s) succeed", module.Name, module.Code)
		response = &ModuleResponse{Name: module.Name}
		break
	}
	return response, handlerErr
}

// PostEdit : edit module
func (c *moduleController) PostEdit(param ModuleParam) (*ModuleResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *ModuleResponse
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("edit module failed while giving empty module primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		module := builtinmodels.Module{ID: param.ID}
		_, err := module.Fetch()
		if nil != err {
			logger.Error.Printf("edit module:%d while fetch module data failed with error:%v", module.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		affected := utils.FormDataCopyFields(param, &module, "json")
		if 0 >= affected {
			logger.Warning.Printf("edit module failed while giving empty module data")
			handlerErr = microsvc.NewHandlerError(results.NothingToDo, "Nothing to do")
			break
		}

		_, err = module.Save()
		if nil != err {
			logger.Error.Printf("edit module while save module data %+v failed with error:%v", module, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("edit module %s(%s) succeed", module.Name, module.Code)
		response = &ModuleResponse{Name: module.Name}
		break
	}
	return response, handlerErr
}

// PostDelete : delete module
func (c *moduleController) PostDelete(param IDParam) (*ModuleResponse, microsvc.HandlerError) {
	var handlerErr microsvc.HandlerError
	var response *ModuleResponse
	for {
		if 0 >= param.ID {
			logger.Warning.Printf("delete module failed while giving empty module primary key")
			handlerErr = microsvc.NewHandlerError(results.InvalidInput, "Invalid input data")
			break
		}
		module := builtinmodels.Module{ID: param.ID}
		_, err := module.Fetch()
		if nil != err {
			logger.Error.Printf("delete module:%d while fetch module data failed with error:%v", module.ID, err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}

		roleModule := builtinmodels.RoleModuleRelation{ModuleID: module.ID}
		count, err := roleModule.Count()
		if nil != err {
			logger.Error.Printf("delete module:%d while find module referenced modules count failed with error:%v", module.ID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}
		if 0 < count {
			logger.Error.Printf("delete module:%d while find module referenced modules were not empty", module.ID)
			handlerErr = microsvc.NewHandlerError(results.NotEmptyReferences, "Referenced modules were not empty")
			break
		}

		_, err = module.Delete()
		if nil != err {
			logger.Error.Printf("delete module:%d failed with error:%v", module.ID, err)
			handlerErr = microsvc.NewHandlerError(results.InnerError, err.Error())
			break
		}

		logger.Info.Printf("delete module %s(%s) succeed", module.Name, module.Code)
		response = &ModuleResponse{Name: module.Name}
		break
	}
	return response, handlerErr
}
