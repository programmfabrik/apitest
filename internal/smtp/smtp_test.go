package smtp

import (
	"context"
	_ "embed"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed smtp_testsession.txt
var smtpSession string

var testTime time.Time = time.Now()
var server *Server = runTestSession()

func TestMessageParsing(t *testing.T) {
	expectedMessages := buildExpectedMessages()

	require.Equal(t, len(server.receivedMessages), len(expectedMessages), "number of received messages")

	for i := range expectedMessages {
		assertMessageEqual(t, expectedMessages[i], server.receivedMessages[i])
	}
}

func TestMessageSearch(t *testing.T) {
	testCases := []struct {
		queries         []string
		expectedIndices []int
	}{
		{
			queries:         []string{``},
			expectedIndices: []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			queries: []string{
				`Content`,
				`Content-Type`,
				`^Content`,
				`^Content-Type`,
				`Content-Type:.*`,
				`^Content-Type:.*$`,
			},
			expectedIndices: []int{1, 2, 3, 4, 5, 8},
		},
		{
			queries: []string{
				`^Transfer`,
				`X-Funky-Header`,
			},
			expectedIndices: []int{},
		},
		{
			queries: []string{
				`Transfer`,
				`Content-Transfer-Encoding`,
				`^Content-Transfer`,
				`Content-Transfer-Encoding:.*`,
				`^Content-Transfer-Encoding:.*$`,
			},
			expectedIndices: []int{3, 4},
		},
		{
			queries: []string{
				`base64`,
				`Content-Transfer-Encoding: base64`,
				`^Content-Transfer-Encoding: base64$`,
			},
			expectedIndices: []int{3},
		},
		{
			queries: []string{
				`Subject: .*[äöüÄÖÜ]`,
				`^Subject: .*[äöüÄÖÜ]`,
				`Tästmail`,
			},
			expectedIndices: []int{6, 7},
		},
	}

	for i := range testCases {
		testCase := testCases[i]

		for j := range testCase.queries {
			query := testCase.queries[j]
			t.Run(query, func(t *testing.T) {
				re := regexp.MustCompile(query)
				actual := server.SearchByHeader(re)

				actualIndices := make([]int, len(actual))
				for ai, av := range actual {
					actualIndices[ai] = av.index
				}
				assert.ElementsMatch(t, testCase.expectedIndices, actualIndices)
			})
		}
	}
}

func TestMultipartSearch(t *testing.T) {
	// This test uses message #8 for all of its tests.

	testCases := []struct {
		queries         []string
		expectedIndices []int
	}{
		{
			queries: []string{
				"From",
				"Testmail",
			},
			expectedIndices: []int{},
		},
		{
			queries: []string{
				"X-Funky-Header",
				"Content-Transfer-Encoding",
			},
			expectedIndices: []int{0, 1},
		},
		{
			queries: []string{
				"X-Funky-Header: Käse",
				"X-Funky-Header: K[äöü]se",
				"^X-Funky-Header: Käse$",
				"Content-Transfer-Encoding: quoted-printable",
				"^Content-Transfer.* quoted",
				"quoted-printable",
			},
			expectedIndices: []int{1},
		},
		{
			queries: []string{
				"X-Funky-Header: Tästmail mit Ümlauten im Header",
				"X-Funky-Header: .*Ü",
				"Content-Transfer-Encoding: base64",
				"^Content-Transfer.* base64",
				"base64",
			},
			expectedIndices: []int{0},
		},
	}

	for i := range testCases {
		testCase := testCases[i]

		for j := range testCase.queries {
			query := testCase.queries[j]
			t.Run(query, func(t *testing.T) {
				re := regexp.MustCompile(query)

				msg, err := server.ReceivedMessage(8)
				require.NoError(t, err)

				actual := msg.SearchPartsByHeader(re)

				actualIndices := make([]int, len(actual))
				for ai, av := range actual {
					actualIndices[ai] = av.index
				}
				assert.ElementsMatch(t, testCase.expectedIndices, actualIndices)
			})
		}
	}
}

func assertHeadersEqual(t *testing.T, expected, actual map[string][]string) {
	assert.Equal(t, len(expected), len(actual))

	for k, v := range expected {
		if assert.Contains(t, actual, k) {
			assert.ElementsMatch(t, v, actual[k])
		}
	}
}

func assertMessageEqual(t *testing.T, expected, actual *ReceivedMessage) {
	assert.Equal(t, expected.index, actual.index)
	assert.Equal(t, expected.smtpFrom, actual.smtpFrom)
	assert.ElementsMatch(t, expected.smtpRcptTo, actual.smtpRcptTo)
	assert.Equal(t, expected.rawMessageData, actual.rawMessageData)
	assert.Equal(t, expected.receivedAt, actual.receivedAt)
	assert.Equal(t, expected.body, actual.body)
	assert.Equal(t, expected.contentType, actual.contentType)
	assert.Equal(t, expected.contentTypeParams, actual.contentTypeParams)
	assert.Equal(t, expected.isMultipart, actual.isMultipart)

	assertHeadersEqual(t, expected.headers, actual.headers)

	if assert.Equal(t, len(expected.multiparts), len(actual.multiparts)) {
		for i, m := range expected.multiparts {
			assertMultipartEqual(t, m, actual.multiparts[i])
		}
	}
}

func assertMultipartEqual(t *testing.T, expected, actual *ReceivedPart) {
	assert.Equal(t, expected.index, actual.index)
	assertHeadersEqual(t, expected.headers, actual.headers)
	assert.Equal(t, expected.body, actual.body)
}

// runTestSession starts a Server, runs a pre-recorded SMTP session,
// stops the Server and returns the Server struct.
func runTestSession() *Server {
	addr := ":9925"

	smtpSrc := strings.ReplaceAll(smtpSession, "\n", "\r\n")

	server := NewServer(addr, 0)
	server.clock = func() time.Time { return testTime }
	go server.ListenAndServe()
	defer server.Shutdown(context.Background())

	// give the server some time to open
	time.Sleep(time.Second)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(smtpSrc))
	if err != nil {
		panic(err)
	}

	// give the server some time to process
	time.Sleep(time.Second)

	return server
}

