package rdbms

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"

	"github.com/kataras/iris"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/router"
)

// BeanGraphQLSchemaInfo graphql info
type BeanGraphQLSchemaInfo struct {
	beanType      reflect.Type
	schemaType    *graphql.Object
	schema        *graphql.Schema
	fieldsMapping map[string]graphFieldInfo
	structureName string
}

type graphFieldInfo struct {
	Name string
	Type *graphql.Scalar
}

// GraphQLDelegate GraphQL operator wrapper
type GraphQLDelegate struct {
	prototypes map[string]*BeanGraphQLSchemaInfo
	routeNames map[string]string
}

var _graphqlDelegate = &GraphQLDelegate{
	prototypes: map[string]*BeanGraphQLSchemaInfo{},
	routeNames: map[string]string{},
}

// GraphQL delegate
func GraphQL() *GraphQLDelegate {
	return _graphqlDelegate
}

// GetGraphQLSchemaInfo of xorm bean
func (g *GraphQLDelegate) GetGraphQLSchemaInfo(bean interface{}) *BeanGraphQLSchemaInfo {
	val := getBeanValue(bean)
	structureName := ""
	if val.Type().Kind() == reflect.Struct {
		structureName = fmt.Sprintf("%s.%s", val.Type().PkgPath(), val.Type().Name())
	} else {
		return nil
	}
	schemaInfo := g.prototypes[structureName]
	if nil == schemaInfo {
		fieldsMapping := map[string]graphFieldInfo{}
		fields := enumerateGraphQLFields(val, fieldsMapping)
		comments := ""
		dsField, has := val.Type().FieldByName("Datasource")
		if has {
			comments = dsField.Tag.Get("comments")
		}

		schemaInfo = &BeanGraphQLSchemaInfo{
			beanType: val.Type(),
			schemaType: graphql.NewObject(
				graphql.ObjectConfig{
					Name:        val.Type().Name(),
					Fields:      fields,
					Description: comments,
				},
			),
			fieldsMapping: fieldsMapping,
			structureName: structureName,
		}

		schemaInfo.schema = NewGraphQLSchema(val, schemaInfo)
		g.prototypes[structureName] = schemaInfo
		g.routeNames[utils.KebabCaseString(schemaInfo.Name())] = structureName
	}
	return schemaInfo
}

func enumerateGraphQLFields(val reflect.Value, fieldsMapping map[string]graphFieldInfo) graphql.Fields {
	fields := graphql.Fields{}
	numFields := val.Type().NumField()
	for i := 0; i < numFields; i++ {
		f := val.Field(i)
		ft := val.Type().Field(i)
		jsonTag := ft.Tag.Get("json")
		ormTag := ft.Tag.Get("xorm")
		if "-" == jsonTag || "-" == ormTag {
			continue
		} else if "" == jsonTag {
			jsonTag = utils.CamelCaseString(ft.Name)
		}

		var graphType *graphql.Scalar = nil
		if strings.Contains(ormTag, " pk ") || strings.HasSuffix(ormTag, " pk") {
			graphType = graphql.ID
		} else {
			switch f.Type().Kind() {
			case reflect.String:
				graphType = graphql.String
				break
			case reflect.Float32, reflect.Float64:
				graphType = graphql.Float
				break
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				graphType = graphql.Int
				break
			case reflect.Struct:
				if "time.Time" == f.Type().String() {
					graphType = graphql.DateTime
				} else if "extends" == ormTag {
					extendFields := enumerateGraphQLFields(f, fieldsMapping)
					for k, v := range extendFields {
						fields[k] = v
					}
				} else {
					// TODO embeded structs fields
				}
				break
			case reflect.Bool:
				graphType = graphql.Boolean
				break
			default:
				break
			}
		}

		if nil != graphType {
			fieldsMapping[jsonTag] = graphFieldInfo{Name: ft.Name, Type: graphType}
			fields[jsonTag] = &graphql.Field{
				Name:        jsonTag,
				Type:        graphType,
				Description: ft.Tag.Get("comments"),
			}
		}
	}
	return fields
}

