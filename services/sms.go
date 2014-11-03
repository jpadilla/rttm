package services

import (
	"fmt"
	"os"

	"github.com/subosito/twilio"
)

// SendSMS builds and sends SMS message via Twilio.
func SendSMS(phone string, body string) {
	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioNumber := os.Getenv("TWILIO_NUMBER")
	tw := twilio.NewClient(twilioAccountSID, twilioAuthToken, nil)

	params := twilio.MessageParams{
		Body: body,
	}

	twilioMessage, twilioResponse, err := tw.Messages.Send(twilioNumber, phone, params)

	fmt.Println(twilioMessage, twilioResponse, err)
}
