package unittests

import (
	"testing"

	"github.com/kevinyjn/gocom/netutils/dboptions"
	"github.com/kevinyjn/gocom/testingutil"
)

func TestDSNParser(t *testing.T) {
	dsn := "jdbc:oracle:thin:@(description=(address=(protocol=tcp)(port=1521)(host=127.0.0.1))(connect_data=(service_name=orcl)))"
	option := dboptions.NewDBConnectionPoolOptionsWithDSN(dsn)
	err := option.ParseDSN()
	testingutil.AssertNil(t, err, "option.ParseDSN error")
}