// NewGraphQLQuerySchema new query type
func NewGraphQLQuerySchema(beanValue reflect.Value, beanGraphQLSchemaInfo *BeanGraphQLSchemaInfo) *graphql.Object {
	queryFields := graphql.Fields{}
	beanGraphQLSchemaType := beanGraphQLSchemaInfo.schemaType
	/* Get (read) single record by id
	   http://domain/graphql/schemaName?query={schenaName(id:1){schemaFields...}}
	*/
	queryFields[utils.CamelCaseString(beanGraphQLSchemaType.Name())] = &graphql.Field{
		Type:        beanGraphQLSchemaType,
		Description: fmt.Sprintf("Get %s by id", beanGraphQLSchemaType.Name()),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.ID,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			condiBean, ok := beanGraphQLSchemaInfo.GenerateConditionBean(p.Args)
			if ok {
				has, err := GetInstance().FetchOne(condiBean)
				if has {
					return condiBean, nil
				}
				return nil, err
			}
			return nil, nil
		},
	}
	/* Get (read) list of record by condition
	   http://domain/graphql/schemaName?query={list(conditions...){schemaFields...}}
	*/
	queryFields["list"] = &graphql.Field{
		Type:        graphql.NewList(beanGraphQLSchemaType),
		Description: fmt.Sprintf("Get %s list", beanGraphQLSchemaType.Name()),
		Args:        beanGraphQLSchemaInfo.GenerateGraphQLArgs(),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			condiBean, ok := beanGraphQLSchemaInfo.GenerateConditionBean(p.Args)
			if ok {
				records, err := GetInstance().FetchAll(condiBean)
				if nil == err {
					return records, nil
				}
				return nil, err
			}
			return nil, nil
		},
	}
	query := graphql.NewObject(
		graphql.ObjectConfig{
			Name:   "Query",
			Fields: queryFields,
		},
	)
	return query
}

// NewGraphQLMutationSchema new mutation type
func NewGraphQLMutationSchema(beanValue reflect.Value, beanGraphQLSchemaInfo *BeanGraphQLSchemaInfo) *graphql.Object {
	mutationFields := graphql.Fields{}
	beanGraphQLSchemaType := beanGraphQLSchemaInfo.schemaType
	/* Create new record item
	   http://domain/graphql/schemaName?query=mutation+_{create(name:"Inca Kola",info:"Inca Kola is a soft drink that was created in Peru in 1935 by British immigrant Joseph Robinson Lindley using lemon verbena (wiki)",price:1.99){id,name,info,price}}
	*/
	mutationFields["create"] = &graphql.Field{
		Type:        beanGraphQLSchemaType,
		Description: fmt.Sprintf("Create new %s", beanGraphQLSchemaType.Name()),
		Args:        beanGraphQLSchemaInfo.GenerateGraphQLArgs(),
		Resolve: func(params graphql.ResolveParams) (interface{}, error) {
			condiBean, ok := beanGraphQLSchemaInfo.GenerateConditionBean(params.Args)
			if ok {
				succ, err := GetInstance().SaveOne(condiBean)
				if nil != err {
					logger.Error.Printf("Create %s with data:%v failed with error:%v", beanGraphQLSchemaType.Name(), params.Args, err)
					return nil, err
				} else if succ {
					logger.Info.Printf("Create %s with data:%v succeed", beanGraphQLSchemaType.Name(), params.Args)
				}
				return condiBean, err
			}
			return nil, nil
		},
	}
	/* Update record by id
	   http://domain/graphql/schemaName?query=mutation+_{update(id:1,price:3.95){id,name,info,price}}
	*/
	mutationFields["update"] = &graphql.Field{
		Type:        beanGraphQLSchemaType,
		Description: fmt.Sprintf("Update %s by primary key", beanGraphQLSchemaType.Name()),
		Args:        beanGraphQLSchemaInfo.GenerateGraphQLArgs(),
		Resolve: func(params graphql.ResolveParams) (interface{}, error) {
			condiBean, ok := beanGraphQLSchemaInfo.GenerateConditionBean(params.Args)
			if ok {
				succ, err := GetInstance().SaveOne(condiBean)
				if nil != err {
					logger.Error.Printf("Update %s with data:%v failed with error:%v", beanGraphQLSchemaType.Name(), params.Args, err)
					return nil, err
				} else if succ {
					logger.Info.Printf("Update %s with data:%v succeed", beanGraphQLSchemaType.Name(), params.Args)
				} else {
					logger.Info.Printf("Update %s with data:%v while nothing changed", beanGraphQLSchemaType.Name(), params.Args)
				}
				return condiBean, err
			}
			return nil, nil
		},
	}
	/* Delete record by id
	   http://domain/graphql/schemaName?query=mutation+_{delete(id:1){id,name,info,price}}
	*/
	mutationFields["delete"] = &graphql.Field{
		Type:        beanGraphQLSchemaType,
		Description: fmt.Sprintf("Update %s by primary key", beanGraphQLSchemaType.Name()),
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: graphql.ID,
			},
		},
		Resolve: func(params graphql.ResolveParams) (interface{}, error) {
			condiBean, ok := beanGraphQLSchemaInfo.GenerateConditionBean(params.Args)
			if ok {
				has, err := GetInstance().FetchOne(condiBean)
				if has {
					rc, err := GetInstance().Delete(condiBean)
					if nil != err {
						logger.Error.Printf("Delete %s with data:%v failed with error:%v", beanGraphQLSchemaType.Name(), params.Args, err)
						return nil, err
					} else if rc > 0 {
						logger.Info.Printf("Delete %s with data:%v succeed", beanGraphQLSchemaType.Name(), params.Args)
					}
					return condiBean, err
				}
				return nil, err
			}
			return nil, nil
		},
	}
	mutation := graphql.NewObject(
		graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: mutationFields,
		},
	)
	return mutation
}

