package cryptoes

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
)

// Base64Encode encode
func Base64Encode(val []byte) string {
	encodeContent := base64.StdEncoding.EncodeToString(val)
	return encodeContent
}

// Base64Decode decode
func Base64Decode(val string) ([]byte, error) {
	decodebytes, err := base64.StdEncoding.DecodeString(val)
	return decodebytes, err
}

// AESEncryptCBC encrypt
func AESEncryptCBC(origData []byte, key string, iv string) (string, error) {
	crypted, err := AesEncrypt(origData, []byte(key), iv, AESEncryptoModeCBC)
	if err != nil {
		return "", err
	}
	return Base64Encode(crypted), nil
}

// AESDecryptCBC decrypt
func AESDecryptCBC(crypted, key string, iv string) ([]byte, error) {
	cryptedBytes, err := Base64Decode(crypted)
	if err != nil {
		return nil, err
	}
	origData, err := AesDecrypt(cryptedBytes, []byte(key), iv, AESEncryptoModeCBC)
	if err != nil {
		return nil, err
	}
	return origData, nil
}

// AESEncryptECB encrypt
func AESEncryptECB(origData []byte, key string) (string, error) {
	crypted, err := AesEncrypt(origData, []byte(key), "", AESEncryptoModeECB)
	if err != nil {
		return "", err
	}
	return Base64Encode(crypted), nil
}

// AESDecryptECB aes decrypt
func AESDecryptECB(crypted, key string) ([]byte, error) {
	cryptedBytes, err := Base64Decode(crypted)
	if err != nil {
		return nil, err
	}
	origData, err := AesDecrypt(cryptedBytes, []byte(key), "", AESEncryptoModeECB)
	if err != nil {
		return nil, err
	}
	return origData, nil
}

// RSAEncryptNE rsa encrypt
func RSAEncryptNE(origData []byte, n *big.Int, e int) (string, error) {
	pubKey := &rsa.PublicKey{
		N: n,
		E: e,
	}
	encBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, origData)
	if err != nil {
		return "", err
	}
	return Base64Encode(encBytes), nil
}
