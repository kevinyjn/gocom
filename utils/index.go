package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Constants
const (
	CharactorsBase = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// ToString convert to string
func ToString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch val.(type) {
	case int:
		return strconv.Itoa(val.(int))
	case int8:
		return strconv.Itoa(int(val.(int8)))
	case int16:
		return strconv.Itoa(int(val.(int16)))
	case int32:
		return strconv.Itoa(int(val.(int32)))
	case int64:
		return strconv.FormatInt(val.(int64), 10)
	case uint:
		return strconv.FormatUint(uint64(val.(uint)), 10)
	case uint8:
		return strconv.FormatUint(uint64(val.(uint8)), 10)
	case uint16:
		return strconv.FormatUint(uint64(val.(uint16)), 10)
	case uint32:
		return strconv.FormatUint(uint64(val.(uint32)), 10)
	case uint64:
		return strconv.FormatUint(val.(uint64), 10)
	case float32:
		return strconv.FormatFloat(float64(val.(float32)), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val.(bool))
	case string:
		return val.(string)
	}
	value := reflect.ValueOf(val)
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}
	var bs []byte
	var err error
	switch value.Type().Kind() {
	case reflect.Struct:
		if reflect.TypeOf(val).Kind() == reflect.Ptr {
			bs, err = json.Marshal(val)
		} else {
			bs, err = json.Marshal(&val)
		}
		break
	case reflect.Array, reflect.Map:
		bs, err = json.Marshal(val)
		break
	}
	if err != nil {
		return err.Error()
	}
	return string(bs)
}

// ToInt convert to int
func ToInt(val interface{}) int {
	return int(ToInt64(val))
}

// ToInt64 converter
func ToInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch val.(type) {
	case int:
		return int64(val.(int))
	case int8:
		return int64(val.(int8))
	case int16:
		return int64(val.(int16))
	case int32:
		return int64(val.(int32))
	case int64:
		return val.(int64)
	case uint:
		return int64(val.(uint))
	case uint8:
		return int64(val.(uint8))
	case uint16:
		return int64(val.(uint16))
	case uint32:
		return int64(val.(uint32))
	case uint64:
		return int64(val.(uint64))
	case float32:
		return int64(val.(float32))
	case float64:
		return int64(val.(float64))
	case string:
		v2, err := strconv.ParseFloat(val.(string), 64)
		if err != nil {
			return 0
		}
		return int64(v2)
	}
	return int64(val.(float64))
}

// ToUint64 converter
func ToUint64(val interface{}) uint64 {
	if val == nil {
		return 0
	}
	return uint64(val.(float64))
}

// ToFloat converter
func ToFloat(val interface{}) float32 {
	if val == nil {
		return 0
	}
	switch val.(type) {
	case string:
		v, err := strconv.ParseFloat(val.(string), 10)
		if nil == err {
			return float32(v)
		}
		return 0
	}
	return val.(float32)
}

// ToDouble converter
func ToDouble(val interface{}) float64 {
	if val == nil {
		return 0
	}
	switch val.(type) {
	case string:
		v, err := strconv.ParseFloat(val.(string), 10)
		if nil == err {
			return v
		}
		return 0
	}
	return val.(float64)
}

// ToBoolean converter
func ToBoolean(val interface{}) bool {
	if val == nil {
		return false
	}
	return val.(bool)
}

// IsEmpty boolean
func IsEmpty(val interface{}) bool {
	if val == nil {
		return true
	}
	switch val.(type) {
	case int:
		return val.(int) == 0
	case int8:
		return val.(int8) == 0
	case int16:
		return val.(int16) == 0
	case int32:
		return val.(int32) == 0
	case int64:
		return val.(int64) == 0
	case uint:
		return val.(uint) == 0
	case uint8:
		return val.(uint8) == 0
	case uint16:
		return val.(uint16) == 0
	case uint32:
		return val.(uint32) == 0
	case uint64:
		return val.(uint64) == 0
	case float32:
		return val.(float32) == 0
	case float64:
		return val.(float64) == 0
	case bool:
		return val.(bool) == false
	case string:
		return val.(string) == ""
	}
	return val.(string) == ""
}

// ObjectIDToString convert objectid to string
func ObjectIDToString(val interface{}) string {
	if val == nil {
		return ""
	}
	return val.(primitive.ObjectID).Hex()
}

