package unittests

import (
	"bytes"
	"testing"

	"github.com/kevinyjn/gocom/httpclient"
)

func TestHTTPQueryWithRetry(t *testing.T) {
	url := "http://127.0.0.1:3000/invalidpath"
	body := []byte("Testing Content")
	resp, err := httpclient.HTTPQuery("POST", url, bytes.NewReader(body), httpclient.WithHTTPHeader("AppId", "a01"), httpclient.WithRetry(2))
	AssertNotNil(t, err, "httpclient.HTTPQuery")
	AssertEquals(t, 0, len(resp), "httpclient.HTTPQuery response")
}
