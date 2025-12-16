package main

import (
	"bufio"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	"github.com/programmfabrik/golib"

	"github.com/programmfabrik/apitest/pkg/lib/api"
	"github.com/programmfabrik/apitest/pkg/lib/compare"
	"github.com/programmfabrik/apitest/pkg/lib/report"
	"github.com/programmfabrik/apitest/pkg/lib/template"
	"github.com/sirupsen/logrus"
)

// Case defines the structure of our single testcase
// It gets read in by our config reader at the moment the mainfest.json gets parsed
type Case struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	RequestData       *any              `json:"request"`
	ResponseData      any               `json:"response"`
	ContinueOnFailure bool              `json:"continue_on_failure"`
	Store             map[string]any    `json:"store"`                // init datastore before testrun
	StoreResponse     map[string]string `json:"store_response_gjson"` // store gjson parsed response in datastore

	Timeout         int   `json:"timeout_ms"`
	WaitBefore      *int  `json:"wait_before_ms"`
	WaitAfter       *int  `json:"wait_after_ms"`
	Delay           *int  `json:"delay_ms"`
	BreakResponse   []any `json:"break_response"`
	CollectResponse any   `json:"collect_response"`

	LogNetwork *bool `json:"log_network"`
	LogVerbose *bool `json:"log_verbose"`
	LogShort   *bool `json:"log_short"`

	loader      template.Loader
	manifestDir string
	ReportElem  *report.ReportElement
	suiteIndex  int
	index       int
	dataStore   *datastore.Datastore

	standardHeader          map[string]any // can be string or []string
	standardHeaderFromStore map[string]string

	ServerURL         string `json:"server_url"`
	ReverseTestResult bool   `json:"reverse_test_result"`

	Filename string
}

func (testCase Case) runAPITestCase(parentReportElem *report.ReportElement) (success bool) {
	if testCase.Name == "" {
		testCase.Name = "<no name>"
	}
	if testCase.LogShort == nil || !*testCase.LogShort {
		if testCase.Description == "" {
			logrus.Infof("     [%2d] '%s'", testCase.index, testCase.Name)
		} else {
			logrus.Infof("     [%2d] '%s': '%s'", testCase.index, testCase.Name, testCase.Description)
		}
	}

	testCase.ReportElem = parentReportElem.NewChild(testCase.Name)
	r := testCase.ReportElem

	start := time.Now()

	// Store standard data into datastore
	if testCase.dataStore == nil && len(testCase.Store) > 0 {
		err := fmt.Errorf("setting datastore. Datastore is nil")
		r.SaveToReportLog(fmt.Sprintf("Error during execution: %s", err.Error()))
		logrus.Errorf("     [%2d] %s", testCase.index, err.Error())
		return false
	}
	err := testCase.dataStore.SetMap(testCase.Store)
	if err != nil {
		err = fmt.Errorf("setting datastore map: %w", err)
		r.SaveToReportLog(fmt.Sprintf("Error during execution: %s", err.Error()))
		logrus.Errorf("     [%2d] %s", testCase.index, err.Error())
		return false
	}

	success = true
	var apiResponse api.Response
	if testCase.RequestData != nil {
		success, apiResponse, err = testCase.run()
	}

	elapsed := time.Since(start)
	if err != nil {
		r.SaveToReportLog(fmt.Sprintf("Error during execution: %s", err.Error()))
		if !testCase.ReverseTestResult || testCase.LogShort == nil || !*testCase.LogShort {
			logrus.Errorf("     [%2d] %s", testCase.index, err.Error())
		}
		success = false
	}

	// Reverse if needed
	if testCase.ReverseTestResult {
		success = !success
	}

	fileBasename := filepath.Base(testCase.Filename)
	logF := logrus.Fields{
		"elapsed": elapsed.String(),
		"size":    golib.HumanByteSize(uint64(len(apiResponse.Body))),
		"request": apiResponse.ReqDur.String(),
		"body":    apiResponse.BodyLoadDur.String(),
		"file":    fileBasename,
	}

	if !success {
		logrus.WithFields(logF).Warnf("     [%2d] failure", testCase.index)
	} else if testCase.LogShort == nil || !*testCase.LogShort {
		logrus.WithFields(logF).Infof("     [%2d] success", testCase.index)
	}

	r.Leave(success)
	return success
	// res.Success = success
	// res.BodySize = uint64(len(apiResponse.Body))
	// res.BodyLoadDur = apiResponse.BodyLoadDur
	// res.RequestDur = apiResponse.ReqDur
	// return res
}