// NewGraphQLSchema new schema
func NewGraphQLSchema(beanValue reflect.Value, beanGraphQLSchemaInfo *BeanGraphQLSchemaInfo) *graphql.Schema {
	querySchema := NewGraphQLQuerySchema(beanValue, beanGraphQLSchemaInfo)
	mutationSchema := NewGraphQLMutationSchema(beanValue, beanGraphQLSchemaInfo)
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    querySchema,
		Mutation: mutationSchema,
	})
	if nil != err {
		logger.Error.Printf("Creating graphql schema %s failed with error:%v", beanGraphQLSchemaInfo.schemaType.Name(), err)
	}
	return &schema
}

// ExecuteGraphQLQuery execute GraphQL query
func ExecuteGraphQLQuery(query string, schema *graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        *schema,
		RequestString: query,
	})
	if result.HasErrors() {
		logger.Error.Printf("Execute graphql %s query:%s failed with error:%v", schema.QueryType().Name(), query, result.Errors)
	}
	return result
}

// GetSchema get schema
func (i *BeanGraphQLSchemaInfo) GetSchema() *graphql.Schema {
	return i.schema
}

// Name name of schema
func (i *BeanGraphQLSchemaInfo) Name() string {
	return i.schemaType.Name()
}

// GenerateConditionBean generate contidion bean
func (i *BeanGraphQLSchemaInfo) GenerateConditionBean(args map[string]interface{}) (interface{}, bool) {
	var condiBeanValue = reflect.New(i.beanType)
	var setFields = 0
	if nil == args {
		return condiBeanValue.Interface(), false
	}

	for k, v := range args {
		fieldInfo, ok := i.fieldsMapping[k]
		if ok {
			fieldSlices := strings.Split(fieldInfo.Name, ".")
			fv := condiBeanValue.Elem()
			for _, name := range fieldSlices {
				fv = fv.FieldByName(name)
				if fv.Type().Kind() == reflect.Ptr {
					fv = fv.Elem()
				}
				if fv.Type().Kind() != reflect.Struct {
					break
				}
			}

			setFields++
			argValue := reflect.ValueOf(v)
			// converts the graphql ID type(int or string)
			switch argValue.Type().Kind() {
			case reflect.Int:
				if fv.Type().Kind() == reflect.String {
					fv.SetString(strconv.Itoa(v.(int)))
				} else {
					fv.Set(argValue)
				}
				break
			case reflect.String:
				if fv.Type().Kind() == reflect.Int {
					iv, _ := strconv.ParseInt(v.(string), 10, 64)
					fv.SetInt(iv)
				} else {
					fv.Set(argValue)
				}
				break
			default:
				fv.Set(argValue)
				break
			}
		}
	}
	return condiBeanValue.Interface(), setFields > 0
}

// GenerateGraphQLArgs generate arguments for graphql
func (i *BeanGraphQLSchemaInfo) GenerateGraphQLArgs() graphql.FieldConfigArgument {
	args := graphql.FieldConfigArgument{}
	for k, fi := range i.fieldsMapping {
		args[k] = &graphql.ArgumentConfig{
			Type: fi.Type,
		}
	}
	return args
}

// HandlerQuery handle GraphQL query
func (i *BeanGraphQLSchemaInfo) HandlerQuery(ctx iris.Context) {
	var query string
	switch ctx.Method() {
	case "GET":
		query = ctx.URLParam("query")
		if strings.HasPrefix(query, "mutation") {
			logger.Warning.Printf("GraphQL %s handler mutation operation were not allowed by get method", i.schemaType.Name())
			if !logger.IsDebugEnabled() {
				return
			}
		}
		break
	case "POST", "PUT", "PATCH":
		query = ctx.FormValue("query")
	default:
		ctx.StatusCode(iris.StatusMethodNotAllowed)
		ctx.WriteString("Method not allowed")
		return
		// break
	}
	result := ExecuteGraphQLQuery(query, i.schema)
	responseBody, err := json.Marshal(result)
	if nil != err {
		logger.Error.Printf("Serialize %s query:%s response failed with error:%v", i.schemaType.Name(), query, err)
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString(err.Error())
	} else {
		ctx.WriteString(string(responseBody))
	}
}

