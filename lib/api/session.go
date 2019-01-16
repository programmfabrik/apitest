package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/programmfabrik/fylr-apitest/lib/logging"
)

/*
Performs API Calls in the context of the TestSuite
*/

type Session struct {
	Store      *Datastore
	client     *http.Client
	serverUrl  string
	token      string
	MaxEventId int
}

type SessionAuthentication struct {
	Login         string            `json:"login"`
	Password      string            `json:"password"`
	Method        string            `json:"method"`
	StoreResponse map[string]string `json:"store_response_qjson"` // store qjson parsed response in datastore
}

func NewSession(
	serverUrl string,
	client *http.Client,
	auth *SessionAuthentication,
	store *Datastore,
) (session Session, err error) {

	session.client = client
	session.serverUrl = serverUrl
	session.Store = store

	response, err := session.SendSessionRequest()
	if err != nil {
		return session, fmt.Errorf("error GET-ing session endpoint: %s", err)
	}

	session.token = response.Token
	if auth != nil {
		if err = session.login(*auth); err != nil {
			return session, fmt.Errorf("error logging session in: %s", err)
		}
	}
	return session, nil
}

//	login gets you an authenticated session token
//	The supported authentication methods are:
//		“easydb” (default): authenticate easydb user
//			login: ist the username
//			password: is the users password
//		“email”: authenticate an easydb user of type “email”
//			login: its email
//			password: collection UUID
//		“anonymous”: a virtual user will be created
//			login: empty
//			password: empty
//		“task”: authenticate easydb to perform a task
//			login: email-adress
//			password: authentication token for the process
//		“collection”: authenticate an easydb user of type “collection”
//			login: collection UUID
//			password: collection secret
func (session *Session) login(auth SessionAuthentication) (err error) {

	response, err := session.SendSessionAuthenticateRequest(auth)
	if err != nil {
		return fmt.Errorf("error performing authentication request: %s", err)
	}

	session.MaxEventId = response.CurMaxEventId
	return nil
}

func (session *Session) SendRequest(request Request) (response Response, err error) {
	logging.DebugWithVerbosityf(logging.V2, "request: %s", request.ToString(*session))
	httpRequest, err := request.buildHttpRequest(session.serverUrl, session.token)
	if err != nil {
		return response, err
	}

	httpResponse, err := session.client.Do(httpRequest)
	if err != nil {
		return response, err
	}
  
	response, err = NewResponse(httpResponse.StatusCode, httpResponse.Header, httpResponse.Body)
	if err != nil {
		return response, fmt.Errorf("error reading httpResponse.Body: %s", err)
	}
	response = Response{
		StatusCode: httpResponse.StatusCode,
		Header:     httpResponse.Header,
		Body:       bodyBytes,
	}
	logging.DebugWithVerbosityf(logging.V2, "response: %s", response.ToString())

	return response, err
}

/*
Convenience methods from within the application
*/

//Body of GET /settings response
//https://docs.easydb.de/en/technical/api/settings/settings.html
type settingsBody struct {
	Name           string `json:"name"`
	Api            int    `json:"api"`
	ServerVersion  int    `json:"server_version"`
	UserSchema     int    `json:"user-schema"`
	Solution       string `json:"solution"`
	DbName         string `json:"db-name"`
	ExternalEasUrl string `json:"external_eas_url"`
	//startup time left out
	//server time left out
}

func (session *Session) SendSettingsRequest() (res settingsBody, err error) {
	request := Request{
		Endpoint:   "settings",
		Method:     "GET",
		DoNotStore: true,
	}
	resp, err := session.SendRequest(request)
	if err != nil {
		return res, err
	}

	if resp.StatusCode != 200 {
		apiErr := apiError{}
		if err = resp.marshalBodyInto(&apiErr); err != nil {
			return res, err
		}
		apiErr.Statuscode = resp.StatusCode

		return res, apiErr
	}

	if err = resp.marshalBodyInto(&res); err != nil {
		return res, err
	}
	return res, nil
}

//Body of GET /session response
//https://docs.easydb.de/en/technical/types/session/session.html
type sessionBody struct {
	Token string `json:"token"`
	//...
}

type apiError struct {
	Statuscode int    `json:"statuscode"`
	Code       string `json:"code"`
}

func (apiErr apiError) Error() string {
	jsonErr, err := json.Marshal(apiErr)
	if err != nil {
		return err.Error()
	}
	return string(jsonErr)
}

func (session *Session) SendSessionRequest() (res sessionBody, err error) {
	request := Request{
		Endpoint:   "session",
		Method:     "GET",
		DoNotStore: true,
	}
	resp, err := session.SendRequest(request)
	if err != nil {
		return res, err
	}

	if resp.StatusCode != 200 {
		apiErr := apiError{}
		if err = resp.marshalBodyInto(&apiErr); err != nil {
			return res, err
		}
		apiErr.Statuscode = resp.StatusCode

		return res, apiErr
	}

	if err = resp.marshalBodyInto(&res); err != nil {
		return res, err
	}
	return res, nil
}

//Body of POST /session/authenticate response
//https://docs.easydb.de/en/technical/api/session/session.html
type sessionAuthenticateBody struct {
	CurMaxEventId int `json:"current_max_event_id"`
}

func (session *Session) SendSessionAuthenticateRequest(auth SessionAuthentication) (res sessionAuthenticateBody, err error) {
	request := Request{
		Endpoint: "session/authenticate",
		Method:   "POST",
		Headers: map[string]string{
			"token":  session.token,
			"method": auth.Method,
		},
		Body: map[string]string{
			"login":    auth.Login,
			"password": auth.Password,
		},
		BodyType:   "urlencoded",
		DoNotStore: true,
	}

	resp, err := session.SendRequest(request)
	if err != nil {
		return res, err
	}
	if resp.statusCode != 200 {
		apiErr := apiError{}
		if err = resp.marshalBodyInto(&apiErr); err != nil {
			return res, err
		}
		apiErr.Statuscode = resp.statusCode

		return res, apiErr
	}

	err = session.Store.SetWithQjson(resp, auth.StoreResponse)
	if err != nil {
		err = fmt.Errorf("unable to store response of session %s", err)
		return
	}

	if err = resp.marshalBodyInto(&res); err != nil {
		return res, err
	}
	return res, nil
}
