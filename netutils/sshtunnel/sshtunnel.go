package sshtunnel

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/kevinyjn/gocom/logger"
	"github.com/kevinyjn/gocom/utils"

	"golang.org/x/crypto/ssh"
)

// TunnelForwarder ssh tunnel
type TunnelForwarder struct {
	Host        string `property:"Host"`
	Port        int    `property:"Port"`
	User        string `property:"User"`
	Password    string `property:"Password,category:password"`
	RemoteHost  string `property:"Remote Host"`
	RemotePort  int    `property:"Remote Port"`
	localHost   string
	localPort   int
	sshConfig   *ssh.ClientConfig
	localServer net.Listener
}

var (
	localListenPort = 13110
)

// NewSSHTunnel TunnelForwarder
func NewSSHTunnel(dsn string, remoteHost string, remotePort int) (*TunnelForwarder, error) {
	sshTunnel := &TunnelForwarder{
		localHost:  "127.0.0.1",
		localPort:  0,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
	}
	if "" != dsn {
		err := sshTunnel.ParseFromDSN(dsn)
		if nil != err {
			logger.Error.Printf("SSHTunnel parse from ssh dsn:%s failed with error:%v", dsn, err)
			return sshTunnel, err
		}
	}
	return sshTunnel, nil
}

// PrivateKeyPath getter
func (c *TunnelForwarder) PrivateKeyPath() string {
	homeDir, err := os.UserHomeDir()
	if nil != err {
		logger.Error.Printf("SSHTunnel get user home directory failed with error:%v", err)
		homeDir = os.Getenv("HOME")
	}
	sshDir := homeDir + "/.ssh"
	utils.EnsureDirectory(sshDir)
	return sshDir + "/id_rsa"
}

// ParsePrivateKey parse private key
func (c *TunnelForwarder) ParsePrivateKey(keyPath string) (ssh.Signer, error) {
	buff, err := ioutil.ReadFile(keyPath)
	if nil != err {
		return nil, err
	}
	return ssh.ParsePrivateKey(buff)
}

// InitUserAuth init with user and password
func (c *TunnelForwarder) InitUserAuth(user, password string) (*ssh.ClientConfig, error) {
	key, err := c.ParsePrivateKey(c.PrivateKeyPath())
	if nil != err {
		// return nil, err
		c.sshConfig = &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		c.sshConfig = &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(key),
				ssh.Password(password),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}
	}

	return c.sshConfig, nil
}

// LocalHost get local tunnel host
func (c *TunnelForwarder) LocalHost() string {
	return c.localHost
}

// LocalPort get local tunnel listen port
func (c *TunnelForwarder) LocalPort() int {
	return c.localPort
}

func (c *TunnelForwarder) ensureLocalServer() (net.Listener, error) {
	if "" == c.localHost {
		c.localHost = "127.0.0.1"
	}
	var err error
	c.sshConfig, err = c.InitUserAuth(c.User, c.Password)
	if nil != err {
		logger.Error.Printf("ensure ssh tunnel %s:%d local server user auth failed with error:%v", c.Host, c.Port, err)
		return nil, err
	}

	chkCount := 0
	c.localPort = 0
	for nil == c.localServer && chkCount < 1000 {
		localListenPort++
		chkCount++
		c.localServer, err = net.Listen("tcp", fmt.Sprintf("%s:%d", c.localHost, localListenPort))
		if nil == err {
			c.localPort = localListenPort
			break
		}
	}
	if nil == c.localServer {
		return c.localServer, fmt.Errorf("ensure ssh tunnel %s:%d local server failed with max try listening count", c.Host, c.Port)
	}
	return c.localServer, nil
}

// Start start the ssh tunnel
func (c *TunnelForwarder) Start() error {
	_, err := c.ensureLocalServer()
	if nil != err {
		logger.Error.Printf("Initialize ssh tunnel %s:%d local server failed with error:%v", c.Host, c.Port, err)
		return err
	}
	if nil == c.localServer {
		logger.Error.Printf("Listening ssh tunnel local server failed with binding local port execeeds max failing count")
		return errors.New("listening ssh tunnel local server failed with binding local port execeeds max failing count")
	}

	logger.Info.Printf("ssh tunnel local server %s:%d for ssh tunnel %s:%d for client %s:%d started", c.localHost, c.localPort, c.Host, c.Port, c.RemoteHost, c.RemotePort)
	go c.run()

	return nil
}

// Stop stop the runloop
func (c *TunnelForwarder) Stop() {
	if nil != c.localServer {
		c.localServer.Close()
		c.localServer = nil
	}
}

func urlDecode(val string) string {
	s, err := url.QueryUnescape(val)
	if nil != err {
		return val
	}
	return s
}

// ParseFromDSN parse
func (c *TunnelForwarder) ParseFromDSN(DSN string) error {
	if "" == DSN {
		return errors.New("Empty dsn")
	}
	slices := strings.Split(DSN, "@")
	if len(slices) > 1 {
		authSlices := strings.Split(slices[0], "//")
		if len(authSlices) > 1 {
			authSlices = []string{authSlices[1]}
		}
		authSlices = strings.Split(authSlices[0], ":")
		c.User = urlDecode(authSlices[0])
		if len(authSlices) > 1 {
			c.Password = urlDecode(authSlices[1])
		}
		slices = []string{slices[1]}
	}
	hostSlices := strings.Split(slices[0], ":")
	c.Host = hostSlices[0]
	if len(hostSlices) > 1 {
		port, err := strconv.Atoi(hostSlices[1])
		if nil != err {
			logger.Error.Printf("Parsing SSH Tunnel DSN port part failed with error:%v", err)
			return fmt.Errorf("Parsing SSH Tunnel DSN port part failed with error:%v", err)
		}
		c.Port = port
	} else {
		c.Port = 22
	}
	return nil
}

func (c *TunnelForwarder) run() {
	var client net.Conn
	var err error
	for {
		if nil == c.localServer {
			break
		}
		client, err = c.localServer.Accept()
		if nil != err {
			logger.Warning.Printf("SSH Tunnel %s:%d handling local client connection occured with error:%v", c.Host, c.Port, err)
			break
		} else {
			logger.Info.Printf("SSH Tunnel %s:%d handling a new local client %s", c.Host, c.Port, client.RemoteAddr().String())
		}

		go c.forward(client)
		// c.handlerClient(client, c.remote)
	}
}

func (c *TunnelForwarder) forward(localConn net.Conn) error {
	// Establish connection to the intermediate server
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.sshConfig)
	if nil != err {
		logger.Error.Printf("Dialing ssh tunnel %s:%d failed with error:%v", c.Host, c.Port, err)
		return err
	}
	// defer c.sshClient.Close()

	remote, err := sshClient.Dial("tcp", fmt.Sprintf("%s:%d", c.RemoteHost, c.RemotePort))
	if nil != err {
		logger.Error.Printf("Dialing ssh tunnel %s:%d remote connection %s:%d failed with error:%v", c.Host, c.Port, c.RemoteHost, c.RemotePort, err)
		sshClient.Close()
		return err
	}

	go copyConnectionStream(localConn, remote)
	go copyConnectionStream(remote, localConn)
	return nil
}

// Transfer the data between  and the remote server
func copyConnectionStream(writer, reader net.Conn) {
	_, err := io.Copy(writer, reader)
	if err != nil {
		logger.Warning.Printf("error while copy %s->%s failed:%v", reader.RemoteAddr().String(), writer.RemoteAddr().String(), err)
	}
}
