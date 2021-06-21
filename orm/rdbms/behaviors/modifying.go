package behaviors

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/kevinyjn/gocom/utils"
)

var MicroSecondsTimestamp = true

// ModifyingBehavior modifying behavior
type ModifyingBehavior struct {
	Obsoleted bool      `xorm:"'obsoleted' Bool index comment('Obsoleted if true or 1')" json:"obsoleted"`
	Changes   int       `xorm:"'changes' version comment('Change version')" json:"changes"`
	CreatedAt Timestamp `xorm:"'created_at' BigInt index comment('Timestamp created')" json:"created_at"` // If you want to save as datetime, add DateTime in tag
	UpdatedAt Timestamp `xorm:"'updated_at' BigInt index comment('Timestamp updated')" json:"updated_at"`
	CreatedBy string    `xorm:"'created_by' Varchar(50) default('') comment('User ID that created')" json:"created_by"`
	UpdatedBy string    `xorm:"'updated_by' Varchar(50) default('') comment('User ID that updated')" json:"updated_by"`
}

func (mb *ModifyingBehavior) BeforeInsert() {
	var now int64
	if MicroSecondsTimestamp {
		now = utils.CurrentMillisecond()
	} else {
		now = time.Now().Unix()
	}
	mb.CreatedAt = Timestamp(now)
	mb.UpdatedAt = Timestamp(now)
}

func (mb *ModifyingBehavior) BeforeUpdate() {
	var now int64
	if MicroSecondsTimestamp {
		now = utils.CurrentMillisecond()
	} else {
		now = time.Now().Unix()
	}
	mb.UpdatedAt = Timestamp(now)
}

// LocalTime parsing and serializing time object as micro seconds number format
type LocalTime time.Time

func (lt *LocalTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	return convertTime(lt, value)
}

// Value implements the driver Valuer interface.
func (lt *LocalTime) Value() (driver.Value, error) {
	fmt.Printf("xxxxxx Value")
	if (*time.Time)(lt).IsZero() {
		return nil, nil
	}
	return ((*time.Time)(lt).Unix()*1000 + int64((*time.Time)(lt).Nanosecond()/1000000)), nil
}

func (lt *LocalTime) Unix() int64 {
	fmt.Printf("xxxxxx Unix")
	return ((*time.Time)(lt).Unix()*1000 + int64((*time.Time)(lt).Nanosecond()/1000000))
}

func (lt *LocalTime) FromDB(value interface{}) error {
	if value == nil {
		return nil
	}
	return convertTime(lt, value)
}

// Now timestamp object
func (lt LocalTime) Now() LocalTime {
	return LocalTime(time.Now())
}

func convertTime(dest *LocalTime, src interface{}) error {
	// Common cases, without reflect.
	switch s := src.(type) {
	case int64:
		*dest = LocalTime(time.Unix(s/1000, s%1000))
		break
	case string:
		t, err := time.Parse("2006-01-02 15:04:05", s)
		if err != nil {
			return err
		}
		*dest = LocalTime(t)
		return nil
	case []uint8:
		t, err := time.Parse("2006-01-02 15:04:05", string(s))
		if err != nil {
			return err
		}
		*dest = LocalTime(t)
		return nil
	case time.Time:
		*dest = LocalTime(s)
		return nil
	case nil:
	default:
		return fmt.Errorf("unsupported driver -> Scan pair: %T -> %T", src, dest)
	}
	return nil
}

type Timestamp int64

func FromTimeToTimestamp(t time.Time) Timestamp {
	return Timestamp(int64(t.UnixNano()) / 1e6)
}

func (t Timestamp) String() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t *Timestamp) Time() time.Time {
	return time.Unix(int64(*t)/1e3, (int64(*t)%1e3)*1e6)
}

func (t *Timestamp) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	*t = Timestamp(utils.ToInt64(value))
	return nil
}

// Value implements the driver Valuer interface.
func (t *Timestamp) Value() (driver.Value, error) {
	fmt.Printf("yyyyyyy Value")
	return t, nil
	// if (*time.Time)(lt).IsZero() {
	// 	return nil, nil
	// }
	// return ((*time.Time)(lt).Unix()*1000 + int64((*time.Time)(lt).Nanosecond()/1000000)), nil
}

func (t *Timestamp) FromDB(b []byte) error {
	var err error
	var value int64
	value, err = strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return nil
	}
	*t = Timestamp(value)
	return nil
}

func (t *Timestamp) ToDB() ([]byte, error) {
	if t == nil {
		return nil, nil
	}
	data := strconv.FormatInt(int64(*t), 10)
	if len(data) == 0 {
		return []byte("0"), nil
	}
	return []byte(data), nil
}

func (t *Timestamp) AutoTime(now time.Time) (interface{}, error) {
	data := int64(now.UnixNano()) / 1e6
	return data, nil
}