// cheRckForBreak Response tests the given response for a so called break response.
// If this break response is present it returns a true
func (testCase Case) breakResponseIsPresent(response api.Response) (present bool, err error) {
	if testCase.BreakResponse == nil {
		return false, nil
	}

	var (
		spec             api.ResponseSerialization
		expectedResponse api.Response
		responsesMatch   compare.CompareResult
	)

	for _, v := range testCase.BreakResponse {
		spec, err = testCase.loadResponseSerialization(v)
		if err != nil {
			return false, fmt.Errorf("loading check response serilization: %w", err)
		}

		expectedResponse, err = api.NewResponseFromSpec(spec)
		if err != nil {
			return false, fmt.Errorf("loading check response from spec: %w", err)
		}

		if expectedResponse.Format.Type != "" {
			response.Format = expectedResponse.Format
		} else {
			expectedResponse.Format = response.Format
		}

		responsesMatch, err = testCase.responsesEqual(expectedResponse, response)
		if err != nil {
			return false, fmt.Errorf("matching break responses: %w", err)
		}

		if testCase.LogVerbose != nil && *testCase.LogVerbose {
			logrus.Tracef("breakResponseIsPresent: %v", responsesMatch)
		}

		if responsesMatch.Equal {
			return true, nil
		}
	}

	return false, nil
}

// checkCollectResponse loops over all given collect responses and
// if this continue response is present it returns the number of responses.
// If no continue response is set, it returns -1 to keep the testsuite running
func (testCase *Case) checkCollectResponse(response api.Response) (responses int, err error) {
	if testCase.CollectResponse == nil {
		return 0, nil
	}

	var (
		loadedResponses any
		jsonRespArray   jsutil.Array
		leftResponses   jsutil.Array
		spec            api.ResponseSerialization
		responsesMatch  compare.CompareResult
	)

	_, loadedResponses, err = template.LoadManifestDataAsObject(testCase.CollectResponse, testCase.manifestDir, testCase.loader)
	if err != nil {
		return -1, fmt.Errorf("loading check response: %w", err)
	}

	switch t := loadedResponses.(type) {
	case jsutil.Array:
		jsonRespArray = t
	case jsutil.Object:
		jsonRespArray = jsutil.Array{t}
	default:
		return -1, fmt.Errorf("loading check response: no valid type")
	}

	leftResponses = make(jsutil.Array, 0)
	for _, v := range jsonRespArray {
		spec, err = testCase.loadResponseSerialization(v)
		if err != nil {
			return -1, fmt.Errorf("loading check response serilization: %w", err)
		}

		expectedResponse, err := api.NewResponseFromSpec(spec)
		if err != nil {
			return -1, fmt.Errorf("loading check response from spec: %w", err)
		}

		if expectedResponse.Format.Type != "" || expectedResponse.Format.PreProcess != nil {
			response.Format = expectedResponse.Format
		} else {
			expectedResponse.Format = response.Format
		}

		responsesMatch, err = testCase.responsesEqual(expectedResponse, response)
		if err != nil {
			return -1, fmt.Errorf("matching check responses: %w", err)
		}
		// !eq && !reverse -> add
		// !eq && reverse -> don't add
		// eq && !reverse -> don't add
		// eq && reverse -> add

		if !responsesMatch.Equal && !testCase.ReverseTestResult ||
			responsesMatch.Equal && testCase.ReverseTestResult {
			leftResponses = append(leftResponses, v)
		}
	}

	testCase.CollectResponse = leftResponses

	if testCase.LogVerbose != nil && *testCase.LogVerbose {
		logrus.Tracef("Remaining CheckReponses: %s", testCase.CollectResponse)
	}

	return len(leftResponses), nil
}

