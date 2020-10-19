package utils

import (
	"bytes"
	"io/ioutil"

	"github.com/kevinyjn/gocom/logger"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// GbkToUtf8 converts gbk encoding text into utf-8
func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := ioutil.ReadAll(reader)
	if nil != err {
		logger.Error.Printf("Converts gbk encoding text:%s into utf-8 encoding while read bytes failed with error:%v", string(s), err)
		return nil, err
	}
	return d, nil
}

// Utf8ToGbk converts utf-8 encoding text into gbk
func Utf8ToGbk(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, err := ioutil.ReadAll(reader)
	if nil != err {
		logger.Error.Printf("Converts utf-8 encoding text:%s into gbk encoding while read bytes failed with error:%v", string(s), err)
		return nil, err
	}
	return d, nil
}

// GB2312ToUtf8 converts gbk encoding text into utf-8
func GB2312ToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.HZGB2312.NewDecoder())
	d, err := ioutil.ReadAll(reader)
	if nil != err {
		logger.Error.Printf("Converts gb2312 encoding text:%s into utf-8 encoding while read bytes failed with error:%v", string(s), err)
		return nil, err
	}
	return d, nil
}

// GB18030ToUtf8 converts gbk encoding text into utf-8
func GB18030ToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GB18030.NewDecoder())
	d, err := ioutil.ReadAll(reader)
	if nil != err {
		logger.Error.Printf("Converts gb18030 encoding text:%s into utf-8 encoding while read bytes failed with error:%v", string(s), err)
		return nil, err
	}
	return d, nil
}