func buildExpectedMessages() []*ReceivedMessage {
	messages := []*ReceivedMessage{
		{
			index:      0,
			smtpFrom:   "testsender@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver@programmfabrik.de"},
			rawMessageData: []byte(`From: testsender@programmfabrik.de
To: testreceiver@programmfabrik.de

Hello World!
A simple plain text test mail.`),
			receivedAt: testTime,
			headers: map[string][]string{
				"From": {"testsender@programmfabrik.de"},
				"To":   {"testreceiver@programmfabrik.de"},
			},
			body: []byte(`Hello World!
A simple plain text test mail.`),
		},
		{
			index:      1,
			smtpFrom:   "testsender2@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver2@programmfabrik.de"},
			rawMessageData: []byte(`MIME-Version: 1.0
From: testsender2@programmfabrik.de
To: testreceiver2@programmfabrik.de
Date: Tue, 25 Jun 2024 11:15:57 +0200
Subject: Example Message
Content-type: multipart/mixed; boundary="d36c3118be4745f9a1cb4556d11fe92d"

Preamble is ignored.

--d36c3118be4745f9a1cb4556d11fe92d
Content-type: text/plain; charset=utf-8

Some plain text
--d36c3118be4745f9a1cb4556d11fe92d
Content-type: text/html; charset=utf-8

Some <b>text</b> <i>in</i> HTML format.
--d36c3118be4745f9a1cb4556d11fe92d--

Trailing text is ignored.`),
			receivedAt: testTime,
			headers: map[string][]string{
				"Mime-Version": {"1.0"},
				"From":         {"testsender2@programmfabrik.de"},
				"To":           {"testreceiver2@programmfabrik.de"},
				"Date":         {"Tue, 25 Jun 2024 11:15:57 +0200"},
				"Subject":      {"Example Message"},
				"Content-Type": {`multipart/mixed; boundary="d36c3118be4745f9a1cb4556d11fe92d"`},
			},
			body: []byte(`Preamble is ignored.

--d36c3118be4745f9a1cb4556d11fe92d
Content-type: text/plain; charset=utf-8

Some plain text
--d36c3118be4745f9a1cb4556d11fe92d
Content-type: text/html; charset=utf-8

Some <b>text</b> <i>in</i> HTML format.
--d36c3118be4745f9a1cb4556d11fe92d--

Trailing text is ignored.`),
			contentType: "multipart/mixed",
			contentTypeParams: map[string]string{
				"boundary": "d36c3118be4745f9a1cb4556d11fe92d",
			},
			isMultipart: true,
			multiparts: []*ReceivedPart{
				{
					index: 0,
					headers: map[string][]string{
						"Content-Type": {"text/plain; charset=utf-8"},
					},
					body: []byte(`Some plain text`),
				},
				{
					index: 1,
					headers: map[string][]string{
						"Content-Type": {"text/html; charset=utf-8"},
					},
					body: []byte(`Some <b>text</b> <i>in</i> HTML format.`),
				},
			},
		},
		{
			index:      2,
			smtpFrom:   "testsender3@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver3@programmfabrik.de"},
			rawMessageData: []byte(`From: testsender3@programmfabrik.de
To: testreceiver3@programmfabrik.de
Content-Type: text/plain; charset=utf-8

Noch eine Testmail. Diesmal mit nicht-ASCII-Zeichen: äöüß`),
			receivedAt: testTime,
			headers: map[string][]string{
				"From":         {"testsender3@programmfabrik.de"},
				"To":           {"testreceiver3@programmfabrik.de"},
				"Content-Type": {"text/plain; charset=utf-8"},
			},
			body:        []byte(`Noch eine Testmail. Diesmal mit nicht-ASCII-Zeichen: äöüß`),
			contentType: "text/plain",
			contentTypeParams: map[string]string{
				"charset": "utf-8",
			},
		},
		{
			index:      3,
			smtpFrom:   "testsender4@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver4@programmfabrik.de"},
			rawMessageData: []byte(`From: testsender4@programmfabrik.de
To: testreceiver4@programmfabrik.de
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64

RWluZSBiYXNlNjQtZW5rb2RpZXJ0ZSBUZXN0bWFpbCBtaXQgbmljaHQtQVNDSUktWmVpY2hlbjog
w6TDtsO8w58K`),
			receivedAt: testTime,
			headers: map[string][]string{
				"From":                      {"testsender4@programmfabrik.de"},
				"To":                        {"testreceiver4@programmfabrik.de"},
				"Content-Type":              {"text/plain; charset=utf-8"},
				"Content-Transfer-Encoding": {"base64"},
			},
			body: []byte(`Eine base64-enkodierte Testmail mit nicht-ASCII-Zeichen: äöüß
`),
			contentType: "text/plain",
			contentTypeParams: map[string]string{
				"charset": "utf-8",
			},
		},
		{
			index:      4,
			smtpFrom:   "testsender5@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver5@programmfabrik.de"},
			rawMessageData: []byte(`From: testsender5@programmfabrik.de
To: testreceiver5@programmfabrik.de
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

Noch eine Testmail mit =C3=A4=C3=B6=C3=BC=C3=9F, diesmal enkodiert in quote=
d-printable.`),
			receivedAt: testTime,
			headers: map[string][]string{
				"From":                      {"testsender5@programmfabrik.de"},
				"To":                        {"testreceiver5@programmfabrik.de"},
				"Content-Type":              {"text/plain; charset=utf-8"},
				"Content-Transfer-Encoding": {"quoted-printable"},
			},
			body:        []byte(`Noch eine Testmail mit äöüß, diesmal enkodiert in quoted-printable.`),
			contentType: "text/plain",
			contentTypeParams: map[string]string{
				"charset": "utf-8",
			},
		},
		{
			index:      5,
			smtpFrom:   "testsender6@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver6@programmfabrik.de"},
			rawMessageData: []byte(`MIME-Version: 1.0
From: testsender6@programmfabrik.de
To: testreceiver6@programmfabrik.de
Date: Tue, 25 Jun 2024 11:15:57 +0200
Subject: Example Message
Content-type: multipart/mixed; boundary="d36c3118be4745f9a1cb4556d11fe92d"

--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64

RWluZSBiYXNlNjQtZW5rb2RpZXJ0ZSBUZXN0bWFpbCBtaXQgbmljaHQtQVNDSUktWmVpY2hlbjog
w6TDtsO8w58K
--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

Noch eine Testmail mit =C3=A4=C3=B6=C3=BC=C3=9F, diesmal enkodiert in quote=
d-printable.
--d36c3118be4745f9a1cb4556d11fe92d--`),
			receivedAt: testTime,
			headers: map[string][]string{
				"Mime-Version": {"1.0"},
				"From":         {"testsender6@programmfabrik.de"},
				"To":           {"testreceiver6@programmfabrik.de"},
				"Date":         {"Tue, 25 Jun 2024 11:15:57 +0200"},
				"Subject":      {"Example Message"},
				"Content-Type": {`multipart/mixed; boundary="d36c3118be4745f9a1cb4556d11fe92d"`},
			},
			body: []byte(`--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64

RWluZSBiYXNlNjQtZW5rb2RpZXJ0ZSBUZXN0bWFpbCBtaXQgbmljaHQtQVNDSUktWmVpY2hlbjog
w6TDtsO8w58K
--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

Noch eine Testmail mit =C3=A4=C3=B6=C3=BC=C3=9F, diesmal enkodiert in quote=
d-printable.
--d36c3118be4745f9a1cb4556d11fe92d--`),
			contentType: "multipart/mixed",
			contentTypeParams: map[string]string{
				"boundary": "d36c3118be4745f9a1cb4556d11fe92d",
			},
			isMultipart: true,
			multiparts: []*ReceivedPart{
				{
					index: 0,
					headers: map[string][]string{
						"Content-Type":              {"text/plain; charset=utf-8"},
						"Content-Transfer-Encoding": {"base64"},
					},
					body: []byte(`Eine base64-enkodierte Testmail mit nicht-ASCII-Zeichen: äöüß
`),
				},
				{
					index: 1,
					headers: map[string][]string{
						"Content-Type":              {"text/plain; charset=utf-8"},
						"Content-Transfer-Encoding": {"quoted-printable"},
					},
					body: []byte(`Noch eine Testmail mit äöüß, diesmal enkodiert in quoted-printable.`),
				},
			},
		},
		{
			index:      6,
			smtpFrom:   "tästsender7@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver7@programmfabrik.de"},
			rawMessageData: []byte(`From: tästsender7@programmfabrik.de
To: testreceiver7@programmfabrik.de
Subject: Tästmail mit Ümlauten im Header

Hello World!
A simple plain text test mail.`),
			receivedAt: testTime,
			headers: map[string][]string{
				"From":    {"tästsender7@programmfabrik.de"},
				"To":      {"testreceiver7@programmfabrik.de"},
				"Subject": {"Tästmail mit Ümlauten im Header"},
			},
			body: []byte(`Hello World!
A simple plain text test mail.`),
		},
		{
			index:      7,
			smtpFrom:   "testsender8@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver8@programmfabrik.de"},
			rawMessageData: []byte(`From: =?utf-8?q?t=C3=A4stsender8=40programmfabrik=2Ede?=
To: testreceiver8@programmfabrik.de
Subject: =?utf-8?q?T=C3=A4stmail_mit_=C3=9Cmlauten_im_Header?=

Hello World!
A simple plain text test mail.`),
			receivedAt: testTime,
			headers: map[string][]string{
				"From":    {"tästsender8@programmfabrik.de"},
				"To":      {"testreceiver8@programmfabrik.de"},
				"Subject": {"Tästmail mit Ümlauten im Header"},
			},
			body: []byte(`Hello World!
A simple plain text test mail.`),
		},
		{
			index:      8,
			smtpFrom:   "testsender9@programmfabrik.de",
			smtpRcptTo: []string{"testreceiver9@programmfabrik.de"},
			rawMessageData: []byte(`MIME-Version: 1.0
From: testsender9@programmfabrik.de
To: testreceiver9@programmfabrik.de
Date: Tue, 25 Jun 2024 11:15:57 +0200
Subject: Example Message
Content-type: multipart/mixed; boundary="d36c3118be4745f9a1cb4556d11fe92d"

--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64
X-Funky-Header: =?utf-8?q?T=C3=A4stmail_mit_=C3=9Cmlauten_im_Header?=

RWluZSBiYXNlNjQtZW5rb2RpZXJ0ZSBUZXN0bWFpbCBtaXQgbmljaHQtQVNDSUktWmVpY2hlbjog
w6TDtsO8w58K
--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable
X-Funky-Header: Käse

Noch eine Testmail mit =C3=A4=C3=B6=C3=BC=C3=9F, diesmal enkodiert in quote=
d-printable.
--d36c3118be4745f9a1cb4556d11fe92d--`),
			receivedAt: testTime,
			headers: map[string][]string{
				"Mime-Version": {"1.0"},
				"From":         {"testsender9@programmfabrik.de"},
				"To":           {"testreceiver9@programmfabrik.de"},
				"Date":         {"Tue, 25 Jun 2024 11:15:57 +0200"},
				"Subject":      {"Example Message"},
				"Content-Type": {`multipart/mixed; boundary="d36c3118be4745f9a1cb4556d11fe92d"`},
			},
			body: []byte(`--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: base64
X-Funky-Header: =?utf-8?q?T=C3=A4stmail_mit_=C3=9Cmlauten_im_Header?=

RWluZSBiYXNlNjQtZW5rb2RpZXJ0ZSBUZXN0bWFpbCBtaXQgbmljaHQtQVNDSUktWmVpY2hlbjog
w6TDtsO8w58K
--d36c3118be4745f9a1cb4556d11fe92d
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable
X-Funky-Header: Käse

Noch eine Testmail mit =C3=A4=C3=B6=C3=BC=C3=9F, diesmal enkodiert in quote=
d-printable.
--d36c3118be4745f9a1cb4556d11fe92d--`),
			contentType: "multipart/mixed",
			contentTypeParams: map[string]string{
				"boundary": "d36c3118be4745f9a1cb4556d11fe92d",
			},
			isMultipart: true,
			multiparts: []*ReceivedPart{
				{
					index: 0,
					headers: map[string][]string{
						"Content-Type":              {"text/plain; charset=utf-8"},
						"Content-Transfer-Encoding": {"base64"},
						"X-Funky-Header":            {"Tästmail mit Ümlauten im Header"},
					},
					body: []byte(`Eine base64-enkodierte Testmail mit nicht-ASCII-Zeichen: äöüß
`),
				},
				{
					index: 1,
					headers: map[string][]string{
						"Content-Type":              {"text/plain; charset=utf-8"},
						"Content-Transfer-Encoding": {"quoted-printable"},
						"X-Funky-Header":            {"Käse"},
					},
					body: []byte(`Noch eine Testmail mit äöüß, diesmal enkodiert in quoted-printable.`),
				},
			},
		},
	}

	appendCRLF := func(b []byte) []byte {
		return []byte(string(b) + "\r\n")
	}

	formatRaw := func(b []byte) []byte {
		return []byte(strings.ReplaceAll(string(b), "\n", "\r\n"))
	}

	for _, m := range messages {
		m.rawMessageData = formatRaw(m.rawMessageData)
		m.rawMessageData = appendCRLF(m.rawMessageData)

		// Format message body only if not in base64 transfer encoding
		cte, ok := m.headers["Content-Transfer-Encoding"]
		if !ok || len(cte) != 1 || cte[0] != "base64" {
			m.body = formatRaw(m.body)
			m.body = appendCRLF(m.body)
		}

		for _, p := range m.multiparts {
			// Format multipart body only if not in base64 transfer encoding
			cte, ok = p.headers["Content-Transfer-Encoding"]
			if !ok || len(cte) != 1 || cte[0] != "base64" {
				p.body = formatRaw(p.body)
			}

			// Multiparts do not add a trailing CRLF
		}
	}

	return messages
}
