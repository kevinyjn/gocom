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
	"time"

	"github.com/kevinyjn/gocom/logger"
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
	AppVersion          string   `yaml:"appVersion" json:"appVersion"`
	ConfigFile          string   `yaml:"configFile" json:"configFile"`
	MQConfigFile        string   `yaml:"mqConfigFile" json:"mqConfigFile"`
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
		goVersion:           getGolangVersion(),
		DockerfileTemplate:  "",
		JenkinsfileTemplate: "",
		AppVersion:          "0.0.1",
		ConfigFile:          "../etc/config.yaml",
		MQConfigFile:        "../etc/mq.yaml",
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
			err = s.updateModelsIndexFile(d, "")
			break
		case "controllersFolder":
			err = s.createGitkeepFiles(d)
			err = s.createControllerIndexSourcefiles(d)
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
	err = s.createEntrypointSources(path.Join(s.BaseDir, "src"))

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
	err := writeFileContents(filePath, model.Encode(), false)
	if nil != err {
		return err
	}
	modelPath := path.Dir(filePath)
	return s.updateModelsIndexFile(modelPath, model.GetStructName())
}

func (s *ProjectScaffolds) updateModelsIndexFile(folderPath string, modelName string) error {
	filePath := path.Join(folderPath, "index.go")
	endModelsComment := "} // ends of models array. NOTE: do not remove or change this comment!"
	if utils.IsPathNotExists(filePath) {
		lines := []string{
			fmt.Sprintf("package %s", getGoSourcePackageName(filePath, "models")),
			"",
			"// AllModelStructures get all model structures",
			"func AllModelStructures() []interface{} {",
			"\tdefs := []interface{}{",
			"\t" + endModelsComment,
			"\treturn defs",
			"}",
			"",
		}

		err := writeFileContents(filePath, strings.Join(lines, "\n"), false)
		if nil != err {
			return err
		}
	}
	if "" != modelName {
		contents, err := ioutil.ReadFile(filePath)
		if nil != err {
			logger.Error.Printf("adding %s to %s failed with opening file:%v", modelName, filePath, err)
			return err
		}
		newContents := strings.Replace(string(contents), endModelsComment, fmt.Sprintf("\t&%s{},\n\t%s", modelName, endModelsComment), 1)
		return writeFileContents(filePath, newContents, true)
	}
	return nil
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
	err := writeFileContents(path.Join(folderPath, "version"), s.AppVersion, false)
	if nil != err {
		return err
	}
	configFileName := path.Base(s.ConfigFile)
	configFileGenerated := false
	if nil != s.EtcFileTemplates {
		params := s.getTemplateParams()
		for _, fname := range s.EtcFileTemplates {
			filePath := path.Join(folderPath, fname)
			if utils.IsPathExists(filePath) {
				if strings.HasSuffix(fname, ".tpl") {
					fname = fname[:len(fname)-4]
				}
				if fname == configFileName {
					configFileGenerated = true
				}
				err = s.createFileByTemplate(folderPath, fname, filePath, params)
			}
		}
	}
	if !configFileGenerated {
		templLines := []string{
			"# Logger configuration",
			"logger:",
			"  level: \"{{ .LoggerLevel | unescaped }}\"",
			"  type: \"{{ .LoggerType | unescaped }}\"",
			"  address: \"{{ .LoggerAddress | unescaped }}\"",
			"  remoteType: \"{{ .LoggerRemoteType | unescaped }}\"",
			"\n# Server configuration",
			"server:",
			"  scheme: http",
			"  host: 0.0.0.0",
			"  port: {{ .ServerPort | unescaped }}",
			"  dev: {{ .ServerDev | unescaped }}",
			"  depoyAddr: \"{{ .ServerDeployAddr | unescaped }}\"",
			"\n# MQ configuration",
			"mq:",
			"  default:",
			"    driver: {{ .MQDriver | unescaped }}",
			"    host: \"{{ .MQHost | unescaped }}\"",
			"    port: {{ .MQPort | unescaped }}",
			"    virtualHost: \"{{ .MQVHost | unescaped }}\"",
			"    username: \"{{ .MQUser | unescaped }}\"",
			"    password: \"{{ .MQPassword | unescaped }}\"",
			"    timeout: 60",
			"    heartbeat: 30",
			"\n# Cache configuration",
			"cache:",
			"  default:",
			"    driver: {{ .CacheDriver | unescaped }}",
			"    host: \"{{ .CacheHost | unescaped }}\"",
			"    port: {{ .CachePort | unescaped }}",
			"    password: \"{{ .CachePassword | unescaped }}\"",
			"    db: {{ .CacheDb | unescaped }}",
			"    clusterMode: {{ .CacheClusterMode | unescaped }}",
			"",
		}
		templContent := strings.Join(templLines, "\n")

		templParams := map[string]interface{}{
			"LoggerLevel":      `${LOGGER_LEVEL-"INFO"}`,
			"LoggerType":       `${LOGGER_TYPE-"filelog"}`,
			"LoggerAddress":    `${LOGGER_ADDR-"../log/app.log"}`,
			"LoggerRemoteType": `${LOGGER_REMOTE_TYPE-"udp"}`,
			"ServerPort":       `${SERVER_PORT-"8080"}`,
			"ServerDev":        `${SERVER_DEV-"false"}`,
			"ServerDeployAddr": `${DEPLOY_ADDR-""}`,
			"MQDriver":         `${MQ_DRIVER-"rabbitmq"}`,
			"MQHost":           `${MQ_HOST-"127.0.0.1"}`,
			"MQPort":           `${MQ_PORT-"5672"}`,
			"MQVHost":          `${MQ_VHOST-"/"}`,
			"MQUser":           `${MQ_USER-"guest"}`,
			"MQPassword":       `${MQ_PASSWORD-"guest"}`,
			"CacheDriver":      `${CACHE_DRIVER-"redis"}`,
			"CacheHost":        `${CACHE_HOST-"127.0.0.1"}`,
			"CachePort":        `${CACHE_PORT-"6379"}`,
			"CachePassword":    `${CACHE_PASSWORD}`,
			"CacheDb":          `${CACHE_DB-"15"}`,
			"CacheClusterMode": `${CACHE_CLUSTER-"false"}`,
		}

		fileExt := path.Ext(configFileName)
		err = s.createFileByTemplateContent(folderPath, strings.Replace(configFileName, fileExt, "-template"+fileExt, 1), templContent, templParams)

		configParams := map[string]interface{}{
			"LoggerLevel":      "INFO",
			"LoggerType":       "filelog",
			"LoggerAddress":    "../log/app.log",
			"LoggerRemoteType": "udp",
			"ServerPort":       "8080",
			"ServerDev":        "false",
			"ServerDeployAddr": "",
			"MQDriver":         "rabbitmq",
			"MQHost":           "127.0.0.1",
			"MQPort":           "5672",
			"MQVHost":          "/",
			"MQUser":           "guest",
			"MQPassword":       "guest",
			"CacheDriver":      "redis",
			"CacheHost":        "127.0.0.1",
			"CachePort":        "6379",
			"CachePassword":    "",
			"CacheDb":          "15",
			"CacheClusterMode": "false",
		}
		err = s.createFileByTemplateContent(folderPath, configFileName, templContent, configParams)
	}
	return nil
}

