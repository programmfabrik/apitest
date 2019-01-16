package test_utils

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var TestServer = NewTestServer(Routes{
	"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
		(*w).Write([]byte("{\"token\": \"mock\"}"))
	},
	"/api/v1/session/authenticate": func(w *http.ResponseWriter, r *http.Request) {
		(*w).Write([]byte("{\"authenticated\": \"true\"}"))
	},
	"/api/v1/settings/purge": func(w *http.ResponseWriter, r *http.Request) {
		(*w).WriteHeader(500)
	},
	"/api/v1/mock": func(w *http.ResponseWriter, r *http.Request) {
		(*w).Write([]byte("{\"mocked\": \"true\"}"))
	},
})

var TestClient = TestServer.Client()

type LoggingMessage struct {
	Level string
	Msg   string
}

type LoggingRegexAssertion struct {
	Level    string
	MsgRegex *regexp.Regexp
}

type LoggingRegexAssertions []LoggingRegexAssertion

func (assertion LoggingRegexAssertion) String() string {
	return fmt.Sprintf("Level: %s MsgRegex: %s", assertion.Level, assertion.MsgRegex.String())
}

func (assertions LoggingRegexAssertions) String() string {
	content := ""
	for i := range assertions {
		content += assertions[i].String()
		content += "\n"
	}
	return content
}

func getMessagesFromLogBuffer(log bytes.Buffer) (res []LoggingMessage) {
	logString := log.String()
	logLines := strings.Split(logString, "\n")
	for _, line := range logLines[:len(logLines)-1] {
		replaceRegex := regexp.MustCompile("elapsed=.*$")
		line = replaceRegex.ReplaceAllString(line, "")

		logRegex := regexp.MustCompile("level=(.*) msg=(.*)$")

		match := logRegex.FindStringSubmatch(line)

		if len(match) > 0 {
			res = append(res, LoggingMessage{Level: match[1], Msg: match[2]})
		}
	}
	return res
}

func assertLoggingMessageEqualsRegex(logMsg LoggingMessage, ass LoggingRegexAssertion) bool {
	return (logMsg.Level == ass.Level) && ass.MsgRegex.Match([]byte(logMsg.Msg))
}

func AssertLoggingEqualsRegex(log bytes.Buffer, want []LoggingRegexAssertion) (bool, []string) {
	success := true
	notMatched := make([]string, 0)

	logMessages := getMessagesFromLogBuffer(log)
	if len(logMessages) != len(want) {
		return false, []string{fmt.Sprintf("Len: Exp %d != %d Got", len(want), len(logMessages))}
	}

	for i := range logMessages {
		equal := assertLoggingMessageEqualsRegex(logMessages[i], want[i])

		if !equal {
			notMatched = append(
				notMatched,
				fmt.Sprintf(
					"[%s] '%s' != [%s] '%s'",
					logMessages[i].Level, logMessages[i].Msg, want[i].Level, want[i].MsgRegex))
		}

		success = success && equal
	}

	return success, notMatched
}