// TimestampToHumanYYYYMMDD formatter
func TimestampToHumanYYYYMMDD(ts int64) string {
	return TimeToHuman("YYYYMMDD", time.Unix(ts, 0))
}

// TimestampToHuman formatter
func TimestampToHuman(format string, ts int64) string {
	return TimeToHuman(format, time.Unix(ts, 0))
}

// TimeToHuman formatter
func TimeToHuman(format string, t time.Time) string {
	switch format {
	case "YYYYMMDD":
		return t.Format("20060102")
	case "YYYYMMDDHHmmss":
		return t.Format("20060102150405")
	case "YYYY-MM-DD":
		return t.Format("2006-01-02")
	case "YYYY-MM-DD HH:mm:ss":
		return t.Format("2006-01-02 15:04:05")
	case "YYYY/MM/DD HH:mm:ss":
		return t.Format("2006/01/02 15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}

// HumanToTimestamp converter
func HumanToTimestamp(format string, v string) int64 {
	var t time.Time
	switch format {
	case "YYYYMMDD":
		t, _ = time.Parse("20060102", v)
	case "YYYYMMDDHHmmss":
		t, _ = time.Parse("20060102150405", v)
	case "YYYY-MM-DD":
		t, _ = time.Parse("2006-01-02", v)
	case "YYYY-MM-DD HH:mm:ss":
		t, _ = time.Parse("2006-01-02 15:04:05", v)
	case "YYYY/MM/DD HH:mm:ss":
		t, _ = time.Parse("2006/01/02 15:04:05", v)
	default:
		t, _ = time.Parse("2006-01-02 15:04:05", v)
	}
	return t.Unix()
}

// CurTimestampMiliSec millisecond
func CurTimestampMiliSec() int64 {
	now := time.Now()
	nowTimestamp := now.Unix()*1000 + int64(now.Nanosecond()/1000000)
	return nowTimestamp
}

// ParseRemoteIP parse remote ip
func ParseRemoteIP(val string) string {
	i := strings.LastIndex(val, ":")
	if i < 0 {
		return val
	}
	ip := val[0:i]

	return ip
}

// GetRemoteAddress get http request remote ip
func GetRemoteAddress(r *http.Request) string {
	ra := r.Header.Get("X-Real-IP")
	if ra != "" {
		return ra
	}
	ra = ParseRemoteIP(r.RemoteAddr)
	return ra
}

// GetSOAPAction get soap action
func GetSOAPAction(r *http.Request) string {
	originSoapAction := r.Header.Get("SOAPAction")
	_li := strings.LastIndex(originSoapAction, "/")
	if _li < 0 {
		log.Println("the soap action was not specified")
		return ""
	}
	soapAction := originSoapAction[_li+1 : len(originSoapAction)-1]
	return soapAction
}

// Md5Encode md5 encode
func Md5Encode(val string) string {
	h := md5.New()
	h.Write([]byte(val))
	cipherStr := h.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

// URLEncode url encode
func URLEncode(val string) string {
	return url.QueryEscape(val)
}

// URLDecode url decode
func URLDecode(val string) string {
	s, err := url.QueryUnescape(val)
	if nil != err {
		return val
	}
	return s
}

// ConvertObjectToMapData converter
func ConvertObjectToMapData(v interface{}, tag string) map[string]interface{} {
	result := make(map[string]interface{})
	value := reflect.ValueOf(v)
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Type().Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			f := value.Field(i)
			tagName := value.Type().Field(i).Tag.Get(tag)
			if tagName == "" {
				continue
			}

			if f.Type().Kind() == reflect.Ptr {
				f = f.Elem()
				if !f.IsValid() {
					continue
				}
			}
			if f.Type().Kind() == reflect.Struct {
				result[tagName] = ConvertObjectToMapData(f.Interface(), tag)
			} else {
				result[tagName] = f.Interface()
			}
		}
	}

	return result
}

// CopyObjectSimply copy object
func CopyObjectSimply(src interface{}, dst interface{}, skipEmpty bool) {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)
	if srcValue.Type().Kind() == reflect.Ptr {
		srcValue = srcValue.Elem()
	}
	if dstValue.Type().Kind() == reflect.Ptr {
		dstValue = dstValue.Elem()
	}

	if dstValue.Type().Kind() == reflect.Struct {
		for i := 0; i < dstValue.NumField(); i++ {
			f := dstValue.Field(i)
			srcField := srcValue.FieldByName(dstValue.Type().Field(i).Name)
			if srcField.IsValid() {
				if skipEmpty && srcField.IsZero() {
					continue
				}
				f.Set(srcField)
			}
		}
	}
}

// CopyMapperDataToObject copy data
func CopyMapperDataToObject(src map[string]interface{}, dst interface{}) {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Type().Kind() == reflect.Ptr {
		dstValue = dstValue.Elem()
	}

	if dstValue.Type().Kind() == reflect.Struct {
		for i := 0; i < dstValue.NumField(); i++ {
			f := dstValue.Field(i)
			tagValue := dstValue.Type().Field(i).Tag.Get("json")
			if "" != tagValue {
				keyName := strings.SplitN(tagValue, ",", 1)[0]
				if "-" == keyName {
					continue
				}

				v, ok := src[keyName]
				if ok {
					f.Set(reflect.ValueOf(v))
				}
			}
		}
	}
}

// DeeplyCopyObject deeply copy src object to dst object
func DeeplyCopyObject(src interface{}, dst interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

// IsObjectEquals deeply compare two objects that if equals
func IsObjectEquals(l interface{}, r interface{}) bool {
	var buf1 bytes.Buffer
	var buf2 bytes.Buffer
	var err error
	err = gob.NewEncoder(&buf1).Encode(l)
	if nil != err {
		return false
	}
	err = gob.NewEncoder(&buf2).Encode(r)
	if nil != err {
		return false
	}
	return bytes.Compare(buf1.Bytes(), buf2.Bytes()) == 0
}

// RandomString random string
func RandomString(l int) string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	str := []byte(CharactorsBase)
	sl := len(str)
	result := []byte{}
	for i := 0; i < l; i++ {
		result = append(result, str[r.Intn(sl)])
	}
	return string(result)
}

var _cachedDigistsMap = map[string]map[byte]int{}

// GetDigistIndex get digist index
func GetDigistIndex(digists string, c byte) int {
	m, ok := _cachedDigistsMap[digists]
	if !ok {
		m = map[byte]int{}
		for idx, c1 := range []byte(digists) {
			m[c1] = idx
		}
		_cachedDigistsMap[digists] = m
	}
	idx, ok := m[c]
	if ok {
		return idx
	}
	return -1
}

// IncreaseValueCustomizedDigist increase
func IncreaseValueCustomizedDigist(val string, digists string) string {
	bvals := []byte(val)
	maxidx := len(digists) - 1
	bvalslen := len(bvals)
	resetidx := bvalslen
	for i := bvalslen - 1; i >= 0; i-- {
		idx := GetDigistIndex(digists, bvals[i])
		if 0 <= idx && maxidx > idx {
			bvals[i] = byte(digists[idx+1])
			break
		}
		resetidx = i
	}
	for ; resetidx < bvalslen; resetidx++ {
		bvals[resetidx] = byte(digists[0])
	}
	return string(bvals)
}

// GenUUID generate uuid with upper characters
func GenUUID() string {
	unix32bits := uint32(time.Now().UTC().Unix())
	buff := make([]byte, 12)
	numRead, err := rand.Read(buff)
	if numRead != len(buff) || err != nil {
		return ""
	}
	return fmt.Sprintf("%X-%X-%X-%X-%X-%X", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:8], buff[8:])
}

// GenLoweruuid generate uuid with lower characters
func GenLoweruuid() string {
	return strings.ToLower(GenUUID())
}

// GetLastPartString get last slice of the text
func GetLastPartString(text string) string {
	slices := strings.Split(text, ".")
	l := len(slices)
	if 1 < l {
		return slices[l-1]
	}
	return text
}

// IsStringMapEquals boolean
func IsStringMapEquals(src, dst map[string]string) bool {
	if nil == src || nil == dst || len(src) != len(dst) {
		return false
	}
	var equals bool = true
	for k, v := range src {
		if v != dst[k] {
			equals = false
			break
		}
	}
	return equals
}

// RemoveItemFromList remove element from list
func RemoveItemFromList(v string, l []string) []string {
	for i, e := range l {
		if v == e {
			l = append(l[0:i], l[i+1:]...)
		}
	}
	return l
}

// IsInList whether a value is in the list
func IsInList(key int, list []int) bool {
	for _, v := range list {
		if v == key {
			return true
		}
	}
	return false
}

// CamelString converts xx_yy to XxYy
func CamelString(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

// SnakeString converts XxYy to xx_yy, XxYY to xx_yy
func SnakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}
