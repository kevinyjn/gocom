package dboptions

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/netutils/pinger"
	"github.com/kevinyjn/gocom/netutils/sshtunnel"
	"github.com/kevinyjn/gocom/utils"

	// justifying
	_ "github.com/denisenkom/go-mssqldb" // sqlserver
	"github.com/go-sql-driver/mysql"
	_ "github.com/godror/godror"    // oracle
	_ "github.com/lib/pq"           // postgres, cockroachdb
	_ "github.com/mattn/go-sqlite3" // sqlite
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
	DSN                 string `property:"Database Connection URL" validate:"required"`
	DriverClassname     string `property:"Database Driver Class Name" validate:"required"`
	Host                string `property:"Database Host,descriptor:false"`
	Port                int    `property:"Database Port,descriptor:false"`
	User                string `property:"Database User"`
	Password            string `property:"Password,category:password"`
	ServiceName         string `property:"Database Service Name,descriptor:false"`
	ServiceID           string `property:"Database Service ID,descriptor:false"`
	Database            string `property:"Database Name,descriptor:false"`
	MaxWaitTime         int    `property:"Max Wait Time,default:500 millis" validate:"required"`
	MaxTotalConnections int    `property:"Max Total Connections,default:8" validate:"required"`
	SSHTunnelDSN        string `property:"SSH Tunnel DSN"`
	sshTunnel           *sshtunnel.TunnelForwarder
	connectionParams    map[string]string
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
			if !strings.HasPrefix(dsn, "(") && !strings.HasPrefix(dsn, "@(") {
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
	if len(slices) <= 1 {
		return fmt.Errorf("Parsing database connection string:%s with invalid format", dsn)
	}
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
	case "postgres", "postgresql":
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
	case "tidb":
		o.Engine = EngineTiDB
		o.Port = 4000
	case "cockroach", "cockroachdb":
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
	o.connectionParams = map[string]string{}
	if strings.Contains(hostText, "/") {
		slices = strings.SplitN(hostText, "/", 2)
		o.Database = strings.TrimRight(slices[1], " ")
		if strings.Contains(slices[1], "?") {
			dbSlices := strings.SplitN(slices[1], "?", 2)
			o.Database = strings.Trim(dbSlices[0], " ")
			paramsParts := strings.Split(dbSlices[1], "&")
			for _, paramsEle := range paramsParts {
				paramSlices := strings.SplitN(paramsEle, "=", 2)
				if len(paramSlices) > 1 {
					o.connectionParams[utils.URLDecode(paramSlices[0])] = utils.URLDecode(paramSlices[1])
				} else {
					o.connectionParams[utils.URLDecode(paramSlices[0])] = ""
				}
			}
		}
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
			logger.Error.Printf("Convert port part:%s from dsn:%s failed with error:%v", slices[1], dsn, err)
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
	connData := DBConnectionData{}
	dbHost, dbPort, err := o.ensureSSHTunnel()

	switch o.Engine {
	case EngineMSSQL:
		connData.Driver = "sqlserver"
		connData.ConnString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;encrypt=disable", dbHost, o.User, o.Password, dbPort, o.Database)
		connData.ConnDescription = fmt.Sprintf("mssql://%s:%s@%s:%d/%s", o.User, strings.Repeat("*", len(o.Password)), o.Host, o.Port, o.Database)
		break
	case EngineMySQL, EngineTiDB:
		connData.Driver = "mysql"
		cfg := mysql.NewConfig()
		cfg.User = o.User
		cfg.Passwd = o.Password
		cfg.Net = "tcp"
		cfg.Addr = fmt.Sprintf("%s:%d", dbHost, dbPort)
		cfg.DBName = o.Database
		connData.ConnString = cfg.FormatDSN()
		connData.ConnDescription = fmt.Sprintf("mysql://%s:%s@%s:%d/%s?charset=utf8", o.User, strings.Repeat("*", len(o.Password)), o.Host, o.Port, o.Database)
		break
	case EngineOracle:
		connData.Driver = "godror"
		servicePart := ""
		if "" != o.ServiceID {
			servicePart = ":" + o.ServiceID
		} else {
			servicePart = "/" + o.ServiceName
		}
		connData.ConnString = fmt.Sprintf("%s/\"%s\"@%s:%d%s", o.User, strings.ReplaceAll(o.Password, "\"", "\\\""), dbHost, dbPort, servicePart)
		if "" != o.Database {
			connData.ConnString = connData.ConnString + "/" + o.Database
		}
		connData.ConnDescription = fmt.Sprintf("oracle://%s:%s@%s:%d%s/%s", o.User, strings.Repeat("*", len(o.Password)), o.Host, o.Port, servicePart, o.Database)
		break
	case EnginePostgres, EngineCockroachDB:
		connData.Driver = "postgres"
		exParams := ""
		descripExParams := ""
		if o.connectionParams != nil {
			for k, v := range o.connectionParams {
				exParams = exParams + fmt.Sprintf(" %s=%s", k, v)
				sep2 := "&"
				if "" == descripExParams {
					sep2 = "?"
				}
				descripExParams = descripExParams + fmt.Sprintf("%s%s=%s", sep2, k, utils.URLEncode(v))
			}
		}
		connData.ConnString = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s%s", dbHost, dbPort, o.User, o.Password, o.Database, exParams)
		connData.ConnDescription = fmt.Sprintf("postgres://%s:%s@%s:%d/%s%s", o.User, strings.Repeat("*", len(o.Password)), o.Host, o.Port, o.Database, descripExParams)
		break
	case EngineSQLite:
		connData.Driver = "sqlite3"
		connData.ConnString = o.Host
		connData.ConnDescription = o.Host
		break
	default:
		logger.Error.Printf("Unsupported database engine: %d", o.Engine)
		// p.cleanSSHTunnel()
		return connData, errors.New("Unsupported database engine")
	}
	return connData, err
}

// Cleanup clean sshtunnel
func (o *DBConnectionPoolOptions) Cleanup() {
	o.cleanSSHTunnel()
}

func (o *DBConnectionPoolOptions) ensureSSHTunnel() (string, int, error) {
	dbHost := o.Host
	dbPort := o.Port
	if "" != o.SSHTunnelDSN && !pinger.Connectable(o.Host, o.Port) {
		sshTunnel, err := sshtunnel.NewSSHTunnel(o.SSHTunnelDSN, o.Host, o.Port)
		if nil != err {
			return dbHost, dbPort, err
		}

		err = sshTunnel.Start()
		if nil != err {
			logger.Error.Printf("Start SSH Tunnel failed with error:%v", err)
			return dbHost, dbPort, fmt.Errorf("Start SSH Tunnel failed with error:%v", err)
		}
		o.sshTunnel = sshTunnel
		dbHost = sshTunnel.LocalHost()
		dbPort = sshTunnel.LocalPort()
	}
	return dbHost, dbPort, nil
}

func (o *DBConnectionPoolOptions) cleanSSHTunnel() {
	if nil != o.sshTunnel {
		o.sshTunnel.Stop()
		o.sshTunnel = nil
	}
}

// IsEngineTypeValid check engine type range
func (o *DBConnectionPoolOptions) IsEngineTypeValid() bool {
	if _EngineTypeMin > o.Engine || _EngineTypeMax < o.Engine {
		return false
	}
	return true
}
