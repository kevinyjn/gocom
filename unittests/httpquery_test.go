package unittests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/kevinyjn/gocom/definations"
	"github.com/kevinyjn/gocom/httpclient"
)

func TestHTTPQueryWithRetry(t *testing.T) {
	url := "http://127.0.0.1:3000/invalidpath"
	body := []byte("Testing Content")
	resp, err := httpclient.HTTPQuery("POST", url, bytes.NewReader(body), httpclient.WithHTTPHeader("AppId", "a01"), httpclient.WithRetry(2))
	AssertNotNil(t, err, "httpclient.HTTPQuery")
	AssertEquals(t, 0, len(resp), "httpclient.HTTPQuery response")
}

func TestHTTPQueryKubernetesAPI(t *testing.T) {
	url := "https://10.10.2.217:6444"
	api := "/api/v1/namespaces/dev/pods/a113-0.0.8-68f9fddff-gp9lb-noexists"
	caFile := "./tests/kubeapi/ca.crt"
	tokenFile := "./tests/kubeapi/token"
	token := ""
	tokenBytes, err := ioutil.ReadFile(tokenFile)
	if nil != err {
		fmt.Printf("read token:%s failed with error:%v", tokenFile, err)
	} else {
		token = string(tokenBytes)
	}
	tlsOption := definations.TLSOptions{
		Enabled: true,
		CaFile:  caFile,
	}
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	resp, err := httpclient.HTTPQuery("GET", url+api, nil, httpclient.WithHTTPTLSOptions(&tlsOption), httpclient.WithHTTPHeaders(headers))
	if nil != err {
		fmt.Printf("api:%s failed with error:%+v", api, err)
	}
	if nil != resp {
		fmt.Printf("api:%s response:%+v", api, string(resp))
	}
}
