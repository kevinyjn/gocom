package mongodb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Constant
const (
	// MongoDBPingInterval mongodb heatbeat ping interval
	MongoDBPingInterval = 5

	MongoDBDriverName = "mongodb"
)

// MongoSession mongodb session
type MongoSession struct {
	Client       *mongo.Client
	Database     string
	LastError    error
	Name         string
	connHost     string
	lastPingTime int64
}

// MongoMeta mongodb models meta structure
type MongoMeta struct {
	meta string
}

// ErrorMongoDB mongodb error
type ErrorMongoDB string

// Error error message for mongodb error
func (s ErrorMongoDB) Error() string {
	return string(s)
}

// MongoBeforeSaveFunc mongodb before save event callback
type MongoBeforeSaveFunc func(v interface{})

var mongosets = make(map[string]*MongoSession)
var mongoBeforeSaveObservers = make([]MongoBeforeSaveFunc, 0)
var mongoDefaultBeforeSaveObserversShouldRegister = true

var cachingObjectsByCondition = make(map[string]interface{})
var cachingObjectsByID = make(map[string]interface{})

// InitMongoDB initialzie mongodb sessions
func InitMongoDB(dbConfigs map[string]definations.DBConnectorConfig) error {
	if mongoDefaultBeforeSaveObserversShouldRegister {
		mongoDefaultBeforeSaveObserversShouldRegister = false
		AddMongoBeforeSaveObserver(mongoDefaultBeforeSaveObserver)
	}

	for cnfname, cnf := range dbConfigs {
		if cnf.Driver != MongoDBDriverName {
			continue
		}
		one := &MongoSession{
			Name:     cnfname,
			Database: cnf.Db,
		}
		addrs := strings.Split(cnf.Address, "@")
		if len(addrs) < 2 {
			one.connHost = addrs[0]
		} else {
			one.connHost = addrs[1]
		}
		cliOptions := options.Client().ApplyURI(cnf.Address)
		if cnf.Mechanism != "" {
			cliOptions.Auth.AuthMechanism = cnf.Mechanism
		}
		client, err := mongo.NewClient(cliOptions)
		if err != nil {
			logger.Error.Printf("Create mongodb client %s %s failed with error:%v", cnfname, cnf.Address, err)
			one.LastError = err
			return err
		}
		one.Client = client
		err = one.connect()
		if err != nil {
			return err
		}

		logger.Info.Printf("Connected to mongodb %s on with user:%s address:%s\n", cnfname, cliOptions.Auth.Username, one.connHost)

		one.LastError = nil

		mongosets[cnfname] = one

		go one.StartKeepAlive()
	}

	return nil
}

// GetMongoDB get mongodb
func GetMongoDB(name string) *MongoSession {
	return mongosets[name]
}

// AddMongoBeforeSaveObserver observer
func AddMongoBeforeSaveObserver(f MongoBeforeSaveFunc) {
	mongoBeforeSaveObservers = append(mongoBeforeSaveObservers, f)
}

// StartKeepAlive keepalive
func (s *MongoSession) StartKeepAlive() {
	ticker := time.NewTicker(time.Second * MongoDBPingInterval)
	for {
		select {
		case ts := <-ticker.C:
			curTs := ts.Unix()
			if curTs-s.lastPingTime > 600 {
				logger.Trace.Printf("ping %s %s %s ...", s.Name, s.Database, s.connHost)
			}
			s.lastPingTime = curTs
			err := s.Ping()
			if err != nil {
				logger.Error.Printf("Mongodb:%s %s ping failed with error:%v, reconnecting...", s.Name, s.Database, err)
				s.LastError = err
				s.connect()
			}
		}
	}
}

// Ping pinger
func (s *MongoSession) Ping() error {
	return s.Client.Ping(nil, nil)
}

func (s *MongoSession) connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s.LastError = s.Client.Connect(ctx)
	if s.LastError != nil {
		logger.Error.Printf("Connect to mongodb %s %s failed with error:%s", s.Name, s.connHost, s.LastError.Error())
	}
	return s.LastError
}

// IsMongoDBReady is ready
func (s *MongoSession) IsMongoDBReady() bool {
	return s.LastError == nil
}

