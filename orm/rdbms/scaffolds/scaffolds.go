package scaffolds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/kevinyjn/gocom/utils"
	"github.com/kevinyjn/gocom/yamlutils"
)

// ProjectScaffolds scaffolding of project directories
type ProjectScaffolds struct {
	Name                string   `yaml:"name" json:"name"`
	BaseDir             string   `yaml:"baseDir" json:"baseDir"`
	EtcFolder           string   `yaml:"etcFolder" json:"etcFolder"`
	ModelsFolder        string   `yaml:"modelsFolder" json:"modelsFolder"`
	ControllersFolder   string   `yaml:"controllersFolder" json:"controllersFolder"`
	MiddlewaresFolder   string   `yaml:"middlewaresFolder" json:"middlewaresFolder"`
	ServicesFolder      string   `yaml:"servicesFolder" json:"servicesFolder"`
	RepositoriesFolder  string   `yaml:"repositoriesFolder" json:"repositoriesFolder"`
	Datasources         []string `yaml:"datasources" json:"datasources"`
	DockerfileTemplate  string   `yaml:"dockerfileTempl" json:"dockerfileTempl"`
	JenkinsfileTemplate string   `yaml:"jenkinsfileTempl" json:"jenkinsfileTempl"`
	EtcFileTemplates    []string `yaml:"etcFileTemplates" json:"etcFileTemplates"`
	creatingDirs        map[string]string
	goVersion           string
}

// NewProjectScaffolds new a project scaffolds
func NewProjectScaffolds(projectName string, projectDir string) *ProjectScaffolds {
	if "" == projectDir {
		projectDir = "../"
	}
	if "" == projectName {
		projectName = "newproject"
	}
	s := &ProjectScaffolds{
		Name:                projectName,
		BaseDir:             path.Join(projectDir, projectName),
		EtcFolder:           "etc",
		ModelsFolder:        "models",
		ControllersFolder:   "controllers",
		MiddlewaresFolder:   "middlewares",
		ServicesFolder:      "services",
		Datasources:         []string{"default"},
		creatingDirs:        map[string]string{},
		goVersion:           "1.14",
		DockerfileTemplate:  "",
		JenkinsfileTemplate: "",
		// RepositoriesFolder: "repositories",
	}
	return s
}

// LoadFromYaml load from configure
func (s *ProjectScaffolds) LoadFromYaml(filePath string) error {
	return yamlutils.LoadConfig(filePath, s)
}

// LoadFromJSON load from configure
func (s *ProjectScaffolds) LoadFromJSON(filePath string) error {
	confContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Could not load configure file ", filePath)
		return err
	}
	return json.Unmarshal(confContent, s)
}

// Generate genetates project directory and base source files
func (s *ProjectScaffolds) Generate() error {
	err := s.EnsureDirecotries()
	if nil != err {
		return err
	}

	for k, d := range s.creatingDirs {
		switch k {
		case "etcFolder":
			err = s.createGitignoreFiles(d, []string{"local.*", "test.*"})
			err = s.createEtcFiles(d)
			break
		case "modelsFolder":
			err = s.createGitkeepFiles(d)
			break
		case "controllersFolder":
			err = s.createGitkeepFiles(d)
			break
		case "middlewaresFolder":
			err = s.createGitkeepFiles(d)
			break
		case "servicesFolder":
			err = s.createGitkeepFiles(d)
			break
		case "repositoriesFolder":
			err = s.createGitkeepFiles(d)
			break
		}
		if nil != err {
			break
		}
	}

	if nil != err {
		return err
	}

	err = s.createGoModFiles(path.Join(s.BaseDir, "src"))
	err = s.createGitignoreFiles(s.BaseDir, nil)
	err = s.createDockerignoreFiles(s.BaseDir, nil)

	params := s.getTemplateParams()
	if "" != s.DockerfileTemplate {
		err = s.createFileByTemplate(s.BaseDir, "Dockerfile", s.DockerfileTemplate, params)
	}
	if "" != s.JenkinsfileTemplate {
		err = s.createFileByTemplate(s.BaseDir, "Jenkinsfile", s.JenkinsfileTemplate, params)
	}

	return err
}

// EnsureDirecotries ensure creating directories
func (s *ProjectScaffolds) EnsureDirecotries() error {
	var err error
	var creatingDirs = map[string]string{}
	err = utils.EnsureDirectory(s.BaseDir)
	if nil != err {
		return err
	}
	s.ensureRequiredDirectories()
	v := reflect.ValueOf(s).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.IsValid() && field.Type().Kind() == reflect.String {
			if "" == field.String() {
				continue
			}
			tagValue := v.Type().Field(i).Tag.Get("yaml")
			var subPath string
			if "" == tagValue || "-" == tagValue || !strings.HasSuffix(tagValue, "Folder") {
				continue
			} else if "etcFolder" == tagValue {
				subPath = path.Join(s.BaseDir, field.String())
			} else {
				subPath = path.Join(s.BaseDir, "src", field.String())
			}
			creatingDirs[tagValue] = subPath
		}
	}
	for tagName, d := range creatingDirs {
		err = utils.EnsureDirectory(d)
		if nil != err {
			break
		}
		_ = tagName
	}
	s.creatingDirs = creatingDirs
	return nil
}

