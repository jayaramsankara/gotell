package apns

import "github.com/anachronistic/apns"

var apnsClient * apns.Client

type ApnsMessage struct {
	Message string `json:"message"`
	Badge int `json:"badge"`
	Sound string `json:"sound"`
}

func init() {
	apnsClient = apns.NewClient("gateway.sandbox.push.apple.com:2195", "YOUR_CERT_PEM", "YOUR_KEY_NOENC_PEM")
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
