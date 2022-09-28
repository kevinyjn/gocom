package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Constants
const (
	CharactorsBase = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// local variables
var (
	_random           = rand.New(rand.NewSource(time.Now().UnixNano()))
	_randomMutex      = sync.Mutex{}
	_cachedDigistsMap = map[string]map[byte]int{}
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
	case []byte:
		return string(val.([]byte))
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
	case reflect.Array, reflect.Map, reflect.Slice:
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
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float64:
		return float32(ToInt64(val))
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
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32:
		return float64(ToInt64(val))
	}
	return val.(float64)
}

// ToBoolean converter
func ToBoolean(val interface{}) bool {
	if val == nil {
		return false
	}
	switch val.(type) {
	case string:
		v, e := strconv.ParseBool(val.(string))
		if nil == e {
			return v
		}
		return false
	default:
		break
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

// IsNumberic boolean
func IsNumberic(val interface{}) bool {
	var isNumberic bool
	switch val.(type) {
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		isNumberic = true
		break
	case float32, float64:
		isNumberic = true
		break
	default:
		isNumberic = regexp.MustCompile(`\d+`).MatchString(ToString(val))
		break
	}
	return isNumberic
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
	case "YYYYMMDD", "yyyyMMdd":
		return t.Format("20060102")
	case "YYYYMMDDHHmmss", "yyyyMMddHHmmss":
		return t.Format("20060102150405")
	case "YYYY-MM-DD", "yyyy-MM-dd":
		return t.Format("2006-01-02")
	case "YYYY-MM-DD HH:mm:ss", "yyyy-MM-dd HH:mm:ss":
		return t.Format("2006-01-02 15:04:05")
	case "YYYY/MM/DD HH:mm:ss", "yyyy/MM/dd HH:mm:ss":
		return t.Format("2006/01/02 15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}

// HumanToTimestamp converter
func HumanToTimestamp(format string, v string) int64 {
	var t time.Time
	switch format {
	case "YYYYMMDD", "yyyyMMdd":
		t, _ = time.ParseInLocation("20060102", v, time.Local)
	case "YYYYMMDDHHmmss", "yyyyMMddHHmmss":
		t, _ = time.ParseInLocation("20060102150405", v, time.Local)
	case "YYYY-MM-DD", "yyyy-MM-dd":
		t, _ = time.ParseInLocation("2006-01-02", v, time.Local)
	case "YYYY-MM-DD HH:mm:ss", "yyyy-MM-dd HH:mm:ss":
		t, _ = time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
	case "YYYY/MM/DD HH:mm:ss", "yyyy/MM/dd HH:mm:ss":
		t, _ = time.ParseInLocation("2006/01/02 15:04:05", v, time.Local)
	default:
		t, _ = time.ParseInLocation("2006-01-02 15:04:05", v, time.Local)
	}
	return t.Unix()
}

// CurrentMillisecond millisecond
func CurrentMillisecond() int64 {
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
	if "" != ra {
		return ra
	}
	ra = r.Header.Get("x-real-ip")
	if "" != ra {
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

// URLPathJoin slices
func URLPathJoin(path string, paths ...string) string {
	path = strings.Trim(path, " ")
	results := []string{}
	endsWithSep := false
	if len(path) > 0 {
		results = append(results, path)
		if '/' == path[len(path)-1] {
			endsWithSep = true
		}
	}
	if len(paths) > 0 {
		for _, p := range paths {
			p = strings.Trim(p, " ")
			if "" == p || "/" == p {
				continue
			}
			if '/' == p[0] {
				if endsWithSep {
					results = append(results, p[1:])
				} else {
					results = append(results, p)
				}
			} else {
				if endsWithSep {
					results = append(results, p)
				} else {
					if len(results) > 0 {
						results = append(results, "/"+p)
					} else {
						results = append(results, p)
					}
				}
			}
			endsWithSep = '/' == p[len(p)-1]
		}
	}
	return strings.Join(results, "")
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
	str := []byte(CharactorsBase)
	sl := len(str)
	result := []byte{}
	_randomMutex.Lock()
	for i := 0; i < l; i++ {
		result = append(result, str[_random.Intn(sl)])
	}
	_randomMutex.Unlock()
	return string(result)
}

// RandomInt random int
func RandomInt(n int) int {
	_randomMutex.Lock()
	result := _random.Intn(n)
	_randomMutex.Unlock()
	return result
}

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
	_randomMutex.Lock()
	numRead, err := _random.Read(buff)
	_randomMutex.Unlock()
	if numRead != len(buff) || err != nil {
		return ""
	}
	return fmt.Sprintf("%X-%X-%X-%X-%X-%X", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:8], buff[8:])
}

// GenLoweruuid generate uuid with lower characters
func GenLoweruuid() string {
	unix32bits := uint32(time.Now().UTC().Unix())
	buff := make([]byte, 12)
	_randomMutex.Lock()
	numRead, err := _random.Read(buff)
	_randomMutex.Unlock()
	if numRead != len(buff) || err != nil {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x-%x", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:8], buff[8:])
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

// PascalCaseString converts xx_yy to XxYy
func PascalCaseString(s string) string {
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

// CamelCaseString converts xx_yy to xxYY
func CamelCaseString(s string) string {
	if strings.HasPrefix(s, "ID") {
		s = "id" + s[2:]
	}
	s = PascalCaseString(s)
	s = strings.ToLower(s[0:1]) + s[1:]
	return s
}

// SnakeCaseString converts XxYy to xx_yy, XxYY to xx_yy
func SnakeCaseString(s string) string {
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

// KebabCaseString converts XxYy to xx-yy, XxYY to xx-yy
func KebabCaseString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '-')
		}
		if d == '_' {
			d = '-'
		} else if d != '-' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

// CamelCaseSlices splits XxYy into ["Xx", "Yy"]
func CamelCaseSlices(s string) []string {
	num := len(s)
	results := []string{}
	j := 0
	jj := false
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && jj {
			results = append(results, s[j:i])
			j = i
			jj = false
		} else {
			jj = true
		}
	}
	if j < num {
		results = append(results, s[j:])
	}
	return results
}

// HashCode hashes a buffer to a unique hashcode
func HashCode(content []byte) uint32 {
	return crc32.ChecksumIEEE(content)
}

// HashCodeAsInt hashes a buffer to a unique hashcode
func HashCodeAsInt(content []byte) int {
	v := int(crc32.ChecksumIEEE(content))
	if 0 > v {
		return -v
	}
	return v
}

// SubStringUTF8 sub utf-8 encode string to support chinese text submation
// this method avoids the traditional []rune() method to optimize the execution time
func SubStringUTF8(s string, length int, start ...int) string {
	offset := 0
	if nil != start {
		offset = start[0]
	}
	return SubStringFromUTF8(s, length, offset, false)
}

// SubStringFromUTF8 sub utf-8 encode string to support chinese text submation
// this method avoids the traditional []rune() method to optimize the execution time
func SubStringFromUTF8(s string, length int, start int, markDots bool) string {
	var n, endpos, startpos int
	cutted := false
	length = length + start

	for endpos = range s {
		if n == start {
			startpos = endpos
		} else if n >= length {
			cutted = true
			break
		}
		n++
	}

	if cutted && markDots {
		return s[startpos:endpos] + "..."
	}
	return s[startpos:endpos]
}

// IsCapital if strings has a capital character
func IsCapital(s string) bool {
	if len(s) <= 0 {
		return false
	}
	c := s[0]
	if c >= 'A' && c <= 'Z' {
		return true
	}
	return false
}
