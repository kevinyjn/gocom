package events

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/mq/mqenv"
	"github.com/kevinyjn/gocom/utils"
)

// SerializationType for event
type SerializationType int

// Constants
const (
	EventSerializationProtobuf SerializationType = iota
	EventSerializationJSON
)

// Event base event interface
type Event interface {
	Initialize(category string, from string, expireDurationAsSeconds int64, payload interface{}) bool
	GetIdentifier() string
	SetIdentifier(string)
	GetStatus() int
	SetStatus(int)
	GetDescription() string
	SetDescription(string)
	GetCategory() string
	SetCategory(string)
	GetFrom() string
	SetFrom(string)
	GetAppID() string
	SetAppID(string)
	GetUserID() string
	SetUserID(string)
	GetReplyTo() string
	SetReplyTo(replyTo string)
	GetCorrelationID() string
	SetCorrelationID(string)
	GetTimestampMilis() int64
	SetTimestampMilis(timestampMilis int64)
	GetExpirationMilis() int64
	SetExpirationMilis(expiresMilis int64)
	IsExpired() bool
	GetShouldRetryTimes() int
	SetShouldRetryTimes(retries int)
	GetRetriedTimes() int
	OnRetry()
	GetHeaders() map[string]string
	SetHeader(name string, value string)
	GetPayload() interface{}
	SetPayload(payload interface{})
	GetOriginBody() []byte
	SetOriginBody([]byte)
	GetProcessResult() interface{}
	SetProcessResult(interface{})
	GetSerializationType() SerializationType
	ContentType() string
	Serialize() ([]byte, error)
	ParseFrom([]byte) error
	Clone() Event
}

// AbstractEvent abstract implement of event
type AbstractEvent struct {
	Identifier       string            `json:"id"`
	Status           int               `json:"code"`
	Description      string            `json:"message"`
	Category         string            `json:"category"`
	From             string            `json:"from"`
	AppID            string            `json:"appId"`
	UserID           string            `json:"userId"`
	TimestampMilis   int64             `json:"timestamp"`
	ExpiresMilis     int64             `json:"expires"`
	ShouldRetryTimes int               `json:"retry"`
	RetriedTimes     int               `json:"retried"`
	ReplyTo          string            `json:"replyTo"`
	CorrelationID    string            `json:"correlationId"`
	Headers          map[string]string `json:"headers,omitempty"`
	Payload          interface{}       `json:"payload,omitempty"`
	OriginBody       []byte            `json:"originBody,omitempty"`
	ProcessResult    interface{}       `json:"processResponse,omitempty"`
}

// Initialize members
func (e *AbstractEvent) Initialize(category, from string, expireDurationAsSeconds int64, payload interface{}) bool {
	e.Identifier = utils.GenLoweruuid()
	e.Category = category
	e.From = from
	e.TimestampMilis = utils.CurrentMillisecond()
	if expireDurationAsSeconds > 0 {
		e.ExpiresMilis = e.TimestampMilis + (expireDurationAsSeconds * 1000)
	} else {
		e.ExpiresMilis = 0
	}
	e.Payload = payload
	e.RetriedTimes = 0
	return true
}

// GetIdentifier events identifier
func (e *AbstractEvent) GetIdentifier() string {
	return e.Identifier
}

// SetIdentifier events identifier
func (e *AbstractEvent) SetIdentifier(identifier string) {
	e.Identifier = identifier
}

// GetStatus events status code
func (e *AbstractEvent) GetStatus() int {
	return e.Status
}

// SetStatus events status code
func (e *AbstractEvent) SetStatus(status int) {
	e.Status = status
}

// GetDescription events result message
func (e *AbstractEvent) GetDescription() string {
	return e.Description
}

// SetDescription events result message
func (e *AbstractEvent) SetDescription(message string) {
	e.Description = message
}

// GetCategory events category
func (e *AbstractEvent) GetCategory() string {
	return e.Category
}

// SetCategory events category
func (e *AbstractEvent) SetCategory(category string) {
	e.Category = category
}

// GetFrom the microservice or system where event comes from
func (e *AbstractEvent) GetFrom() string {
	return e.From
}

// SetFrom the microservice or system where event comes from
func (e *AbstractEvent) SetFrom(from string) {
	e.From = from
}

// GetAppID the application id where event comes from
func (e *AbstractEvent) GetAppID() string {
	return e.AppID
}

