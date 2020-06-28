package orderutil

import (
	"math/rand"
	"strconv"
	"time"
)

// NewOrderID generate order id
func NewOrderID(prefix string) string {
	now := time.Now()
	timeval := now.Format("20060102150405")
	endfix := 1000000000 + rand.Intn(1000000000)
	return prefix + timeval + strconv.Itoa(endfix)
}

// NewShortOrderID generage order id in short mode
func NewShortOrderID(prefix string) string {
	now := time.Now()
	endfix := 10000 + rand.Intn(10000)
	return prefix + strconv.FormatInt(now.Unix(), 10) + strconv.Itoa(endfix)
}
