package yamlutils

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

// LoadConfig loader
func LoadConfig(filePath string, v interface{}) error {
	confContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		if !strings.Contains(filePath, "local.") {
			fmt.Println("Could not load configure file ", filePath)
		}
		return err
	}
	fmt.Println("Loading configure file ", filePath)
	err = yaml.Unmarshal(confContent, v)
	if err != nil {
		fmt.Println("Could not load configure file ", filePath, ", ", err)
		return err
	}

	return nil
}