// SetAppID the application id where event comes from
func (e *AbstractEvent) SetAppID(appID string) {
	e.AppID = appID
}

// GetUserID the user id where event comes from
func (e *AbstractEvent) GetUserID() string {
	return e.UserID
}

// SetUserID the user id where event comes from
func (e *AbstractEvent) SetUserID(userID string) {
	e.UserID = userID
}

// GetReplyTo if specified, the consumer should reply the process result to the topic of queue that the field specified
func (e *AbstractEvent) GetReplyTo() string {
	return e.ReplyTo
}

// SetReplyTo the consumer should reply the process result to the topic of queue
func (e *AbstractEvent) SetReplyTo(replyTo string) {
	e.ReplyTo = replyTo
}

// GetCorrelationID unique message identifier generated by producer from reply to
func (e *AbstractEvent) GetCorrelationID() string {
	return e.CorrelationID
}

// SetCorrelationID unique message identifier generated by producer from reply to
func (e *AbstractEvent) SetCorrelationID(correlationID string) {
	e.CorrelationID = correlationID
}

// GetTimestampMilis the timestamp(as mili-second) that the event created
func (e *AbstractEvent) GetTimestampMilis() int64 {
	return e.TimestampMilis
}

// SetTimestampMilis the timestamp(as mili-second) that the event created
func (e *AbstractEvent) SetTimestampMilis(timestampMilis int64) {
	e.TimestampMilis = timestampMilis
}

// GetExpirationMilis the timestamp(as mili-second) that the event would expired, the event would be considered as no-expires if expiration is 0
func (e *AbstractEvent) GetExpirationMilis() int64 {
	return e.ExpiresMilis
}

// SetExpirationMilis the timestamp(as mili-second) that the event would expired, the event would be considered as no-expires if expiration is 0
func (e *AbstractEvent) SetExpirationMilis(expiresMilis int64) {
	e.ExpiresMilis = expiresMilis
}

// IsExpired if the event were expired
func (e *AbstractEvent) IsExpired() bool {
	return e.ExpiresMilis > 0 && e.ExpiresMilis < utils.CurrentMillisecond()
}

// GetShouldRetryTimes how many times that the event should be retried
func (e *AbstractEvent) GetShouldRetryTimes() int {
	if 0 < e.ShouldRetryTimes {
		return e.ShouldRetryTimes
	}
	return 0
}

// SetShouldRetryTimes how many times that the event should be retried
func (e *AbstractEvent) SetShouldRetryTimes(retries int) {
	if retries > 0 {
		e.ShouldRetryTimes = retries
	} else {
		e.ShouldRetryTimes = 0
	}
}

// GetRetriedTimes the retry times that the event already retried
func (e *AbstractEvent) GetRetriedTimes() int {
	return e.RetriedTimes
}

// OnRetry triggers that the event were retried
func (e *AbstractEvent) OnRetry() {
	e.RetriedTimes++
}

// GetHeaders key-value headers
func (e *AbstractEvent) GetHeaders() map[string]string {
	return e.Headers
}

// SetHeader with name and value
func (e *AbstractEvent) SetHeader(name string, value string) {
	if nil == e.Headers {
		e.Headers = map[string]string{}
	}
	e.Headers[name] = value
}

// GetPayload events payload data
func (e *AbstractEvent) GetPayload() interface{} {
	return e.Payload
}

// SetPayload events payload data
func (e *AbstractEvent) SetPayload(paylaod interface{}) {
	e.Payload = paylaod
}

// GetOriginBody events origin request data
func (e *AbstractEvent) GetOriginBody() []byte {
	return e.OriginBody
}

// SetOriginBody events origin request data
func (e *AbstractEvent) SetOriginBody(originBody []byte) {
	e.OriginBody = originBody
}

// GetProcessResult events payload data
func (e *AbstractEvent) GetProcessResult() interface{} {
	return e.ProcessResult
}

// SetProcessResult events payload data
func (e *AbstractEvent) SetProcessResult(paylaod interface{}) {
	e.ProcessResult = paylaod
}

// GetSerializationType for serializing
func (e *AbstractEvent) GetSerializationType() SerializationType {
	return EventSerializationJSON
}

// CongentType of serializing content
func (e *AbstractEvent) ContentType() string {
	switch e.GetSerializationType() {
	case EventSerializationJSON:
		return "application/json"
	case EventSerializationProtobuf:
		return "application/protobuf"
	}
	return "application/json"
}

