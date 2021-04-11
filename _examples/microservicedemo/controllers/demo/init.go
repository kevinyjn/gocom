package demo

import "github.com/kevinyjn/gocom/microsvc"

// GetControllers in current package
func GetControllers() []microsvc.MQController {
	return []microsvc.MQController{
		&demoController{},
	}
}
