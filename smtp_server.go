package main

import (
	"context"

	esmtp "github.com/emersion/go-smtp"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/programmfabrik/apitest/internal/smtp"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

// startSmtpServer starts the testing SMTP server, if configured.
func (ats *Suite) startSmtpServer() {
	if ats.SmtpServer == nil || ats.smtpServer != nil {
		return
	}

	ats.smtpServer = smtp.NewServer(ats.SmtpServer.Addr, ats.SmtpServer.MaxMessageSize)

	go func() {
		if !ats.config.logShort {
			logrus.Infof("Starting SMTP Server: %s", ats.SmtpServer.Addr)
		}

		err := ats.smtpServer.ListenAndServe()
		if err != nil && !errors.Is(err, esmtp.ErrServerClosed) {
			// Error starting or closing listener:
			logrus.Fatalf("SMTP server ListenAndServe: %s", err.Error())
		}
	}()

	util.WaitForTCP(ats.SmtpServer.Addr)
}

// stopSmtpServer stops the SMTP server that was started using StartSMTPServer.
func (ats *Suite) stopSmtpServer() {
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
	} else if !ats.config.logShort {
		logrus.Info("SMTP Server stopped")
	}

	ats.smtpServer = nil
}
