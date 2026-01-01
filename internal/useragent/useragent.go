package useragent

import (
	"fmt"
	"strings"

	"github.com/zhz8888/pikpakapi-go/internal/constants"
	"github.com/zhz8888/pikpakapi-go/internal/signer"
)

func BuildCustomUserAgent(deviceID string, userID string) string {
	deviceSign := signer.GenerateDeviceSign(deviceID, constants.PackageName)
	timestamp := signer.GetTimestamp()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ANDROID-%s/%s ", constants.PackageName, signer.ClientVersion))
	sb.WriteString("protocolVersion/200 ")
	sb.WriteString("accesstype/ ")
	sb.WriteString(fmt.Sprintf("clientid/%s ", signer.ClientID))
	sb.WriteString(fmt.Sprintf("clientversion/%s ", signer.ClientVersion))
	sb.WriteString("action_type/ ")
	sb.WriteString("networktype/WIFI ")
	sb.WriteString("sessionid/ ")
	sb.WriteString(fmt.Sprintf("deviceid/%s ", deviceID))
	sb.WriteString("providername/NONE ")
	sb.WriteString(fmt.Sprintf("devicesign/%s ", deviceSign))
	sb.WriteString("refresh_token/ ")
	sb.WriteString(fmt.Sprintf("sdkversion/%s ", constants.SDKVersion))
	sb.WriteString(fmt.Sprintf("datetime/%d ", timestamp))
	sb.WriteString(fmt.Sprintf("usrno/%s ", userID))
	sb.WriteString(fmt.Sprintf("appname/%s ", constants.PackageName))
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
