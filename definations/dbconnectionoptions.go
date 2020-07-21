package definations

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/utils"

	// justifying
	_ "github.com/denisenkom/go-mssqldb" // sqlserver
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql" // mysql, tidb
	_ "github.com/godror/godror"       // oracle
	_ "github.com/lib/pq"              // postgres, cockroachdb
	_ "github.com/mattn/go-sqlite3"    // sqlite
)

// Constants
const (
	_EngineTypeMin    = 1
	EngineMySQL       = 1
	EngineMSSQL       = 2
	EngineOracle      = 3
	EnginePostgres    = 4
	EngineSQLite      = 5
	EngineTiDB        = 6
	EngineCockroachDB = 7
	_EngineTypeMax    = EngineCockroachDB
)

// DBConnectionPoolOptions options
type DBConnectionPoolOptions struct {
	Engine              int
	Host                string `property:"Database Host"`
	Port                int    `property:"Database Port"`
	User                string `property:"Database User"`
	Password            string `property:"Password"`
	ServiceName         string `property:"Database Service Name"`
	ServiceID           string `property:"Database Service ID"`
	Database            string `property:"Database Name"`
	DSN                 string `property:"Database Connection URL"`
	MaxWaitTime         int    `property:"Max Wait Time"`
	MaxTotalConnections int    `property:"Max Total Connections"`
	SSHTunnelDSN        string `property:"SSH Tunnel DSN"`
}

// DBConnectionData formatted connection information
type DBConnectionData struct {
	Driver          string
	ConnString      string
	ConnDescription string
}

// NewDBConnectionPoolOptionsWithDSN DBConnectionPoolOptions
func NewDBConnectionPoolOptionsWithDSN(dsn string) *DBConnectionPoolOptions {
	opts := &DBConnectionPoolOptions{
		DSN: dsn,
	}
	err := opts.ParseDSN()
	if nil != err {
		logger.Error.Printf("Parse database connection string:%s failed with error:%v", dsn, err)
		return nil
	}
	return opts
}

// ParseDSN parse dsn field from options
func (o *DBConnectionPoolOptions) ParseDSN() error {
	if "" != o.DSN {
		dsn := o.DSN
		if strings.HasPrefix(dsn, "jdbc:") {
			dsn = dsn[5:]
		}
		if strings.HasPrefix(dsn, "oracle:thin:") {
			o.Engine = EngineOracle
			o.Port = 1521
			dsn = dsn[12:]
			if !strings.HasPrefix(dsn, "@") && !strings.HasPrefix(dsn, "//") {
				var slices []string
				if strings.Contains(dsn, "//") {
					slices = strings.SplitN(dsn, "//", 2)
				} else {
					slices = strings.SplitN(dsn, "@", 2)
				}
				if len(slices) > 1 {
					dsn = slices[1]
				}
				slices = strings.SplitN(slices[0], "/", 2)
				if strings.Contains(slices[0], "@") {
					slices = strings.SplitN(slices[0], "@", 2)
				}
				if strings.Contains(slices[0], ":") {
					slices = strings.SplitN(slices[0], ":", 2)
				}
				o.User = utils.URLDecode(slices[0])
				if len(slices) > 1 {
					o.Password = utils.URLDecode(slices[1])
				}
			}
			if !strings.HasPrefix(dsn, "(") {
				return o.parseCommonDSN(dsn)
			}
			r, _ := regexp.Compile(`\((?:host|HOST)=([\w\.\d]+)\)`)
			ss := r.FindStringSubmatch(dsn)
			if len(ss) > 1 {
				o.Host = ss[1]
			}
			r, _ = regexp.Compile(`\((?:port|PORT)=(\d+)\)`)
			ss = r.FindStringSubmatch(dsn)
			if len(ss) > 1 {
				o.Port, _ = strconv.Atoi(ss[1])
			}
			r, _ = regexp.Compile(`\((?:service_name|SERVICE_NAME)=([\w\.\d]+)\)`)
			ss = r.FindStringSubmatch(dsn)
			if len(ss) > 1 {
				o.ServiceName = ss[1]
			}
			r, _ = regexp.Compile(`\((?:sid_name|SID_NAME)=([\w\.\d]+)\)`)
			ss = r.FindStringSubmatch(dsn)
			if len(ss) > 1 {
				o.ServiceID = ss[1]
			}
			r, _ = regexp.Compile(`\((?:instance_name|INSTANCE_NAME)=([\w\.\d]+)\)`)
			ss = r.FindStringSubmatch(dsn)
			if len(ss) > 1 {
				o.Database = ss[1]
			}
		} else {
			return o.parseCommonDSN(dsn)
		}
	}
	return nil
}