func (s *ProjectScaffolds) createGitkeepFiles(folderPath string) error {
	filePath := path.Join(folderPath, ".gitkeep")
	return writeFileContents(filePath, "", false)
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
	return writeFileContents(filePath, strings.Join(lines, "\n"), false)
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
	return writeFileContents(filePath, strings.Join(lines, "\n"), false)
}

func (s *ProjectScaffolds) createGoModFiles(folderPath string) error {
	filePath := path.Join(folderPath, "go.mod")
	ver := getGolangVersion()
	s.goVersion = ver
	lines := []string{
		fmt.Sprintf("module \"%s\"\n", s.getGoModName()),
		fmt.Sprintf("go %s\n", ver),
		"require (",
		"\tgithub.com/kevinyjn/gocom master",
		")\n",
	}
	return writeFileContents(filePath, strings.Join(lines, "\n"), false)
}

func (s *ProjectScaffolds) getGoModName() string {
	return fmt.Sprintf("project.local/%s", utils.KebabCaseString(s.Name))
}

func (s *ProjectScaffolds) createEntrypointSources(folderPath string) error {
	filePath := path.Join(folderPath, "init.go")
	err := writeFileContents(filePath, s.getGoInitSource(), false)
	if nil != err {
		return err
	}
	filePath = path.Join(folderPath, "main.go")
	return writeFileContents(filePath, s.getGoMainSource(), false)
}

