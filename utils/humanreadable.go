package utils

import (
	"bytes"
	"fmt"

	"github.com/kevinyjn/gocom/logger"
)

// Constants
const (
	SizeGB   = 1024 * 1024 * 1024
	SizeMB   = 1024 * 1024
	SizeKB   = 1024
	DecadeKB = 10240
)

// HumanByteSize convert byte size as human recognizable
func HumanByteSize(bs int) string {
	var name string
	var n float64
	if bs > SizeGB {
		name = "GB"
		n = float64(bs) / SizeGB
	} else if bs > SizeMB {
		name = "MB"
		n = float64(bs) / SizeMB
	} else if bs > DecadeKB {
		name = "KB"
		n = float64(bs) / SizeKB
	} else {
		return fmt.Sprintf("%d Bytes", bs)
	}
	return fmt.Sprintf("%.2f %s", n, name)
}

// HumanByteText convert byte data as human readable text, not like %q contains quatations
func HumanByteText(bs []byte) string {
	var l = len(bs)
	buf := bytes.Buffer{}
	buf.Grow(l)
	var err error
	belongsToUnicode := 0
	for i, c := range bs {
		if c < 0x20 {
			if c > 0x06 && c < 0x0e {
				_, err = buf.WriteString(fmt.Sprintf("%q", c)[1:3])
			} else {
				_, err = buf.WriteString(fmt.Sprintf("\\x%02x", c))
			}
		} else if c == 0x7f {
			_, err = buf.WriteString(fmt.Sprintf("\\u%04x", c))
		} else if c > 0x7f {
			if (c & 0xf8) == 0xf0 {
				if i < l-3 && ((bs[i+3])&0xc0) == 0x80 && ((bs[i+2])&0xc0) == 0x80 && ((bs[i+1])&0xc0) == 0x80 {
					belongsToUnicode = 4
				}
			} else if (c & 0xf0) == 0xe0 {
				if i < l-2 && ((bs[i+2])&0xc0) == 0x80 && ((bs[i+1])&0xc0) == 0x80 {
					belongsToUnicode = 3
				}
			} else if (c & 0xe0) == 0xc0 {
				if i < l-1 && ((bs[i+1])&0xc0) == 0x80 {
					belongsToUnicode = 2
				}
			}
			if belongsToUnicode > 0 {
				err = buf.WriteByte(c)
				belongsToUnicode--
			} else {
				_, err = buf.WriteString(fmt.Sprintf("\\x%02x", c))
			}
		} else {
			err = buf.WriteByte(c)
		}
		if nil != err {
			logger.Fatal.Printf("format bytes as humanlize text:%q failed with error:%v", bs, err)
			break
		}
	}
	return string(buf.Bytes())
}
