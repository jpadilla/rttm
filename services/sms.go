package services

import (
	"fmt"
	"os"

	"github.com/subosito/twilio"
)

var (
	tw               *twilio.Client
	twilioAccountSID = os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken  = os.Getenv("TWILIO_AUTH_TOKEN")
	twilioNumber     = os.Getenv("TWILIO_NUMBER")
)

func init() {
	tw = twilio.NewClient(twilioAccountSID, twilioAuthToken, nil)
}

func SendSMS(body string, phone string) {
	// Send Message
	params := twilio.MessageParams{
		Body: body,
	}

	twilioMessage, twilioResponse, err := tw.Messages.Send(twilioNumber, phone, params)

	fmt.Println(twilioMessage, twilioResponse, err)
}