func (testCase Case) executeRequest(counter int) (responsesMatch compare.CompareResult, req api.Request, apiResp api.Response, err error) {

	// Store datastore
	err = testCase.dataStore.SetMap(testCase.Store)
	if err != nil {
		err = fmt.Errorf("setting datastore map: %w", err)
		return responsesMatch, req, apiResp, err
	}

	// Do Request
	req, err = testCase.loadRequest()
	if err != nil {
		err = fmt.Errorf("loading request: %w", err)
		return responsesMatch, req, apiResp, err
	}

	// Log request on trace level (so only v2 will trigger this)
	if testCase.LogNetwork != nil && *testCase.LogNetwork {
		logrus.Tracef("[REQUEST]:\n%s\n\n", limitLines(req.ToString(logCurl), Config.Apitest.Limit.Request))
	}

	expRes, err := testCase.loadExpectedResponse()
	if err != nil {
		testCase.logReq(req)
		err = fmt.Errorf("loading response: %w", err)
		return responsesMatch, req, apiResp, err
	}

	apiResp, err = req.Send()
	if err != nil {
		testCase.logReq(req)
		err = fmt.Errorf("sending request: %w", err)
		return responsesMatch, req, apiResp, err
	}

	apiResp.Format = expRes.Format

	apiRespJsonString, err := apiResp.ServerResponseToJsonString(false)
	// If we don't define an expected response, we won't have a format
	// That's problematic if the response is not JSON, as we try to parse it for the datastore anyway
	// So we don't fail the test in that edge case
	if err != nil && (testCase.ResponseData != nil || len(testCase.StoreResponse) > 0) {
		testCase.logReq(req)
		err = fmt.Errorf("getting json from response: %w", err)
		return responsesMatch, req, apiResp, err
	}

	// Store in custom store
	err = testCase.dataStore.SetWithGjson(apiRespJsonString, testCase.StoreResponse)
	if err != nil {
		testCase.logReq(req)
		err = fmt.Errorf("store response with gjson: %w", err)
		return responsesMatch, req, apiResp, err
	}

	// Store in datastore -1 list
	if counter == 0 {
		testCase.dataStore.AppendResponse(apiRespJsonString)
	} else {
		testCase.dataStore.UpdateLastResponse(apiRespJsonString)
	}

	// Compare Responses
	responsesMatch, err = testCase.responsesEqual(expRes, apiResp)
	if err != nil {
		testCase.logReq(req)
		err = fmt.Errorf("matching responses: %w", err)
		return responsesMatch, req, apiResp, err
	}

	return responsesMatch, req, apiResp, nil
}

func (testCase Case) logBody(prefix, body string, limit int) {
	if testCase.ReverseTestResult {
		return
	}
	if testCase.ContinueOnFailure {
		return
	}
	if testCase.LogNetwork == nil || *testCase.LogNetwork {
		return
	}

	errString := fmt.Sprintf("[%s]:\n%s\n\n", prefix, limitLines(body, limit))
	testCase.ReportElem.SaveToReportLog(errString)
	logrus.Debug(errString)
}

func (testCase Case) logResp(response api.Response) {
	testCase.logBody("RESPONSE", response.ToString(), Config.Apitest.Limit.Response)
}

// logReq print the request to the console
func (testCase Case) logReq(req api.Request) {
	testCase.logBody("REQUEST", req.ToString(logCurl), Config.Apitest.Limit.Request)
}

func limitLines(in string, limitCount int) string {
	if limitCount <= 0 {
		return in
	}
	out := ""
	scanner := bufio.NewScanner(strings.NewReader(in))
	k := 0
	for scanner.Scan() && k < limitCount {
		out += scanner.Text() + "\n"
		k++
	}
	if k >= limitCount {
		out += fmt.Sprintf("[Limited after %d lines]", limitCount)
	}
	return out
}