// GetCollection get mongodb collection
func (s *MongoSession) GetCollection(collectionName string) *mongo.Collection {
	if s.LastError != nil {
		logger.Error.Printf("Get mongodb collection:%s while the mongodb connection was failed with error:%s", collectionName, s.LastError.Error())
		return nil
	}
	col := s.Client.Database(s.Database).Collection(collectionName)
	if col == nil {
		logger.Error.Printf("Get mongodb collection:%s failed", collectionName)
	}
	return col
}

// FindAll finder
func (s *MongoSession) FindAll(collectionName string, filter bson.D) ([]bson.M, error) {
	results := []bson.M{}
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return results, fmt.Errorf("Could not get db collection by collection name:%s", collectionName)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		logger.Error.Printf("Mongodb FindAll collection:%s failed with error:%v", collectionName, err)
		return results, err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var row bson.M
		err := cur.Decode(&row)
		if err != nil {
			logger.Error.Printf("Mongodb FindAll collection:%s decode row failed with error:%v", collectionName, err)
		}
		// do something with result....
		results = append(results, row)
	}
	if err := cur.Err(); err != nil {
		logger.Error.Printf("Mongodb FindAll collection:%s read records failed with error:%v", collectionName, err)
		return results, err
	}
	return results, nil
}

// QueryFind query
func (s *MongoSession) QueryFind(model interface{}, filter interface{}, limit int64, results interface{}) (int, error) {
	num := 0
	resultsValue := reflect.ValueOf(results)
	if resultsValue.Type().Kind() != reflect.Ptr {
		return num, errors.New("The results parameter sould passing pointer of array like &[]interface{}")
	}
	collectionName := GetCollectionName(model)
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return num, fmt.Errorf("Could not get db collection by collection name:%s", collectionName)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cur, err := collection.Find(ctx, filter, &options.FindOptions{Limit: &limit})
	if err != nil {
		logger.Error.Printf("Mongodb query find on collection:%s failed with error:%v", collectionName, err)
		return num, err
	}
	if cur == nil {
		logger.Error.Printf("Mongodb query find on collection:%s while cursor is nil", collectionName)
		return num, ErrorMongoDB("Mongodb query find while cursor is nil")
	}
	resultsValue = resultsValue.Elem()
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		row := newMongoModel(model)
		err := cur.Decode(row)
		if err != nil {
			logger.Error.Printf("MongoDB QueryFind collection:%s decode row data failed with error:%v", collectionName, err)
			continue
		}
		num++
		// do something with result....
		// row.AfterLoaded()
		resultsValue = reflect.Append(resultsValue, reflect.ValueOf(row))
		// rows = append(rows, row)
	}
	if err := cur.Err(); err != nil {
		logger.Error.Printf("MongoDB QueryFind collection:%s read records failed with error:%v", collectionName, err)
	}
	reflect.ValueOf(results).Elem().Set(resultsValue)
	// results = resultsValue.Interface()
	return num, nil
}

// GetCollectionName get collection name by model
func GetCollectionName(dst interface{}) string {
	value := reflect.ValueOf(dst)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	} else if value.Kind() == reflect.String {
		return dst.(string)
	}
	valueType := value.Type()
	collectionName := valueType.Name()
	metaField, exists := valueType.FieldByName("meta")
	if exists {
		collectionName = metaField.Tag.Get("collection")
		if collectionName == "" {
			collectionName = valueType.Name()
		} else {
			slices := strings.Split(collectionName, ",")
			collectionName = slices[0]
		}
	}
	return collectionName
}

// GetMetaField get meta field by model
func GetMetaField(model interface{}) (reflect.StructField, bool) {
	value := reflect.ValueOf(model)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	} else if value.Kind() == reflect.String {
		return reflect.StructField{}, false
	}
	return value.Type().FieldByName("meta")
}

func newMongoModel(v interface{}) interface{} {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	} else if value.Kind() == reflect.String {
		return &bson.M{}
	}

	result := reflect.New(value.Type()).Interface()
	return result
}

