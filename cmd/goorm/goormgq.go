package main

/*
Commandline tool for generating ORM GraphQL api project in go
*/

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kevinyjn/gocom/orm/rdbms/scaffolds"
	"github.com/kevinyjn/gocom/utils"
	"gopkg.in/yaml.v2"
)

func main() {
	projectCmd := flag.NewFlagSet("create", flag.ExitOnError)
	projectConfig := projectCmd.String("f", "", "Project creating config file")

	modelCmd := flag.NewFlagSet("model", flag.ExitOnError)
	modelConfig := modelCmd.String("f", "", "Model creating config file")
	modelDir := modelCmd.String("d", "", "Model destination save directory")

	if len(os.Args) < 2 {
		fmt.Println("Expected 'create' or 'model' commands")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "create":
		err = projectCmd.Parse(os.Args[2:])
		if nil != err {
			fmt.Println(err)
		}
		err = doCreateProject(*projectConfig, projectCmd.Args())
		break
	case "model":
		err = modelCmd.Parse(os.Args[2:])
		err = doCreateModel(*modelConfig, *modelDir, modelCmd.Args())
		if nil != err {
			fmt.Println(err)
		}
		break
	default:
		err = fmt.Errorf("Unsupported command '%s'", os.Args[0])
		fmt.Println(err)
		break
	}

	if nil != err {
		os.Exit(1)
	}
}

func doCreateProject(configFile string, args []string) error {
	projectName := "example"
	destDir := ""
	if len(args) > 0 {
		projectName = strings.Trim(args[0], "-")
	}
	if len(args) > 1 {
		destDir = strings.Trim(args[1], "-")
	}

	s := scaffolds.NewProjectScaffolds(projectName, destDir)
	if "" != configFile {
		err := unmarshalConfig(configFile, s)
		if nil != err {
			return err
		}
	}

	inputReader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter project name:(default %s)", s.Name)
	r0, err := inputReader.ReadString('\n')
	if nil != err {
		return err
	}
	if "" == r0 {
		s.Name = r0
	}

	fmt.Printf("Enter project path:(default %s)", s.BaseDir)
	r0, err = inputReader.ReadString('\n')
	if nil != err {
		return err
	}
	if "" == r0 {
		s.BaseDir = r0
	}

	if "" == s.DockerfileTemplate {
		fmt.Printf("Enter Dockerfile template path:(omit empty to skip create Dockerfile)")
		r0, err = inputReader.ReadString('\n')
		if nil != err {
			return err
		}
		if "" == r0 {
			if utils.IsPathNotExists(r0) {
				err = fmt.Errorf("'%s' does not exists", r0)
				fmt.Println(err)
				return err
			}
			s.DockerfileTemplate = r0
		}
	}
	if "" == s.JenkinsfileTemplate {
		fmt.Printf("Enter Jenkinsfile template path:(omit empty to skip create Jenkinsfile)")
		r0, err = inputReader.ReadString('\n')
		if nil != err {
			return err
		}
		if "" == r0 {
			if utils.IsPathNotExists(r0) {
				err = fmt.Errorf("'%s' does not exists", r0)
				fmt.Println(err)
				return err
			}
			s.JenkinsfileTemplate = r0
		}
	}

	err = s.Generate()
	if nil != err {
		fmt.Printf("Generate project:%s failed with error:%v", s.Name, err)
		return err
	}

	return nil
}

func doCreateModel(configFile string, modelDir string, args []string) error {
	s := scaffolds.ProjectScaffolds{
		ModelsFolder: modelDir,
	}

	inputReader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter model save path:(default %s)", s.ModelsFolder)
	r0, err := inputReader.ReadString('\n')
	if nil != err {
		return err
	}
	if "" == r0 {
		s.ModelsFolder = r0
	}

	m := &scaffolds.ModelScaffolds{}
	var readEnter = true
	if "" != configFile {
		err := unmarshalConfig(configFile, m)
		if nil != err {
			return err
		}

		fmt.Printf("Create model using this configure file?(Y[es]|N[o])\n")
		r0, err = inputReader.ReadString('\n')
		if nil != err {
			return err
		}
		if "" == r0 {
			if strings.ToLower(string(r0[0:1])) == "y" {
				readEnter = false
			} else if strings.ToLower(string(r0[0:1])) == "n" {
				readEnter = true
			}
		}
	}

	if readEnter {
		// todo
		err = fmt.Errorf("Read model parameters from keyboards enter not supported")
		fmt.Println(err)
		return err
	}

	err = s.GenerateModel(m)
	if nil != err {
		fmt.Printf("Generate model failed with error:%v", err)
		return err
	}

	return nil
}

func unmarshalConfig(filePath string, v interface{}) error {
	if "" == filePath {
		return errors.New("No configure file specified")
	}

	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		if !strings.Contains(filePath, "local.") {
			fmt.Println("Could not load configure file ", filePath)
		}
		return err
	}

	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		err = yaml.Unmarshal(fileContent, v)
	} else if strings.HasSuffix(filePath, ".json") {
		err = json.Unmarshal(fileContent, v)
	} else {
		err = fmt.Errorf("Unsupported config file '%s'", filePath)
		fmt.Println(err)
	}

	if nil != err {
		fmt.Printf("Load config file %s failed with error:%v\n", filePath, err)
		return err
	}

	return nil
}