// RegisterGraphQLRoutes register models as graphql api
func RegisterGraphQLRoutes(app router.Party, beans []interface{}) error {
	var errMessages = []string{}
	for _, bean := range beans {
		schemaInfo := GraphQL().GetGraphQLSchemaInfo(bean)
		if nil == schemaInfo {
			logger.Error.Printf("Get graphql schema info by bean:%v failed", bean)
			errMessages = append(errMessages, fmt.Sprintf("%v were invalid for registering GraphQL", bean))
			continue
		}
		routeName := utils.KebabCaseString(schemaInfo.Name())
		logger.Info.Printf("Registering graphql handler for %s %s/%s", schemaInfo.Name(), app.GetRelPath(), routeName)
		app.Get("/"+routeName, schemaInfo.HandlerQuery)
		app.Post("/"+routeName, schemaInfo.HandlerQuery)
	}
	if len(errMessages) > 0 {
		return errors.New(strings.Join(errMessages, ";"))
	}
	return nil
}

func handlerMQGraphQLMessage(m mqenv.MQConsumerMessage) *mqenv.MQPublishMessage {
	routeName := m.RoutingKey
	lastIdx := strings.LastIndex(routeName, ".")
	if lastIdx >= 0 {
		routeName = routeName[lastIdx+1:]
	}
	schemaKey := GraphQL().routeNames[routeName]
	var responseContent []byte
	for {
		if "" == schemaKey {
			logger.Warning.Printf("Could not get GraphQL schema by routing name:%s", routeName)
			responseContent = []byte(fmt.Sprintf("Could not get GraphQL schema by routing name:%s", routeName))
			break
		}
		schemaInfo := GraphQL().prototypes[schemaKey]
		if nil == schemaInfo {
			logger.Warning.Printf("Could not get GraphQL schema by routing name:%s", routeName)
			responseContent = []byte(fmt.Sprintf("Could not get GraphQL schema by routing name:%s", routeName))
			break
		}
		query := string(m.Body)
		result := ExecuteGraphQLQuery(query, schemaInfo.schema)
		responseBody, err := json.Marshal(result)
		if nil != err {
			logger.Error.Printf("Serializing GraphQL executes query result failed with error:%v", err)
			responseContent = []byte(err.Error())
			break
		}
		responseContent = responseBody
		break
	}

	// response
	mqCategory := m.ConsumerTag
	publishMessage := mqenv.MQPublishMessage{
		Body:          responseContent,
		RoutingKey:    m.ReplyTo,
		CorrelationID: m.CorrelationID,
		EventLabel:    "",
		Headers:       map[string]string{},
	}
	mq.PublishMQ(mqCategory, &publishMessage)
	return nil
}

// RegisterGraphQLMQs register models as graphql mq api
func RegisterGraphQLMQs(mqConfigCategory string, responseMQConfigCategory string, beans []interface{}) error {
	var errMessages = []string{}
	mqConfig := mq.GetMQConfig(mqConfigCategory)
	if nil == mqConfig {
		logger.Error.Printf("Registering GraphQL MQ handlers while could not get MQ config by category:%s", mqConfigCategory)
		return fmt.Errorf("Could not get MQ config by category:%s", mqConfigCategory)
	}
	consumeProxy := &mqenv.MQConsumerProxy{
		Queue:       mqConfig.Queue,
		Callback:    handlerMQGraphQLMessage,
		ConsumerTag: responseMQConfigCategory,
		AutoAck:     true,
	}
	err := mq.ConsumeMQ(mqConfigCategory, consumeProxy)
	if nil != err {
		logger.Error.Printf("Registering GraphQL MQ handlers by category:%s failed with error:%v", mqConfigCategory, err)
		return err
	}
	for _, bean := range beans {
		schemaInfo := GraphQL().GetGraphQLSchemaInfo(bean)
		if nil == schemaInfo {
			logger.Error.Printf("Get graphql schema info by bean:%v failed for registering GraphQL MQ", bean)
			errMessages = append(errMessages, fmt.Sprintf("%v were invalid for registering GraphQL MQ", bean))
			continue
		}
		routeName := utils.KebabCaseString(schemaInfo.Name())
		logger.Info.Printf("Registering graphql MQ handler for %s %s %s", schemaInfo.Name(), mqConfigCategory, routeName)
		GraphQL().routeNames[routeName] = schemaInfo.structureName
	}
	if len(errMessages) > 0 {
		return errors.New(strings.Join(errMessages, ";"))
	}
	return nil
}