// FindOne finder
func (s *MongoSession) FindOne(dst interface{}, filter interface{}, opts ...*options.FindOneOptions) error {
	collectionName := GetCollectionName(dst)
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return ErrorMongoDB(fmt.Sprintf("Could not get collection object by collection name:%s", collectionName))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := collection.FindOne(ctx, filter, opts...).Decode(dst)
	if err != nil {
		// logger.Error.Printf("Query mongodb collection:%s failed with error:%v", collectionName, err)
		return err
	}
	return nil
}

// SaveOne save
func (s *MongoSession) SaveOne(dst interface{}, isInsert bool) error {
	collectionName := GetCollectionName(dst)
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return ErrorMongoDB(fmt.Sprintf("Could not get collection object by collection name:%s", collectionName))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dispatchMongoBeforeSaveEvent(dst)

	if isInsert {
		value := reflect.ValueOf(dst).Elem()
		fieldMKey := value.FieldByName("ID")
		// if fieldMKey.IsValid() && fieldMKey.Type() == reflect.TypeOf(primitive.NilObjectID) {
		// 	if fieldMKey.Interface().(primitive.ObjectID) == primitive.NilObjectID {
		// 		fieldMKey.Set(reflect.ValueOf(primitive.NewObjectID()))
		// 	}
		// }

		res, err := collection.InsertOne(ctx, dst)
		if err != nil {
			logger.Error.Printf("Save mongodb document by collection:%s failed with error:%v", collectionName, err)
			return err
		}
		if fieldMKey.IsValid() && (fieldMKey.Type() == reflect.TypeOf(res.InsertedID)) {
			fieldMKey.Set(reflect.ValueOf(res.InsertedID))
		}
	} else {
		value := reflect.ValueOf(dst).Elem()
		field := value.FieldByName("ID")
		key := field.Interface()
		updates := mongoOrganizeUpdatesFields(dst)
		filter := bson.M{"_id": key}
		_, err := collection.UpdateOne(ctx, filter, updates)
		if err != nil {
			logger.Error.Printf("Save mongodb document by collection:%s failed with error:%v", collectionName, err)
			return err
		}
		return nil
	}
	return nil
}

// DeleteOne delete
func (s *MongoSession) DeleteOne(dst interface{}, isInsert bool) error {
	collectionName := GetCollectionName(dst)
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return ErrorMongoDB(fmt.Sprintf("Could not get collection object by collection name:%s", collectionName))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	value := reflect.ValueOf(dst).Elem()
	field := value.FieldByName("ID")
	key := field.Interface()
	filter := bson.M{"_id": key}
	_, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		logger.Error.Printf("Delete mongodb document by collection:%s failed with error:%v", collectionName, err)
		return err
	}
	return nil
}

// MongoDBInsertMany insert many
func (s *MongoSession) MongoDBInsertMany(collectionName string, docs []interface{}) bool {
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		logger.Fatal.Printf("Insert objects failed with error:%v", err)
		return false
	}

	return true
}

type singleGrouppingAggregateResult struct {
	ID    string      `bson:"_id"`
	Value interface{} `bson:"value"`
}

func (e *singleGrouppingAggregateResult) formatSingleGrouppingAggregatePipeline(aggName string, aggField string) []bson.M {
	return []bson.M{
		{
			"$group": bson.M{
				"_id": aggName,
				"value": bson.M{
					"$" + aggName: "$" + aggField,
				},
			},
		},
	}
}

// QueryMaxID query max id
func (s *MongoSession) QueryMaxID(model interface{}, result interface{}, aggField string) error {
	collectionName := GetCollectionName(model)
	collection := s.GetCollection(collectionName)
	if collection == nil {
		return ErrorMongoDB(fmt.Sprintf("Could not get collection object by collection name:%s", collectionName))
	}
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Ptr {
		return ErrorMongoDB(fmt.Sprintf("Query max id of collection:%s while the result value must referes a pointer.", collectionName))
	}
	value = value.Elem()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	aggResult := singleGrouppingAggregateResult{}
	pipeline := aggResult.formatSingleGrouppingAggregatePipeline("max", aggField)
	opts := options.Aggregate()
	cur, err := collection.Aggregate(ctx, pipeline, opts)
	defer cur.Close(ctx)
	if err != nil {
		logger.Error.Printf("Query max id of collection:%s failed with error:%v", collectionName, err)
		return err
	}
	for cur.Next(ctx) {
		err = cur.Decode(&aggResult)
		if err != nil {
			logger.Error.Printf("Query max id of collection:%s while decode result failed with error:%v", collectionName, err)
			return err
		}
		value.Set(reflect.ValueOf(aggResult.Value))
		break
	}
	return nil
}

