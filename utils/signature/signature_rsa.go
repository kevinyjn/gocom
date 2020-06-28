package signature

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
)

// VerifySignatureDataRSA RSA
func VerifySignatureDataRSA(pubKey *rsa.PublicKey, hash crypto.Hash, params map[string]interface{}, sign string, configures ...SigningOption) error {
	signContent := FormatSignatureContent(params, configures...)
	signData, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}
	sha1i := sha1.New()
	sha1i.Write([]byte(signContent))
	err = rsa.VerifyPKCS1v15(pubKey, hash, sha1i.Sum(nil), signData)
	return err
}

// VerifySignatureDataRSAEx RSA
func VerifySignatureDataRSAEx(pubKey *rsa.PublicKey, hash crypto.Hash, content string, sign []byte) error {
	sha1i := sha1.New()
	sha1i.Write([]byte(content))
	err := rsa.VerifyPKCS1v15(pubKey, hash, sha1i.Sum(nil), sign)
	return err
}