func (s *ProjectScaffolds) getGoInitSource() string {
	lines := []string{
		"package main\n",
		"import (",
		"\t\"flag\"",
		"\t\"fmt\"",
		")\n",
		"// Variables of program information",
		"var (",
		fmt.Sprintf("\tBuildVersion = \"%s\"", s.AppVersion),
		fmt.Sprintf("\tBuildName    = \"%s\"", s.Name),
		fmt.Sprintf("\tBuiltTime    = \"%s\"", utils.TimeToHuman("YYYY-MM-DD HH:mm:ss", time.Now())),
		"\tCommitID     = \"\"",
		fmt.Sprintf("\tConfigFile   = \"%s\"", s.ConfigFile),
		fmt.Sprintf("\tMQConfigFile = \"%s\"", s.MQConfigFile),
		")\n",
		"func init() {",
		"\tvar showVer bool\n\tvar confFile string\n",
		"\tflag.BoolVar(&showVer, \"v\", false, \"Build version\")",
		fmt.Sprintf("\tflag.StringVar(&confFile, \"config\", \"%s\", \"Configure file\")", s.ConfigFile),
		"\tflag.Parse()\n",
		"\tif confFile != \"\" {\n\t\tConfigFile = confFile\n\t}",
		"\tif showVer {",
		"\t\tfmt.Printf(\"Build name:\\t%s\\n\", BuildName)",
		"\t\tfmt.Printf(\"Build version:\\t%s\\n\", BuildVersion)",
		"\t\tfmt.Printf(\"Built time:\\t%s\\n\", BuiltTime)",
		"\t\tfmt.Printf(\"Commit ID:\\t%s\\n\", CommitID)",
		"\t}",
		"}\n",
	}
	return strings.Join(lines, "\n")
}

