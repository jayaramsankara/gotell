package apns

import (
	"github.com/anachronistic/apns"
	"os"
)

var apnsClient * apns.Client

type ApnsMessage struct {
	Message string `json:"message"`
	Badge int `json:"badge"`
	Sound string `json:"sound"`
}

func init() {
	var apnsCertFile = os.Getenv("APNS_CERT")
	var apnsKeyfile = os.Getenv("APNS_KEY")
	apnsClient = apns.NewClient("gateway.sandbox.push.apple.com:2195", apnsCertFile, apnsKeyfile)
}
func Notify(message * ApnsMessage, deviceToken string) {
	payload := apns.NewPayload()
	payload.Alert = message.Message
	payload.Badge = message.Badge
	payload.Sound = message.Sound

	pn := apns.NewPushNotification()
	pn.DeviceToken = deviceToken
	pn.AddPayload(payload)
	
	apnsClient.Send(pn)
}