func (testCase Case) run() (successs bool, apiResponse api.Response, err error) {
	var (
		responsesMatch compare.CompareResult
		request        api.Request
		timedOutFlag   bool
	)

	startTime := time.Now()
	r := testCase.ReportElem
	requestCounter := 0
	collectPresent := testCase.CollectResponse != nil

	if testCase.WaitBefore != nil {
		if testCase.LogShort == nil || !*testCase.LogShort {
			logrus.Infof("wait_before_ms: %d", *testCase.WaitBefore)
		}
		time.Sleep(time.Duration(*testCase.WaitBefore) * time.Millisecond)
	}

	// Poll repeats the request until the right response is found, or a timeout triggers
	for {
		// delay between repeating a request
		if testCase.Delay != nil {
			time.Sleep(time.Duration(*testCase.Delay) * time.Millisecond)
		}

		responsesMatch, request, apiResponse, err = testCase.executeRequest(requestCounter)
		if testCase.LogNetwork != nil && *testCase.LogNetwork {
			logrus.Debugf("[RESPONSE]:\n%s\n\n", limitLines(apiResponse.ToString(), Config.Apitest.Limit.Response))
		}
		if err != nil {
			testCase.logResp(apiResponse)
			return false, apiResponse, err
		}

		if responsesMatch.Equal && !collectPresent {
			break
		}

		breakPresent, err := testCase.breakResponseIsPresent(apiResponse)
		if err != nil {
			testCase.logReq(request)
			testCase.logResp(apiResponse)
			return false, apiResponse, fmt.Errorf("checking for break response: %w", err)
		}

		if breakPresent {
			testCase.logReq(request)
			testCase.logResp(apiResponse)
			return false, apiResponse, fmt.Errorf("break response found")
		}

		collectLeft, err := testCase.checkCollectResponse(apiResponse)
		if err != nil {
			testCase.logReq(request)
			testCase.logResp(apiResponse)
			return false, apiResponse, fmt.Errorf("checking for continue response: %w", err)
		}

		if collectPresent && collectLeft <= 0 {
			break
		}

		// break if timeout or we do not have a repeater
		timedOut := time.Since(startTime) > (time.Duration(testCase.Timeout) * time.Millisecond)
		if timedOut && testCase.Timeout != -1 {
			if timedOut && testCase.Timeout > 0 {
				logrus.Warnf("Pull Timeout '%dms' exceeded", testCase.Timeout)
				r.SaveToReportLogF("Pull Timeout '%dms' exceeded", testCase.Timeout)
				timedOutFlag = true
			}
			break
		}

		requestCounter++
	}

	if !responsesMatch.Equal || timedOutFlag {
		if !testCase.ReverseTestResult {
			for _, v := range responsesMatch.Failures {
				logrus.Errorf("[%s] %s", v.Key, v.Message)
				r.SaveToReportLog(fmt.Sprintf("[%s] %s", v.Key, v.Message))
			}
		} else {
			for _, v := range responsesMatch.Failures {
				logrus.Infof("Reverse Test Result of: [%s] %s", v.Key, v.Message)
				r.SaveToReportLog(fmt.Sprintf("reverse test result: [%s] %s", v.Key, v.Message))
			}
		}

		collectArray, ok := testCase.CollectResponse.(jsutil.Array)
		if ok {
			for _, v := range collectArray {
				jsonV, err := jsutil.Marshal(v)
				if err != nil {
					testCase.logReq(request)
					testCase.logResp(apiResponse)
					return false, apiResponse, err
				}
				logrus.Errorf("Collect response not found: %s", jsonV)
				r.SaveToReportLog(fmt.Sprintf("Collect response not found: %s", jsonV))
			}
		}

		testCase.logReq(request)
		testCase.logResp(apiResponse)
		return false, apiResponse, nil
	}

	if testCase.WaitAfter != nil {
		if testCase.LogShort == nil || !*testCase.LogShort {
			logrus.Infof("wait_after_ms: %d", *testCase.WaitAfter)
		}
		time.Sleep(time.Duration(*testCase.WaitAfter) * time.Millisecond)
	}

	return true, apiResponse, nil
}

func (testCase Case) loadRequest() (req api.Request, err error) {
	req, err = testCase.loadRequestSerialization()
	if err != nil {
		return api.Request{}, fmt.Errorf("loadRequestSerialization: %w", err)
	}
	return req, err
}