func (s *ProjectScaffolds) getGoMainSource() string {
	lines := []string{
		"package main",
		"",
		"import (",
		"\t\"fmt\"\n",
		fmt.Sprintf("\t\"%s/controllers\"", s.getGoModName()),
		"",
		"\t\"github.com/kevinyjn/gocom/application\"",
		"\t\"github.com/kevinyjn/gocom/caching\"",
		"\t\"github.com/kevinyjn/gocom/config\"",
		"\t\"github.com/kevinyjn/gocom/logger\"",
		"\t\"github.com/kevinyjn/gocom/mq\"",
		"\t\"github.com/kevinyjn/gocom/orm/rdbms\"",
		"\t\"github.com/kevinyjn/gocom/utils\"\n",
		"\t\"github.com/kataras/iris\"",
		")\n",
		"func main() {",
		"\tenv, err := config.Init(ConfigFile)",
		"\tif nil != err {\n\t\treturn\n\t}\n",
		"\tlogger.Init(&env.Logger)\n",
		"\tif len(env.MQs) > 0 {",
		"\t\terr = mq.Init(MQConfigFile, env.MQs)",
		"\t\tif nil != err {",
		"\t\t\tlogger.Error.Printf(\"Please check the mq configuration and restart. error:%v\", err)",
		"\t\t}\n\t}",
		"\tif len(env.Caches) > 0 {",
		"\t\tif !caching.InitCacheProxy(env.Caches) {",
		"\t\t\tlogger.Warning.Println(\"Please check the caching configuration and restart.\")",
		"\t\t}\n\t}",
		"\tif len(env.DBs) > 0 {",
		"\t\tfor category, dbConfig := range env.DBs {",
		"\t\t\t_, err = rdbms.GetInstance().Init(category, &dbConfig)",
		"\t\t\tif nil != err {",
		"\t\t\t\tlogger.Error.Printf(\"Please check the database:%s configuration and restart. error:%v\", category, err)",
		"\t\t\t}\n\t\t}\n\t}",
		"\n\tlistenAddr := fmt.Sprintf(\"%s:%d\", env.Server.Host, env.Server.Port)",
		"\tlogger.Info.Printf(\"Starting server on %s...\", listenAddr)\n",
		"\tapp := application.GetApplication(config.ServerMain)",
		"\taccessLogger, accessLoggerClose := logger.NewAccessLogger()",
		"\tdefer accessLoggerClose()",
		"\tapp.Use(accessLogger)",
		"\tcontrollers.InitAll(app)",
		"\tapp.RegisterView(iris.HTML(config.TemplateViewPath, config.TemplateViewEndfix))",
		"\tif utils.IsPathExists(config.Favicon) {\n\t\tapp.Favicon(config.Favicon)\n\t}",
		"\tapp.StaticWeb(config.StaticRoute, config.StaticAssets)",
		"\tapp.Run(iris.Addr(listenAddr), iris.WithoutServerError(iris.ErrServerClosed))",
		"}",
		"",
	}
	return strings.Join(lines, "\n")
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

func (s *ProjectScaffolds) createFileByTemplateContent(folderPath string, fileName string, templContent string, params map[string]interface{}) error {
	filePath := path.Join(folderPath, fileName)
	tpl := template.New("index")
	tpl = tpl.Funcs(template.FuncMap{"unescaped": func(x string) interface{} { return template.HTML(x) }})
	tpl, err := tpl.Parse(templContent)
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

func (s *ProjectScaffolds) createControllerIndexSourcefiles(folderPath string) error {
	filePath := path.Join(folderPath, "index.go")
	packageName := getGoSourcePackageName(filePath, "controllers")
	lines := []string{
		fmt.Sprintf("package %s\n", packageName),
		"import (",
		"\t\"github.com/kataras/iris\"\n",
		"\t",
		fmt.Sprintf("\t\"%s/controllers/api\"", s.getGoModName()),
		")\n",
		"// InitAll initialize all handlers",
		"func InitAll(app *iris.Application) {",
		"\tapi.Init(app)",
		"}\n",
	}
	writeFileContents(filePath, strings.Join(lines, "\n"), false)

	apiFolderPath := path.Join(folderPath, "api")
	err := utils.EnsureDirectory(apiFolderPath)
	if nil != err {
		return err
	}
	return s.createControllerAPISourcefiles(apiFolderPath)
}

func (s *ProjectScaffolds) createControllerAPISourcefiles(folderPath string) error {
	filePath := path.Join(folderPath, "init.go")
	packageName := getGoSourcePackageName(filePath, "controllers")
	lines := []string{
		fmt.Sprintf("package %s\n", packageName),
		"import (",
		"\t\"github.com/kataras/iris\"",
		"\t\"github.com/kevinyjn/gocom/orm/rdbms\"\n",
		fmt.Sprintf("\t\"%s/models\"", s.getGoModName()),
		")\n",
		"// Init initialize api handlers",
		"func Init(app *iris.Application) {",
		"\tapiApp := app.Party(GraphQLRoute, beforeAPIAuthMiddlewareHandler)",
		"\tbeans := models.AllModelStructures()",
		"\trdbms.RegisterGraphQLRoutes(apiApp, beans)",
		"\trdbms.RegisterGraphQLMQs(APIConsumerMQCategory, APIProduceMQCategory, beans)",
		"}\n",
	}
	err := writeFileContents(filePath, strings.Join(lines, "\n"), false)

	filePath = path.Join(folderPath, "accesscontrol.go")
	lines = []string{
		fmt.Sprintf("package %s\n", packageName),
		"import (",
		"\t\"github.com/kataras/iris\"",
		")\n",
		"// beforeAPIAuthMiddlewareHandler access control middleware",
		"func beforeAPIAuthMiddlewareHandler(ctx iris.Context) {",
		"\tctx.Next()",
		"}\n",
	}
	err = writeFileContents(filePath, strings.Join(lines, "\n"), false)

	filePath = path.Join(folderPath, "consts.go")
	lines = []string{
		fmt.Sprintf("package %s\n", packageName),
		"// Constants",
		"const (",
		"\tGraphQLRoute          = \"/api/graphql\"",
		"\tAPIConsumerMQCategory = \"api-consumer\"",
		"\tAPIProduceMQCategory  = \"api-producer\"",
		")\n",
	}
	err = writeFileContents(filePath, strings.Join(lines, "\n"), false)

	filePath = path.Join(folderPath, "handlers.go")
	lines = []string{
		fmt.Sprintf("package %s\n", packageName),
	}
	err = writeFileContents(filePath, strings.Join(lines, "\n"), false)

	return err
}

func getGolangVersion() string {
	ver := "1.14"
	r := regexp.MustCompile(`(\d+\.\d+)`)
	found := r.FindAllStringSubmatch(runtime.Version(), 1)
	if len(found) > 0 {
		ver = found[0][0]
	}
	return ver
}

func writeFileContents(filePath, contents string, forceOverride bool) error {
	if false == forceOverride && utils.IsPathExists(filePath) {
		return nil
	}
	fl, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if nil != err {
		return err
	}
	_, err = fl.WriteString(contents)
	fl.Close()
	return err
}

func getGoSourcePackageName(filePath string, defaultName string) string {
	pkgName := path.Dir(filePath)
	if "" == pkgName || "." == pkgName || "./" == pkgName {
		pkgName = defaultName
	} else {
		spIdx := strings.LastIndexAny(pkgName, "./\\ ")
		if spIdx >= 0 {
			pkgName = pkgName[spIdx+1:]
		}
	}
	return pkgName
}
