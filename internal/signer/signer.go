package signer

import (
	"fmt"
	"time"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/crypto"
)

const (
	ClientID      = constants.ClientID
	ClientVersion = constants.ClientVersion
	PackageName   = constants.PackageName
)

var salts = []string{
	"Gez0T9ijiI9WCeTsKSg3SMlx",
	"zQdbalsolyb1R/",
	"ftOjr52zt51JD68C3s",
	"yeOBMH0JkbQdEFNNwQ0RI9T3wU/v",
	"BRJrQZiTQ65WtMvwO",
	"je8fqxKPdQVJiy1DM6Bc9Nb1",
	"niV",
	"9hFCW2R1",
	"sHKHpe2i96",
	"p7c5E6AcXQ/IJUuAEC9W6",
	"",
	"aRv9hjc9P+Pbn+u3krN6",
	"BzStcgE8qVdqjEH16l4",
	"SqgeZvL5j9zoHP95xWHt",
	"zVof5yaJkPe3VFpadPof",
}

func GetTimestamp() int64 {
	return time.Now().UnixMilli()
}

func CaptchaSign(deviceID string, timestamp string) string {
	sign := ClientID + ClientVersion + PackageName + deviceID + timestamp
	for _, salt := range salts {
		sign = crypto.MD5Hash(sign + salt)
	}
	return fmt.Sprintf("1.%s", sign)
}

func GenerateDeviceSign(deviceID string, packageName string) string {
	signatureBase := deviceID + packageName + "1appkey"

	sha1Hash := crypto.SHA1Hash(signatureBase)
	md5Result := crypto.MD5Hash(sha1Hash)

	return fmt.Sprintf("div101.%s%s", deviceID, md5Result)
}