// GetProjectSubFolderPath get path by name
func (s *ProjectScaffolds) GetProjectSubFolderPath(tagName string) string {
	folderPath := s.creatingDirs[tagName]
	return folderPath
}

// GenerateModel by model defination
func (s *ProjectScaffolds) GenerateModel(model *ModelScaffolds) error {
	filePath := path.Join(s.creatingDirs["modelsFolder"], model.Path)
	return writeFileContents(filePath, model.Encode())
}

func (s *ProjectScaffolds) getTemplateParams() map[string]interface{} {
	params := map[string]interface{}{
		"Name":      s.Name,
		"GoVersion": s.goVersion,
	}
	return params
}

func (s *ProjectScaffolds) ensureRequiredDirectories() error {
	if "" == s.EtcFolder {
		s.EtcFolder = "etc"
	}
	if "" == s.ModelsFolder {
		s.ModelsFolder = "models"
	}
	if "" == s.ControllersFolder {
		s.ControllersFolder = "controllers"
	}
	return nil
}

func (s *ProjectScaffolds) createEtcFiles(folderPath string) error {
	err := writeFileContents(path.Join(folderPath, "version"), "0.0.1")
	if nil != err {
		return err
	}
	if nil != s.EtcFileTemplates {
		params := s.getTemplateParams()
		for _, fname := range s.EtcFileTemplates {
			filePath := path.Join(folderPath, fname)
			_, err = os.Stat(filePath)
			if os.IsExist(err) {
				if strings.HasSuffix(fname, ".tpl") {
					fname = fname[:len(fname)-4]
				}
				err = s.createFileByTemplate(folderPath, fname, filePath, params)
			}
		}
	}
	return nil
}

func (s *ProjectScaffolds) createGitkeepFiles(folderPath string) error {
	filePath := path.Join(folderPath, ".gitkeep")
	return writeFileContents(filePath, "")
}

func (s *ProjectScaffolds) createGitignoreFiles(folderPath string, lines []string) error {
	filePath := path.Join(folderPath, ".gitignore")
	if len(lines) <= 0 {
		lines = []string{
			"node_modules",
			"package-lock.json",
			"tests",
			".DS_Store",
			".project",
			".vscode",
			"target",
			"log",
			"pkg",
			"uploads",
			"data",
			"__debug_bin",
			"*.tar",
		}
	}
	return writeFileContents(filePath, strings.Join(lines, "\n"))
}

func (s *ProjectScaffolds) createDockerignoreFiles(folderPath string, lines []string) error {
	filePath := path.Join(folderPath, ".dockerignore")
	if len(lines) <= 0 {
		lines = []string{
			"node_modules",
			".gitignore",
			"*.md",
			"Dockerfile",
			"scripts",
			"log",
			"docs",
			".vscode",
			"target",
			"local.*",
			"uploads",
		}
	}
	return writeFileContents(filePath, strings.Join(lines, "\n"))
}

func (s *ProjectScaffolds) createGoModFiles(folderPath string) error {
	filePath := path.Join(folderPath, "go.mod")
	ver := "1.14"
	r := regexp.MustCompile(`(\d+\.\d+)`)
	found := r.FindAllStringSubmatch(runtime.Version(), 1)
	if len(found) > 0 {
		ver = found[0][0]
	}
	s.goVersion = ver
	lines := []string{
		fmt.Sprintf("module \"%s.project\"\n", s.Name),
		fmt.Sprintf("go %s\n", ver),
		"require (",
		"\tgithub.com/kevinyjn/gocom master",
		")\n",
	}
	return writeFileContents(filePath, strings.Join(lines, "\n"))
}

func (s *ProjectScaffolds) createFileByTemplate(folderPath string, fileName string, templPath string, params map[string]interface{}) error {
	filePath := path.Join(folderPath, fileName)
	tpl, err := template.ParseFiles(templPath)
	if nil != err {
		return err
	}
	fl, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if nil != err {
		return err
	}
	var wt = bytes.Buffer{}
	err = tpl.Execute(&wt, params)
	if nil != err {
		return err
	}
	fl.Write(wt.Bytes())
	return nil
}

func writeFileContents(filePath, contents string) error {
	fl, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if nil != err {
		return err
	}
	defer fl.Close()
	_, err = fl.WriteString(contents)
	return err
}