func (testCase Case) loadExpectedResponse() (res api.Response, err error) {
	// unspecified response is interpreted as status_code 200
	if testCase.ResponseData == nil {
		return api.NewResponse(golib.IntRef(http.StatusOK), nil, nil, nil, nil, res.Format)
	}
	spec, err := testCase.loadResponseSerialization(testCase.ResponseData)
	if err != nil {
		return res, fmt.Errorf("loading response spec: %w", err)
	}
	res, err = api.NewResponseFromSpec(spec)
	if err != nil {
		return res, fmt.Errorf("creating response from spec: %w", err)
	}
	return res, nil
}

func (testCase Case) responsesEqual(expected, got api.Response) (comp compare.CompareResult, err error) {
	if expected.StatusCode == nil {
		// if the statuscode is not set, use the default status code 200
		expected.StatusCode = golib.IntRef(200)
	} else {
		// if the statuscode is set to 0,
		// remove the statuscode key from the expected response to accept any response code
		if *expected.StatusCode == 0 {
			expected.StatusCode = nil
		}
	}

	var (
		expectedJSON any
		gotJSON      any
	)

	expectedJSON, err = expected.ToGenericJSON()
	if err != nil {
		return comp, fmt.Errorf("loading expected generic json: %w", err)
	}
	if len(expected.Body) == 0 && len(expected.BodyControl) == 0 {
		expected.Format.IgnoreBody = true
	} else {
		expected.Format.IgnoreBody = false
	}

	gotJSON, err = got.ServerResponseToGenericJSON(expected.Format, false)
	if err != nil {
		return comp, fmt.Errorf("loading response generic json: %w", err)
	}
	return compare.JsonEqual(expectedJSON, gotJSON, compare.ComparisonContext{})
}

func (testCase Case) loadRequestSerialization() (req api.Request, err error) {
	var (
		spec        api.Request
		requestData any
		specBytes   []byte
	)

	_, requestData, err = template.LoadManifestDataAsObject(*testCase.RequestData, testCase.manifestDir, testCase.loader)
	if err != nil {
		return spec, fmt.Errorf("loading request data: %w", err)
	}

	specBytes, err = jsutil.Marshal(requestData)
	if err != nil {
		return spec, fmt.Errorf("marshaling requet: %w", err)
	}
	err = jsutil.Unmarshal(specBytes, &spec)
	spec.ManifestDir = testCase.manifestDir
	spec.DataStore = testCase.dataStore

	if spec.ServerURL == "" {
		spec.ServerURL = testCase.ServerURL
	}
	if len(spec.Headers) == 0 {
		spec.Headers = make(map[string]any)
	}
	for k, v := range testCase.standardHeader {
		_, exist := spec.Headers[k]
		if !exist {
			spec.Headers[k] = v
		}
	}

	if len(spec.HeaderFromStore) == 0 {
		spec.HeaderFromStore = make(map[string]string)
	}
	for k, v := range testCase.standardHeaderFromStore {
		_, exist := spec.HeaderFromStore[k]
		if !exist {
			spec.HeaderFromStore[k] = v
		}
	}

	return spec, nil
}

func (testCase Case) loadResponseSerialization(genJSON any) (spec api.ResponseSerialization, err error) {
	resLoader := testCase.loader
	_, responseData, err := template.LoadManifestDataAsObject(genJSON, testCase.manifestDir, resLoader)
	if err != nil {
		return spec, fmt.Errorf("loading response data: %w", err)
	}

	specBytes, err := jsutil.Marshal(responseData)
	if err != nil {
		return spec, fmt.Errorf("marshaling response: %w", err)
	}

	err = jsutil.Unmarshal(specBytes, &spec)
	if err != nil {
		return spec, fmt.Errorf("unmarshaling response: %w", err)
	}

	// the body must not be parsed if it is not expected in the response, or should not be stored
	if spec.Body == nil && len(testCase.StoreResponse) == 0 {
		spec.Format.IgnoreBody = true
	}

	return spec, nil
}
