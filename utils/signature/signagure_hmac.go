package signature

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
)

// GenerateSignatureDataHmacSHA1 SHA1
func GenerateSignatureDataHmacSHA1(appKey string, params map[string]interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContent(params, configures...)
	// fmt.Println(" == sign text:", signContent)
	h := hmac.New(sha1.New, []byte(appKey))
	h.Write([]byte(signContent))
	signValue := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signValue
}

// GenerateSignatureDataHmacSHA1Ex SHA1
func GenerateSignatureDataHmacSHA1Ex(appKey string, payload interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContentEx(payload, configures...)
	// fmt.Println(" == sign text:", signContent)
	h := hmac.New(sha1.New, []byte(appKey))
	h.Write([]byte(signContent))
	signValue := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signValue
}

// VerifySignatureDataHmacSHA1 SHA1
func VerifySignatureDataHmacSHA1(signValue string, appKey string, params map[string]interface{}, configures ...SigningOption) bool {
	// log.Println(" == sign text:", signContent)
	signValueGenerated := GenerateSignatureDataHmacSHA1(appKey, params, configures...)
	return signValue == signValueGenerated
}

// GenerateSignatureDataHmacMD5 MD5
func GenerateSignatureDataHmacMD5(appKey string, params map[string]interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContent(params, configures...)
	// log.Println(" == sign text:", signContent)
	h := hmac.New(md5.New, []byte(appKey))
	h.Write([]byte(signContent))
	signValue := base64.StdEncoding.EncodeToString(h.Sum([]byte("")))
	return signValue
}

// GenerateSignatureDataHmacMD5Ex MD5
func GenerateSignatureDataHmacMD5Ex(appKey string, payload interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContentEx(payload, configures...)
	// log.Println(" == sign text:", signContent)
	h := hmac.New(md5.New, []byte(appKey))
	h.Write([]byte(signContent))
	signValue := base64.StdEncoding.EncodeToString(h.Sum([]byte("")))
	return signValue
}

// VerifySignatureDataHmacMD5 MD5
func VerifySignatureDataHmacMD5(signValue string, appKey string, params map[string]interface{}, configures ...SigningOption) bool {
	// log.Println(" == sign text:", signContent)
	signValueGenerated := GenerateSignatureDataHmacMD5(appKey, params, configures...)
	return signValue == signValueGenerated
}
