package builtincontrollers

import (
	"github.com/kevinyjn/gocom/config/results"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/microsvc"
	"github.com/kevinyjn/gocom/microsvc/builtinmodels"
	"github.com/kevinyjn/gocom/orm/rdbms"
	"github.com/kevinyjn/gocom/utils"
)

// userController processes with builtin RBAC user business
type userController struct {
	microsvc.AbstractController
	SerializationStrategy interface{} `serialization:"json"`
	Name                  string
	observers             []string
}

// GetInfo : get user
func (c *userController) GetInfo(param IDParam) (*builtinmodels.User, microsvc.HandlerError) {
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
		count, err := rdbms.GetInstance().FetchRecordsAndCountTotal(&user, param.Pagination.GetPageSize(), param.Pagination.GetPageOffset(), &response.Items)
		if nil != err {
			logger.Error.Printf("get user list while fetch user records failed with error:%v", err)
			handlerErr = microsvc.NewHandlerError(results.DataNotExists, err.Error())
			break
		}
		response.Total = count

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

		_, err := user.Save()
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
func (c *userController) PostDelete(param IDParam) (*UserResponse, microsvc.HandlerError) {
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
