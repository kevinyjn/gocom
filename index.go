package gocom

import (
	"crypto/rand"
	"fmt"
	"time"
)

// GlobalUUID global uuid
var GlobalUUID = GenUUID()

// GenUUID generate uuid
func GenUUID() string {
	unix32bits := uint32(time.Now().UTC().Unix())
	buff := make([]byte, 12)
	numRead, err := rand.Read(buff)
	if numRead != len(buff) || err != nil {
		return ""
	}
	return fmt.Sprintf("%X-%X-%X-%X-%X-%X", unix32bits, buff[0:2], buff[2:4], buff[4:6], buff[6:8], buff[8:])
}