// Serialize serializes the event
func (e *AbstractEvent) Serialize() ([]byte, error) {
	var body []byte
	var err error
	switch e.GetSerializationType() {
	case EventSerializationJSON:
		body, err = json.Marshal(e)
		break
	case EventSerializationProtobuf:
		err = fmt.Errorf("Unsupported serializing type:%+v", e.GetSerializationType())
		break
	default:
		err = fmt.Errorf("Unsupported serializing type:%+v", e.GetSerializationType())
		break
	}
	if nil != err {
		logger.Error.Printf("Serialize event %+v as mode(%+v) failed with error:%v", e, e.GetSerializationType(), err)
	}
	return body, err
}

// ParseFrom parses the body buffer and filles to event data
func (e *AbstractEvent) ParseFrom(body []byte) error {
	var err error
	switch e.GetSerializationType() {
	case EventSerializationJSON:
		err = json.Unmarshal(body, e)
		break
	case EventSerializationProtobuf:
		err = fmt.Errorf("Unsupported serializing type:%+v", e.GetSerializationType())
		break
	default:
		err = fmt.Errorf("Unsupported serializing type:%+v", e.GetSerializationType())
		break
	}
	if nil != err {
		logger.Error.Printf("Parse event buffer %v from mode(%+v) failed with error:%v", body, e.GetSerializationType(), err)
	}
	return err
}

// standardQueryEvent the package that comes from consumer message
type standardQueryEvent struct {
	AbstractEvent
}

// CreateEventFromConsumerMessage generates filter event by consumer message
func CreateEventFromConsumerMessage(category string, handleName string, msg mqenv.MQConsumerMessage) Event {
	ev := standardQueryEvent{}
	ev.Initialize(category, handleName, 0, msg.Body)
	ev.loadFromConsumerMessage(msg)
	return &ev
}

// CreateEventFromConsumerMessageAndPayload generates filter event by consumer message and payload object
func CreateEventFromConsumerMessageAndPayload(category string, handleName string, msg mqenv.MQConsumerMessage, payloadObject interface{}) Event {
	ev := standardQueryEvent{}
	ev.Initialize(category, handleName, 0, payloadObject)
	ev.loadFromConsumerMessage(msg)
	return &ev
}

func (ev *standardQueryEvent) loadFromConsumerMessage(msg mqenv.MQConsumerMessage) {
	ev.SetTimestampMilis(msg.Timestamp.Unix()*1000 + int64(msg.Timestamp.Nanosecond()/1000000))
	ev.SetIdentifier(msg.MessageID)
	ev.SetAppID(msg.AppID)
	ev.SetUserID(msg.UserID)
	ev.SetCorrelationID(msg.CorrelationID)
	ev.SetReplyTo(msg.ReplyTo)
	ev.SetOriginBody(msg.Body)
	headers := map[string]string{}
	if nil != msg.Headers {
		for k, v := range msg.Headers {
			k = strings.ToLower(k)
			ev.SetHeader(k, v)
			headers[k] = v
		}
	}

	// parse expires
	expireFactor := int64(1)
	expires := headers["expiremilis"]
	if "" == expires {
		expires = headers["expire"]
		if "" != expires {
			expireFactor = 1000
		}
	}
	if "" != expires {
		expireValue, err := strconv.ParseInt(expires, 10, 64)
		if nil == err {
			ev.SetExpirationMilis(expireValue * expireFactor)
		} else {
			logger.Error.Printf("parse expire timestamp %s as integer failed with error:%v", expires, err)
		}
	}

}

// Clone object
func (e *standardQueryEvent) Clone() Event {
	ev := standardQueryEvent(*e)
	return &ev
}

// CreatePublishMessageFromEventAndPayload creates
func CreatePublishMessageFromEventAndPayload(event Event, payload []byte) mqenv.MQPublishMessage {
	pubMsg := mqenv.MQPublishMessage{
		Body:          payload,
		Exchange:      "",
		RoutingKey:    "",
		CorrelationID: event.GetCorrelationID(),
		ReplyTo:       event.GetReplyTo(),
		MessageID:     event.GetIdentifier(),
		AppID:         event.GetAppID(),
		UserID:        event.GetUserID(),
		ContentType:   "application/json",
		Headers:       event.GetHeaders(),
	}
	return pubMsg
}