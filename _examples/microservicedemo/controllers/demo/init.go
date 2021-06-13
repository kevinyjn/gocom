package demo

import "github.com/kevinyjn/gocom/microsvc"

// GetControllers in current package
func GetControllers() []microsvc.Controller {
	return []microsvc.Controller{
		&demoController{},
	}
}
