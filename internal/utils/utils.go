package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/crypto"
)

const (
	ClientID      = constants.ClientID
	ClientSecret  = constants.ClientSecret
	ClientVersion = constants.ClientVersion
	PackageName   = constants.PackageName
	SDKVersion    = constants.SDKVersion
	AppName       = PackageName
	APIHost       = constants.APIHost
	UserHost      = constants.UserHost
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

func BuildCustomUserAgent(deviceID string, userID string) string {
	deviceSign := GenerateDeviceSign(deviceID, PackageName)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ANDROID-%s/%s ", AppName, ClientVersion))
	sb.WriteString("protocolVersion/200 ")
	sb.WriteString("accesstype/ ")
	sb.WriteString(fmt.Sprintf("clientid/%s ", ClientID))
	sb.WriteString(fmt.Sprintf("clientversion/%s ", ClientVersion))
	sb.WriteString("action_type/ ")
	sb.WriteString("networktype/WIFI ")
	sb.WriteString("sessionid/ ")
	sb.WriteString(fmt.Sprintf("deviceid/%s ", deviceID))
	sb.WriteString("providername/NONE ")
	sb.WriteString(fmt.Sprintf("devicesign/%s ", deviceSign))
	sb.WriteString("refresh_token/ ")
	sb.WriteString(fmt.Sprintf("sdkversion/%s ", SDKVersion))
	sb.WriteString(fmt.Sprintf("datetime/%d ", GetTimestamp()))
	sb.WriteString(fmt.Sprintf("usrno/%s ", userID))
	sb.WriteString(fmt.Sprintf("appname/%s ", AppName))
	sb.WriteString("session_origin/ ")
	sb.WriteString("grant_type/ ")
	sb.WriteString("appid/ ")
	sb.WriteString("clientip/ ")
	sb.WriteString("devicename/Xiaomi_M2004j7ac ")
	sb.WriteString("osversion/13 ")
	sb.WriteString("platformversion/10 ")
	sb.WriteString("accessmode/ ")
	sb.WriteString("devicemodel/M2004J7AC")

	return sb.String()
}
