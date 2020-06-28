package signature

import (
	"crypto/sha1"
	"encoding/hex"
)

// GenerateSignatureDataSHA1 SHA1
func GenerateSignatureDataSHA1(appKey string, params map[string]interface{}, configures ...SigningOption) string {
	signContent := FormatSignatureContent(params, configures...)
	// log.Println(" == sign text:", signContent)
	sha1i := sha1.New()
	sha1i.Write([]byte(signContent))
	signValue := hex.EncodeToString(sha1i.Sum([]byte(appKey)))
	return signValue
}

// VerifySignatureDataSHA1 SHA1
func VerifySignatureDataSHA1(signValue string, appKey string, params map[string]interface{}, configures ...SigningOption) bool {
	// log.Println(" == sign text:", signContent)
	signValueGenerated := GenerateSignatureDataSHA1(appKey, params, configures...)
	return signValue == signValueGenerated
}
