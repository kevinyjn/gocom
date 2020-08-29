package unittests

import (
	"fmt"
	"testing"

	"github.com/kevinyjn/gocom/definations"
)

func TestDSNParser(t *testing.T) {
	dsn := "jdbc:oracle:thin:@(description=(address=(protocol=tcp)(port=1521)(host=127.0.0.1))(connect_data=(service_name=orcl)))"
	option := definations.NewDBConnectionPoolOptionsWithDSN(dsn)
	err := option.ParseDSN()
	if nil != err {
		t.Errorf("Parse %s failed with error:%v", dsn, err)
	} else {
		fmt.Printf("Succeed. %+v\n", option)
	}
}
