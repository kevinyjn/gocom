package unittests

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/kevinyjn/gocom/testingutil"
	"github.com/kevinyjn/gocom/utils"
)

func TestHumanReadableByteSize(t *testing.T) {
	testingutil.AssertEquals(t, "1.21 GB", utils.HumanByteSize(1024*1024*1024+210*1024*1024+1036), "GB size")
	testingutil.AssertEquals(t, "1.21 MB", utils.HumanByteSize(1024*1024+210*1024+1036), "MB size")
	testingutil.AssertEquals(t, "10.21 KB", utils.HumanByteSize(10*1024+220), "KB size")
	testingutil.AssertEquals(t, "10240 Bytes", utils.HumanByteSize(10240), "Byte size")
}

func TestHumanReadableBytesText(t *testing.T) {
	// length nearly to buffer capacity
	bs := bytes.Join([][]byte{[]byte{0, 1, 2}, []byte("中ABC国人民大会堂，人民英雄纪念碑"), []byte{3, 0x7f}, []byte("天安门")}, []byte{})
	testingutil.AssertEquals(t, "\\x00\\x01\\x02中ABC国人民大会堂，人民英雄纪念碑\\x03\\u007f天安门", utils.HumanByteText(bs), "ContainsEmptyBytes")
	// length would equals to buffer capacity
	testingutil.AssertEquals(t, "\\x00\\x01\\x02中ABC国人民大会堂，人民英雄纪念碑\\x03\\u007f与天安门",
		utils.HumanByteText(bytes.Join([][]byte{[]byte{0, 1, 2}, []byte("中ABC国人民大会堂，人民英雄纪念碑"), []byte{3, 0x7f}, []byte("与天安门")}, []byte{})), "ContainsEmptyBytes")
	testingutil.AssertEquals(t, "\\x00\\x01\\x02中ABC国人", utils.HumanByteText(bytes.Join([][]byte{[]byte{0, 1, 2}, []byte("中ABC国人")}, []byte{})), "ContainsEmptyBytes")
	testingutil.AssertEquals(t, "中ABC国人", utils.HumanByteText([]byte("中ABC国人")), "Chars")

	// testingutil.AssertEquals(t, "@\\xa9\\xf6\\xbc\\x9c\\xb80z\\x93\\x01{", utils.HumanByteText([]byte{64, 169, 246, 188, 156, 184, 48, 122, 147, 1, 123}), "Bytes1")

	buf := bytes.Buffer{}
	for i := 0; i < 256; i++ {
		buf.WriteByte(byte(i))
	}
	testingutil.AssertEquals(t,
		"\\x00\\x01\\x02\\x03\\x04\\x05\\x06\\a\\b\\t\\n\\v\\f\\r\\x0e\\x0f\\x10\\x11\\x12\\x13\\x14\\x15\\x16\\x17\\x18\\x19\\x1a\\x1b\\x1c\\x1d\\x1e\\x1f !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\\u007f\\x80\\x81\\x82\\x83\\x84\\x85\\x86\\x87\\x88\\x89\\x8a\\x8b\\x8c\\x8d\\x8e\\x8f\\x90\\x91\\x92\\x93\\x94\\x95\\x96\\x97\\x98\\x99\\x9a\\x9b\\x9c\\x9d\\x9e\\x9f\\xa0\\xa1\\xa2\\xa3\\xa4\\xa5\\xa6\\xa7\\xa8\\xa9\\xaa\\xab\\xac\\xad\\xae\\xaf\\xb0\\xb1\\xb2\\xb3\\xb4\\xb5\\xb6\\xb7\\xb8\\xb9\\xba\\xbb\\xbc\\xbd\\xbe\\xbf\\xc0\\xc1\\xc2\\xc3\\xc4\\xc5\\xc6\\xc7\\xc8\\xc9\\xca\\xcb\\xcc\\xcd\\xce\\xcf\\xd0\\xd1\\xd2\\xd3\\xd4\\xd5\\xd6\\xd7\\xd8\\xd9\\xda\\xdb\\xdc\\xdd\\xde\\xdf\\xe0\\xe1\\xe2\\xe3\\xe4\\xe5\\xe6\\xe7\\xe8\\xe9\\xea\\xeb\\xec\\xed\\xee\\xef\\xf0\\xf1\\xf2\\xf3\\xf4\\xf5\\xf6\\xf7\\xf8\\xf9\\xfa\\xfb\\xfc\\xfd\\xfe\\xff",
		utils.HumanByteText(buf.Bytes()), "Bytes")
	testingutil.AssertEquals(t,
		"\\x12\\x05utf-8\\x1a\\x1a2\"\\x010*$9a790c08-b5d7-4b5a-8ca5-5c6680b6202122/952362bf-295e-4ba1-ae0a-db287af28809@Ӑ\\xf3\\x99\\xb80z\\x93\\x01",
		utils.HumanByteText([]byte{
			18, 5, 117, 116, 102, 45, 56, 26, 26, 50, 34, 1, 48, 42, 36, 57, 97, 55, 57, 48, 99, 48, 56, 45, 98, 53, 100, 55, 45, 52, 98, 53, 97, 45, 56, 99, 97, 53, 45, 53, 99, 54, 54, 56, 48, 98, 54, 50, 48, 50, 49, 50, 50, 47, 57, 53, 50, 51, 54, 50, 98, 102, 45, 50, 57, 53, 101, 45, 52, 98, 97, 49, 45, 97, 101, 48, 97, 45, 100, 98, 50, 56, 55, 97, 102, 50, 56, 56, 48, 57, 64, 211, 144, 243, 153, 184, 48, 122, 147, 1,
		}), "Bytes")
	testingutil.AssertEquals(t,
		"\\x12\\x05utf-8\\x1a\\x1a2\"\\x010*$512918ab-5246-4a6d-a08f-13d71b8eaca222/952362bf-295e-4ba1-ae0a-db287af28809@Ĥ\\xfc\\x99\\xb80z\\x93\\x01",
		utils.HumanByteText([]byte{
			18, 5, 117, 116, 102, 45, 56, 26, 26, 50, 34, 1, 48, 42, 36, 53, 49, 50, 57, 49, 56, 97, 98, 45, 53, 50, 52, 54, 45, 52, 97, 54, 100, 45, 97, 48, 56, 102, 45, 49, 51, 100, 55, 49, 98, 56, 101, 97, 99, 97, 50, 50, 50, 47, 57, 53, 50, 51, 54, 50, 98, 102, 45, 50, 57, 53, 101, 45, 52, 98, 97, 49, 45, 97, 101, 48, 97, 45, 100, 98, 50, 56, 55, 97, 102, 50, 56, 56, 48, 57, 64, 196, 164, 252, 153, 184, 48, 122, 147, 1,
		}), "Bytes")

	// press test
	t1 := time.Now()
	for i := 0; i < 100000; i++ {
		utils.HumanByteText(bs)
	}
	t2 := time.Now()
	fmt.Printf("press test HumanByteText with 62bytes in 100000 cost %d microseconds\n", t2.Sub(t1).Microseconds())
}
