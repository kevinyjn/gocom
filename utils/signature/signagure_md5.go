package signature

import (
	"crypto/md5"
	"encoding/hex"
)

// GenerateSignatureDataMd5 MD5
func GenerateSignatureDataMd5(appKey string, params map[string]interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContent(params, configures...) + appKey
	// log.Println(" == sign text:", signContent)
	h := md5.New()
	h.Write([]byte(signContent))
	signValue := hex.EncodeToString(h.Sum(nil))
	return signValue
}

// GenerateSignatureDataMd5Ex MD5
func GenerateSignatureDataMd5Ex(appKey string, payload interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContentEx(payload, configures...) + appKey
	// log.Println(" == sign text:", signContent)
	h := md5.New()
	h.Write([]byte(signContent))
	signValue := hex.EncodeToString(h.Sum(nil))
	return signValue
}

// VerifySignatureDataMd5 MD5
func VerifySignatureDataMd5(signValue string, appKey string, params map[string]interface{}, configures ...SigningOption) bool {
	// log.Println(" == sign text:", signContent)
	signValueGenerated := GenerateSignatureDataMd5(appKey, params, configures...)
	return signValue == signValueGenerated
}
