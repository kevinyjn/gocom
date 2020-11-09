package unittests

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

func TestDecryptTexts(t *testing.T) {
	testDecryptAES(t)
	fmt.Println("Testing decrypt texts finished")
}

func testDecryptAES(t *testing.T) {
	txt := "eNqdVttv21QY/1/8iErk4ziO07euHWODXVj2QIdI5MtJ7MS3+Tgl9tI/Bg2JVrwwsXZ0W8suXcU2oFm29QkhJh6QJkDsiSGhie9cHLtjjLI+VN/9fN93fv6dnJciIzb8kTxqJrEbdKVZSUZyXVZlXZoRPjQ6afawlYys0K/4jksqUYyJVfFDG3uVU0bi4iA5GnRCafa8JNS2a9NSUCM3BIaPwXT/t2ub28/BPtcFtarMSMfDIHGYJs9ITTyEoO3x+KG0nJ+vlM93AzKIjcDC4vh5I7bzsxdT298/wKLpmWDRNJCPZ1lqZqKrpmNmDshIhj9VqddrYDxq9akNpNPp0C83C7rJ+gJ5nsTn6Gm2kUDP6gwIqTRbm5GccBATNoXvBoMEC5mOxySCrTCwuTVx6TZqmq4oqsz+uC0LA3yy0yE4kWbfVHWwptiIpVldh3U0O6xnTVUgCzV0HSZVVQWp0NWRzP6I9358aLIxuZh5XFw0SYfOi1o7j75d+eLO9pOtB7eeMkfP5CGHgiy1eHuHgl4v63g9UCpM7XbskpalU+VInDkpjIJqekWBJTQtPyM9XrBpWWkuny12fxoHkdFFYtNMUcpKddq8b4nmPTtlF4kU0M5Y/Eh2/oJLIo9eAI07Bhmi1Kl4CU2lwiZqv0NeRMoxs9dlNkVTwSy3Js8urdwd3/n56uOLF3Yf3b05Wdv4/vqve+PxNkQv9D2KU0SHNPsM6yC+1afIYNM61jC/hLNDcyofS/MISKN3eaJebaiaqlbplA5roKHVanpr+/nu9uTZeMzsLJR2JSNVbTC46LVaXa2D9/2MZVWR3Nr7fOOTq39e3Jps3tihTdNBndR5ec3sFTUX+LZpw+T/rGXRE7fUWl9Z+2myuvZk48pklYFsSPdVfNPVgm9QVamhui5Xa7Up5agl99RYG73rkn9jISq3Tdfz2jZODBdQ/wGQEbMGoShzeIgtcEcJ/ZRhF7eejm+v3fz67mR1svZgvPvDxmf3tvbGk82HV9ae0E8qDgdRzlvf/TFZvXyfgYjWJNOiIipX3w5J5CaG17agr/0Qm/fcwLU4M+4f+nCQ4LjNCSUnFqQdkFkaL2MWpMlAanXtANSCFHk5b4Env6IFpPxnD2qj3ISuyDqD10GaOJpgP19cA2k6YmhkVnEN31y79/jLnYXqV7dvXMhdlmcQUr4iZiURpuxR7b6BlEt/0YtzDL/dCWOKZpp+7UYeOghcCom9T9lzZfXFVTJfYlAQUtp5b2AEiZukZW8UuxZtS9Uq2rSdkCSFQ62oNHkejKQUmL9i7ZhtWppMrv84Wd25TEMdI+7itlvaxb6MyEh9eFCJYNM5ywoH8L6+YP7HO8wKYABqgnOytbAbJRy6CsUJQopSchQYB8y3AfYDj64JaFcecbofMZnOJHOx+IcarfWP+UwtGjBK4gGm1BISg771rC5X2P5JfgFzNuALvnNYjBsGglhjfG6AA0vQ/MnYBqySgcl7n1pyjQMmnD43EYZ748WQqNZmS8Nxvi32UXc8I2ecU3ybuYktOoqmJ8wNbBd2G9iuZSRhLJKE1S7HCNxOdfplic+MPQN9N2jDx7S/GD1uIXaj9hL2QotjjtnCgekBAgpao3gxiOPC9OxcIOmG6JXg2AUeEntY/jCnUa3gVhHN7fXCXiJB7tML3wGIk+c0XiMHlX6HIg7JhqwVblS4129f+WWrlKkULq2UUX2dLkrPz+7e5u+Fo1Y4JEp58RJ85if4Fb/wLgknf5noy9PkBsjzceKEtkgjxhJmMeCwPPZTWgDIDZbCPj6TRjQMcGB40vLfQB4VvQ=="
	decodebytes, err := base64.StdEncoding.DecodeString(txt)
	if nil != err {
		t.Errorf("decode base64:%s failed with error:%v", txt, err)
		return
	}
	buf := bytes.Buffer{}
	buf.Write(decodebytes)
	r, err := zlib.NewReader(&buf)
	if nil != err {
		t.Errorf("new zib reader failed with error:%v", err)
		return
	}
	out := bytes.Buffer{}
	io.Copy(&out, r)

	var dat map[string]interface{}
	err = json.Unmarshal(out.Bytes(), &dat)
	if nil != err {
		t.Errorf("decode json:%s failed with error:%v", string(out.Bytes()), err)
		return
	}
	fmt.Printf("decoded buffer:%+v", dat)
}
