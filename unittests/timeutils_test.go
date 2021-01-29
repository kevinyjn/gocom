package unittests

import (
	"testing"
	"time"

	"github.com/kevinyjn/gocom/utils"
)

func TestTimeFormattionParse(t *testing.T) {
	tv := "2021-01-29 11:37:25"
	v := utils.HumanToTimestamp("YYYY-MM-DD HH:mm:ss", tv)
	t2 := time.Unix(v, 0)
	v2 := utils.TimeToHuman("YYYY-MM-DD HH:mm:ss", t2)
	AssertEquals(t, tv, v2, "utils.HumanToTimestamp")
	// v3, _ := time.Parse("2006-01-02 15:04:05", tv)
	// v4 := utils.TimeToHuman("YYYY-MM-DD HH:mm:ss", v3)
	// AssertEquals(t, tv, v4, "utils.HumanToTimestamp")
}
