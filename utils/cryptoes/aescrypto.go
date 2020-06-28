package cryptoes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"strconv"
)

// AESEncryptoMode aes encrypto mode
type AESEncryptoMode int

// Constants
const (
	AESEncryptoModeECB = AESEncryptoMode(1)
	AESEncryptoModeCBC = AESEncryptoMode(2)
)

// AESCryptoModeError error
type AESCryptoModeError int

func (k AESCryptoModeError) Error() string {
	return "AES crypto failed with invalid mode: " + strconv.Itoa(int(k))
}

// PKCS7Padding do pkcs7 padding for aes encrypt
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS7UnPadding do pkcs7 unpadding
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	if length <= unpadding {
		return []byte{}
	}
	return origData[:(length - unpadding)]
}

func getEncryptoMode(block cipher.Block, iv []byte, mode AESEncryptoMode) (cipher.BlockMode, error) {
	var blockMode cipher.BlockMode
	switch mode {
	case AESEncryptoModeECB:
		blockMode = NewECBEncrypter(block)
		break
	case AESEncryptoModeCBC:
		blockMode = cipher.NewCBCEncrypter(block, iv)
		break
	default:
		return nil, AESCryptoModeError(mode)
	}
	return blockMode, nil
}

func getDecryptoMode(block cipher.Block, iv []byte, mode AESEncryptoMode) (cipher.BlockMode, error) {
	var blockMode cipher.BlockMode
	switch mode {
	case AESEncryptoModeECB:
		blockMode = NewECBDecrypter(block)
		break
	case AESEncryptoModeCBC:
		blockMode = cipher.NewCBCDecrypter(block, iv)
		break
	default:
		return nil, AESCryptoModeError(mode)
	}
	return blockMode, nil
}

// AesEncrypt do aes encrypt
func AesEncrypt(origData, key []byte, iv string, mode AESEncryptoMode) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	ivBytes := []byte(iv)
	if iv == "" {
		ivBytes = key[:blockSize]
	}
	blockMode, err := getEncryptoMode(block, ivBytes, mode)
	if err != nil {
		return nil, err
	}
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

// AesDecrypt do aes decrypt
func AesDecrypt(crypted, key []byte, iv string, mode AESEncryptoMode) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	ivBytes := []byte(iv)
	if iv == "" {
		ivBytes = key[:blockSize]
	}
	blockMode, err := getDecryptoMode(block, ivBytes, mode)
	if err != nil {
		return nil, err
	}
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

// NewECBEncrypter returns a BlockMode which encrypts in electronic code book
// mode, using the given Block.
func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}
func (x *ecbEncrypter) BlockSize() int { return x.blockSize }
func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

// NewECBDecrypter returns a BlockMode which decrypts in electronic code book
// mode, using the given Block.
func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}
func (x *ecbDecrypter) BlockSize() int { return x.blockSize }
func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}
