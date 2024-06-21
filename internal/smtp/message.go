package smtp

import "time"

// ReceivedMessage contains a single email message as received via SMTP.
type ReceivedMessage struct {
	smtpFrom       string
	smtpRcptTo     []string
	rawMessageData []byte
	receivedAt     time.Time
}

func NewReceivedMessage(
	from string, rcptTo []string, rawMessageData []byte, receivedAt time.Time,
) (*ReceivedMessage, error) {
	msg := &ReceivedMessage{
		smtpFrom:       from,
		smtpRcptTo:     rcptTo,
		rawMessageData: rawMessageData,
		receivedAt:     receivedAt,
	}

	// TODO: Parse message

	return msg, nil
}

// TODO: Getters
