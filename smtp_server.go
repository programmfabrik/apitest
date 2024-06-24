package main

import (
	"context"

	esmtp "github.com/emersion/go-smtp"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/programmfabrik/apitest/internal/smtp"
)

// StartSmtpServer starts the testing SMTP server, if configured.
func (ats *Suite) StartSmtpServer() {
	if ats.SmtpServer == nil || ats.smtpServer != nil {
		return
	}

	ats.smtpServer = smtp.NewServer(ats.SmtpServer.Addr, ats.SmtpServer.MaxMessageSize)

	go func() {
		if !ats.Config.LogShort {
			logrus.Infof("Starting SMTP Server: %s", ats.SmtpServer.Addr)
		}

		err := ats.smtpServer.ListenAndServe()
		if !errors.Is(err, esmtp.ErrServerClosed) {
			// Error starting or closing listener:
			logrus.Fatal("SMTP server ListenAndServe:", err)
		}
	}()
}

// StopSmtpServer stops the SMTP server that was started using StartSMTPServer.
func (ats *Suite) StopSmtpServer() {
	if ats.SmtpServer == nil || ats.smtpServer == nil {
		return
	}

	// TODO: Shouldn't this use a context with a timeout (also at http_server.go)?
	err := ats.smtpServer.Shutdown(context.Background())
	if err != nil {
		// logrus.Error is used instead of Fatal, because an error
		// during closing of a server shouldn't affect the outcome of
		// the test.
		logrus.Error("SMTP Server shutdown:", err)
	} else if !ats.Config.LogShort {
		logrus.Info("SMTP Server stopped")
	}
}
