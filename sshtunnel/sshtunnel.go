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

	"golang.org/x/crypto/ssh"
)

// TunnelForwarder ssh tunnel
type TunnelForwarder struct {
	Host       string
	Port       int
	User       string
	Password   string
	RemoteHost string
	RemotePort int
	localHost  string
	localPort  int
	client     *ssh.Client
	sshConfig  *ssh.ClientConfig
	local      net.Listener
	remote     net.Conn
}

var (
	localListenPort = 13110
)

// PrivateKeyPath getter
func (c *TunnelForwarder) PrivateKeyPath() string {
	return os.Getenv("HOME") + "/.ssh/id_rsa"
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
		return nil, err
	}
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

// Start start the ssh tunnel
func (c *TunnelForwarder) Start() error {
	c.localHost = "127.0.0.1"
	c.localPort = 0
	var err error
	c.sshConfig, err = c.InitUserAuth(c.User, c.Password)
	if nil != err {
		logger.Error.Printf("Initialize ssh tunnel %s:%d user auth failed with error:%v", c.Host, c.Port, err)
		return err
	}

	chkCount := 0
	for nil == c.local && chkCount < 1000 {
		localListenPort++
		chkCount++
		c.local, err = net.Listen("tcp", fmt.Sprintf("%s:%d", c.localHost, localListenPort))
		if nil == err {
			c.localPort = localListenPort
			break
		}
	}
	if nil == c.local {
		logger.Error.Printf("Listening ssh tunnel local server failed with binding local port execeeds max failing count")
		return errors.New("listening ssh tunnel local server failed with binding local port execeeds max failing count")
	}

	c.client, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.sshConfig)
	if nil != err {
		logger.Error.Printf("Dialing ssh tunnel %s:%d failed with error:%v", c.Host, c.Port, err)
		c.local.Close()
		return err
	}
	// defer c.client.Close()

	c.remote, err = c.client.Dial("tcp", fmt.Sprintf("%s:%d", c.RemoteHost, c.RemotePort))
	if nil != err {
		logger.Error.Printf("Dialing ssh tunnel %s:%d remote connection %s:%d failed with error:%v", c.Host, c.Port, c.RemoteHost, c.RemotePort, err)
		c.local.Close()
		c.client.Close()
		return err
	}

	logger.Info.Printf("ssh tunnel %s:%d for %s:%d started", c.Host, c.Port, c.RemoteHost, c.RemotePort)
	go c.run()

	return nil
}

// Stop stop the runloop
func (c *TunnelForwarder) Stop() {
	if nil != c.remote {
		logger.Info.Printf("ssh tunnel %s:%d for %s:%d stopped", c.Host, c.Port, c.RemoteHost, c.RemotePort)
		c.remote.Close()
		c.remote = nil
	}
	if nil != c.client {
		c.client.Close()
		c.client = nil
	}
	if nil != c.local {
		c.local.Close()
		c.local = nil
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
		if nil == c.local {
			break
		}
		client, err = c.local.Accept()
		if nil != err {
			logger.Warning.Printf("SSH Tunnel %s:%d handling local client connection occured with error:%v", c.Host, c.Port, err)
			break
		} else {
			logger.Info.Printf("SSH Tunnel %s:%d handling a new local client %s", c.Host, c.Port, client.RemoteAddr().String())
		}

		c.handlerClient(client, c.remote)
	}
}

func (c *TunnelForwarder) handlerClient(client net.Conn, remote net.Conn) {
	defer client.Close()
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			logger.Warning.Printf("error while copy remote(%s)->local:", remote.RemoteAddr().String(), err)
		}
		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			logger.Warning.Printf("error while copy local->remote(%s):", remote.RemoteAddr().String(), err)
		}
		chDone <- true
	}()

	<-chDone
}