func dispatchMongoBeforeSaveEvent(v interface{}) {
	for _, f := range mongoBeforeSaveObservers {
		f(v)
	}
}

func mongoDefaultBeforeSaveObserver(v interface{}) {

}

func mongoStructFieldGetTagNames(field reflect.StructField) []string {
	name := field.Tag.Get("bson")
	if name == "" {
		name = field.Tag.Get("json")
	}
	if name != "" {
		names := strings.Split(name, ",")
		return names
	}
	return []string{}
}

func mongoOrganizeUpdatesFields(v interface{}) interface{} {
	value := reflect.ValueOf(v)
	t := reflect.TypeOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
		t = t.Elem()
	}
	if value.Kind() == reflect.Struct {
		updates := bson.M{}
		for i := 0; i < value.NumField(); i++ {
			f := t.Field(i)
			tagNames := mongoStructFieldGetTagNames(f)
			if len(tagNames) == 0 {
				continue
			}
			saveFieldName := tagNames[0]
			for _, tagName := range tagNames {
				if tagName == "" || tagName == "-" {
					saveFieldName = ""
					break
				} else if tagName == "omitempty" {
					if f.Type == reflect.TypeOf(primitive.ObjectID{}) {
						if value.Field(i).Interface().(primitive.ObjectID) == primitive.NilObjectID {
							saveFieldName = ""
							break
						}
						continue
					}
					switch f.Type.Kind() {
					case reflect.String:
						if value.Field(i).String() == "" {
							saveFieldName = ""
						}
						break
					case reflect.Ptr:
					case reflect.Array:
						if value.Field(i).IsNil() {
							saveFieldName = ""
						}
						break
					case reflect.Int:
						if value.Field(i).Int() == 0 {
							saveFieldName = ""
						}
						break
					}
					if saveFieldName == "" {
						break
					}
				}
			}
			if saveFieldName == "" {
				continue
			}
			updates[saveFieldName] = value.Field(i).Interface()
		}
		v2 := bson.M{
			"$set": updates,
		}
		return v2
	}
	return v
}

// MongoCollectionIndex struct
type MongoCollectionIndex struct {
	Fields []string
	Unique bool
}

// PrepareCollectionIndexes prepare indexes
func PrepareCollectionIndexes(dst interface{}) []MongoCollectionIndex {
	value := reflect.ValueOf(dst)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	} else if value.Kind() == reflect.String {
		return nil
	}
	indexes := []MongoCollectionIndex{}
	metaField, exists := value.Type().FieldByName("meta")
	if exists {
		indexText := metaField.Tag.Get("index")
		if indexText != "" {
			slices := strings.Split(indexText, ",")
			for _, s := range slices {
				slices2 := strings.Split(s, "|")
				idx := MongoCollectionIndex{
					Fields: []string{},
					Unique: len(slices2) > 1,
				}
				for _, s2 := range slices2 {
					_s2 := strings.TrimSpace(s2)
					if _s2 != "" {
						idx.Fields = append(idx.Fields, _s2)
					}
				}
				if len(idx.Fields) > 0 {
					indexes = append(indexes, idx)
				}
			}
		}
	}
	if len(indexes) > 0 {
		return indexes
	}
	return nil
}

// GetModelPk get model pk
func GetModelPk(model interface{}) reflect.Value {
	value := reflect.ValueOf(model)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	} else if value.Kind() == reflect.String {
		return reflect.Value{}
	}
	pk := "ID"
	metaField, exists := value.Type().FieldByName("meta")
	if exists {
		pkText := metaField.Tag.Get("pk")
		if pkText != "" {
			pk = pkText
		}
	}

	return value.FieldByName(pk)
}