func (o *DBConnectionPoolOptions) parseCommonDSN(dsn string) error {
	slices := strings.SplitN(dsn, "://", 2)
	switch strings.ToLower(slices[0]) {
	case "sqlserver":
		o.Engine = EngineMSSQL
		o.Port = 1433
		break
	case "mysql":
		o.Engine = EngineMySQL
		o.Port = 3306
		break
	case "oracle":
		o.Engine = EngineOracle
		o.Port = 1521
		break
	case "postgres":
		o.Engine = EnginePostgres
		o.Port = 5432
		break
	case "sqlite", "file":
		o.Engine = EngineSQLite
		o.Port = 1000
		if len(slices) > 1 && "" == o.Host {
			o.Host = slices[1]
		}
		return nil
		break
	case "tidb":
		o.Engine = EngineTiDB
		o.Port = 4000
	case "cockroach":
		o.Engine = EngineCockroachDB
		o.Port = 26257
		break
	default:
		return fmt.Errorf("Parsing database connection string:%s while could not get connection schema", dsn)
	}
	if len(slices) < 2 {
		return fmt.Errorf("Parsing database connection string:%s while could not find connection string part", dsn)
	}
	slices = strings.SplitN(slices[1], "@", 2)
	hostText := slices[0]
	if len(slices) > 1 {
		hostText = slices[1]
		slices = strings.SplitN(slices[0], ":", 2)
		o.User = slices[0]
		if len(slices) > 1 {
			o.Password = slices[1]
		}
	}
	if strings.Contains(hostText, "/") {
		slices = strings.SplitN(hostText, "/", 2)
		o.Database = strings.TrimRight(slices[1], " ;&?,")
	} else if strings.Contains(hostText, ";") {
		slices = strings.SplitN(hostText, ";", 2)
		slices2 := strings.Split(slices[1], ";")
		for _, s := range slices2 {
			slices3 := strings.SplitN(s, "=", 2)
			if "DatabaseName" == slices3[0] && len(slices3) > 1 {
				o.Database = slices3[1]
				break
			}
		}
	} else {
		slices = []string{hostText, ""}
	}
	slices = strings.SplitN(slices[0], ":", 2)
	o.Host = slices[0]
	if len(slices) > 1 {
		if strings.Contains(slices[1], ":") {
			slices2 := strings.SplitN(slices[1], ":", 2)
			slices[1] = slices2[0]
			slices3 := strings.SplitN(slices2[1], "/", 2)
			o.ServiceID = slices3[0]
			if len(slices3) > 1 {
				o.Database = slices3[1]
			}
		} else if strings.Contains(slices[1], "/") {
			slices2 := strings.SplitN(slices[1], "/", 2)
			slices[1] = slices2[0]
			o.Database = slices2[1]
		}
		port, err := strconv.Atoi(slices[1])
		if nil != err {
			logger.Error.Printf("Convert port part:%s from dsn:%s failed with error:%s", slices[1], dsn, err.Error())
		} else {
			o.Port = port
		}
	}
	if strings.Contains(o.Database, "/") {
		slices = strings.Split(o.Database, "/")
		o.ServiceName = slices[0]
		o.Database = slices[1]
	}
	return nil
}

// GetConnectionData prepare dsn and format connection data
func (o *DBConnectionPoolOptions) GetConnectionData() (DBConnectionData, error) {
	options := o
	dbHost := options.Host
	dbPort := options.Port
	connData := DBConnectionData{}

	switch options.Engine {
	case EngineMSSQL:
		connData.Driver = "sqlserver"
		connData.ConnString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;encrypt=disable", dbHost, options.User, options.Password, dbPort, options.Database)
		connData.ConnDescription = fmt.Sprintf("mssql://%s@%s:%d/%s", options.User, options.Host, options.Port, options.Database)
		break
	case EngineMySQL, EngineTiDB:
		connData.Driver = "mysql"
		cfg := mysql.NewConfig()
		cfg.User = options.User
		cfg.Passwd = options.Password
		cfg.Net = "tcp"
		cfg.Addr = fmt.Sprintf("%s:%d", dbHost, dbPort)
		cfg.DBName = options.Database
		connData.ConnString = cfg.FormatDSN()
		connData.ConnDescription = fmt.Sprintf("mysql://%s:@%s:%d/%s?charset=utf8", options.User, options.Host, options.Port, options.Database)
		break
	case EngineOracle:
		connData.Driver = "godror"
		if "" != options.ServiceID {
			connData.ConnString = fmt.Sprintf("%s/%s@%s:%d:%s", utils.URLEncode(options.User), utils.URLEncode(options.Password), dbHost, dbPort, options.ServiceID)
		} else {
			connData.ConnString = fmt.Sprintf("%s/%s@%s:%d/%s", utils.URLEncode(options.User), utils.URLEncode(options.Password), dbHost, dbPort, options.ServiceName)
		}
		if "" != options.Database {
			connData.ConnString = connData.ConnString + "/" + options.Database
		}
		connData.ConnDescription = connData.ConnString
		break
	case EnginePostgres, EngineCockroachDB:
		connData.Driver = "postgres"
		connData.ConnString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", dbHost, dbPort, options.User, options.Password, options.Database)
		connData.ConnDescription = fmt.Sprintf("postgres://%s:@%s:%d/%s", options.User, options.Host, options.Port, options.Database)
		break
	case EngineSQLite:
		connData.Driver = "sqlite3"
		connData.ConnString = options.Host
		connData.ConnDescription = options.Host
		break
	default:
		logger.Error.Printf("Unsupported database engine: %d", options.Engine)
		// p.cleanSSHTunnel()
		return connData, errors.New("Unsupported database engine")
	}
	return connData, nil
}
