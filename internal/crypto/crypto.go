package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
)

func MD5Hash(input string) string {
	data := md5.Sum([]byte(input))
	return hex.EncodeToString(data[:])
}

func MD5HashBytes(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func SHA1Hash(input string) string {
	hash := sha1.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

func DoubleHash(input string) string {
	sha1Result := SHA1Hash(input)
	return MD5Hash(sha1Result)
}
