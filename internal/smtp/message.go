package smtp

import "time"

// ReceivedMessage contains a single email message as received via SMTP.
type ReceivedMessage struct {
	smtpFrom       string
	smtpRcptTo     []string
	rawMessageData []byte
	receivedAt     time.Time
}

// TODO: Constructor that takes in from, rcptTo, rawMessageData, receivedAt and that also parses the message

// TODO: Getters
