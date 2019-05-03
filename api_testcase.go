package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/datastore"
	"time"

	"github.com/programmfabrik/fylr-apitest/lib/cjson"

	"github.com/programmfabrik/fylr-apitest/lib/api"
	"github.com/programmfabrik/fylr-apitest/lib/compare"
	"github.com/programmfabrik/fylr-apitest/lib/report"
	"github.com/programmfabrik/fylr-apitest/lib/template"
	"github.com/programmfabrik/fylr-apitest/lib/util"
	log "github.com/sirupsen/logrus"
)

const (
	defaultTimeout int = 10
)

// Case defines the structure of our single testcase
// It gets read in by our config reader at the moment the mainfest.json gets parsed
type Case struct {
	Name              string                 `json:"name"`
	RequestData       *util.GenericJson      `json:"request"`
	ResponseData      util.GenericJson       `json:"response"`
	ContinueOnFailure bool                   `json:"continue_on_failure"`
	Store             map[string]interface{} `json:"store"`                // init datastore before testrun
	StoreResponse     map[string]string      `json:"store_response_qjson"` // store qjson parsed response in datastore

	Timeout         int                `json:"timeout_ms"`
	Delay           *int               `json:"delay_ms"`
	BreakResponse   []util.GenericJson `json:"break_response"`
	CollectResponse util.GenericJson   `json:"collect_response"`

	loader      template.Loader
	manifestDir string
	reporter    *report.Report
	suiteIndex  int
	index       int
	dataStore   *datastore.Datastore
	logNetwork  bool
	logVerbose  bool

	standardHeader          map[string]*string
	standardHeaderFromStore map[string]string

	ServerURL string
}

func (testCase Case) runAPITestCase() (success bool) {
	r := testCase.reporter

	if testCase.Name == "" {
		testCase.Name = "<no name>"
	}

	log.Infof("     [%2d] '%s'", testCase.index, testCase.Name)

	start := time.Now()

	// Store standard data into datastore
	if testCase.dataStore == nil && len(testCase.Store) > 0 {
		err := fmt.Errorf("error setting datastore. Datastore is nil")
		r.SaveToReportLog(fmt.Sprintf("Error during execution: %s", err))
		log.Errorf("     [%2d] %s", testCase.index, err)

		return false
	}
	err := testCase.dataStore.SetMap(testCase.Store)
	if err != nil {
		err = fmt.Errorf("error setting datastore map:%s", err)
		r.SaveToReportLog(fmt.Sprintf("Error during execution: %s", err))
		log.Errorf("     [%2d] %s", testCase.index, err)

		return false
	}

	success = true
	if testCase.RequestData != nil {
		success, err = testCase.run()
	}

	elapsed := time.Since(start)

	if err != nil {
		r.SaveToReportLog(fmt.Sprintf("Error during execution: %s", err))
		log.Errorf("     [%2d] %s", testCase.index, err)
		success = false
	}

	if !success {
		log.WithFields(log.Fields{"elapsed": elapsed.Seconds()}).Warnf("     [%2d] failure", testCase.index)
	} else {
		log.WithFields(log.Fields{"elapsed": elapsed.Seconds()}).Infof("     [%2d] success", testCase.index)
	}

	return
}

// cheRckForBreak Response tests the given response for a so called break response.
// If this break response is present it returns a true
func (testCase Case) breakResponseIsPresent(request api.Request, response api.Response) (found bool, err error) {

	if testCase.BreakResponse != nil {
		for _, v := range testCase.BreakResponse {
			spec, err := testCase.loadResponseSerialization(v)
			if err != nil {
				return false, fmt.Errorf("error loading check response serilization: %s", err)
			}

			eResp, err := api.NewResponseFromSpec(spec)
			if err != nil {
				return false, fmt.Errorf("error loading check response from spec: %s", err)
			}

			responsesMatch, err := testCase.responsesEqual(eResp, response)
			if err != nil {
				return false, fmt.Errorf("error matching break responses: %s", err)
			}

			if testCase.logVerbose {
				log.Tracef("breakResponseIsPresent: %v", responsesMatch)
			}

			if responsesMatch.Equal {
				return true, nil
			}
		}

	}
	return false, nil
}

// checkCollectResponse loops over all given collect responses and than
// If this continue response is present it returns a true.
// If no continue response is set, it also returns true to keep the testsuite running
func (testCase *Case) checkCollectResponse(request api.Request, response api.Response) (left int, err error) {

	if testCase.CollectResponse != nil {
		_, loadedResponses, err := template.LoadManifestDataAsObject(testCase.CollectResponse, testCase.manifestDir, testCase.loader)
		if err != nil {
			return -1, fmt.Errorf("error loading check response: %s", err)
		}

		jsonRespArray := util.JsonArray{}
		switch t := loadedResponses.(type) {
		case util.JsonArray:
			jsonRespArray = t
		case util.JsonObject:
			jsonRespArray = util.JsonArray{t}
		default:
			return -1, fmt.Errorf("error loading check response no valid typew")

		}

		leftResponses := make(util.JsonArray, 0)
		for _, v := range jsonRespArray {
			spec, err := testCase.loadResponseSerialization(v)
			if err != nil {
				return -1, fmt.Errorf("error loading check response serilization: %s", err)
			}

			eResp, err := api.NewResponseFromSpec(spec)
			if err != nil {
				return -1, fmt.Errorf("error loading check response from spec: %s", err)
			}

			responsesMatch, err := testCase.responsesEqual(eResp, response)
			if err != nil {
				return -1, fmt.Errorf("error matching check responses: %s", err)
			}

			if !responsesMatch.Equal {
				leftResponses = append(leftResponses, v)
			}
		}

		testCase.CollectResponse = leftResponses

		if testCase.logVerbose {
			log.Tracef("Remaining CheckReponses: %s", testCase.CollectResponse)
		}

		return len(leftResponses), nil
	}

	return 0, nil
}

func (testCase Case) executeRequest(counter int) (
	responsesMatch compare.CompareResult,
	req api.Request,
	apiResp api.Response,
	err error) {

	// Store datastore
	err = testCase.dataStore.SetMap(testCase.Store)
	if err != nil {
		err = fmt.Errorf("error setting datastore map:%s", err)
	}

	//Do Request
	req, err = testCase.loadRequest()
	if err != nil {
		err = fmt.Errorf("error loading request: %s", err)
		return
	}

	//Log request on trace level (so only v2 will trigger this)
	if testCase.logNetwork {
		log.Tracef("[REQUEST]:\n%s", req.ToString())
	}

	apiResp, err = req.Send()
	if err != nil {
		testCase.LogReq(req)
		err = fmt.Errorf("error sending request: %s", err)
		return
	}

	apiRespJson, err := apiResp.ToJsonString()
	if err != nil {
		testCase.LogReq(req)
		err = fmt.Errorf("error getting json from response: %s", err)
		return
	}

	// Store in custom store
	err = testCase.dataStore.SetWithQjson(apiRespJson, testCase.StoreResponse)
	if err != nil {
		testCase.LogReq(req)
		err = fmt.Errorf("error store repsonse with qjson: %s", err)
		return
	}

	if !req.DoNotStore {
		var json string

		json, err = apiResp.ToJsonString()
		if err != nil {
			testCase.LogReq(req)
			err = fmt.Errorf("error prepareing response for datastore: %s", err)
			return
		}
		// Store in datastore -1 list
		if counter == 0 {
			testCase.dataStore.AppendResponse(json)
		} else {
			testCase.dataStore.UpdateLastResponse(json)
		}
	}

	//Compare Responses
	response, err := testCase.loadResponse()
	if err != nil {
		testCase.LogReq(req)
		err = fmt.Errorf("error loading response: %s", err)
		return
	}

	responsesMatch, err = testCase.responsesEqual(response, apiResp)
	if err != nil {
		testCase.LogReq(req)
		err = fmt.Errorf("error matching responses: %s", err)
		return
	}

	return
}

func (testCase Case) LogResp(response api.Response) {
	if !testCase.logNetwork && !testCase.ContinueOnFailure {
		log.Debugf("[RESPONSE]:\n%s\n", response.ToString())
	}
}

func (testCase Case) LogReq(request api.Request) {
	if !testCase.logNetwork && !testCase.ContinueOnFailure {
		log.Debugf("[REQUEST]:\n%s\n", request.ToString())
	}
}

func (testCase Case) run() (success bool, err error) {

	r := testCase.reporter

	var responsesMatch compare.CompareResult
	var request api.Request
	var apiResponse api.Response
	var timedOutFlag bool

	startTime := time.Now()

	requestCounter := 0

	collectPresent := testCase.CollectResponse != nil

	//Poll repeats the request until the right response is found, or a timeout triggers
	for {
		// delay between repeating a request
		if testCase.Delay != nil {
			time.Sleep(time.Duration(*testCase.Delay) * time.Millisecond)
		} else {
			time.Sleep(time.Duration(defaultTimeout) * time.Millisecond)
		}

		responsesMatch, request, apiResponse, err = testCase.executeRequest(requestCounter)
		if testCase.logNetwork {
			log.Debugf("[RESPONSE]:\n%s", apiResponse.ToString())
		}

		if err != nil {
			testCase.LogReq(request)
			testCase.LogResp(apiResponse)
			return false, err
		}

		if responsesMatch.Equal && !collectPresent {
			break
		}

		breakPresent, err := testCase.breakResponseIsPresent(request, apiResponse)
		if err != nil {
			testCase.LogReq(request)
			testCase.LogResp(apiResponse)
			return false, fmt.Errorf("error checking for break response: %s", err)
		}

		if breakPresent {
			testCase.LogReq(request)
			testCase.LogResp(apiResponse)
			return false, fmt.Errorf("Break response found")
		}

		collectLeft, err := testCase.checkCollectResponse(request, apiResponse)
		if err != nil {
			testCase.LogReq(request)
			testCase.LogResp(apiResponse)
			return false, fmt.Errorf("error checking for continue response: %s", err)
		}

		if collectPresent && collectLeft <= 0 {
			break

		}

		//break if timeout or we do not have a repeater
		if timedOut := time.Now().Sub(startTime) > (time.Duration(testCase.Timeout) * time.Millisecond); timedOut && testCase.Timeout != -1 {
			if timedOut && testCase.Timeout > 0 {
				log.Warnf("Pull Timeout '%dms' exceeded", testCase.Timeout)
				r.SaveToReportLogF("Pull Timeout '%dms' exceeded", testCase.Timeout)
				timedOutFlag = true
			}
			break
		}

		requestCounter++
	}

	if !responsesMatch.Equal || timedOutFlag {
		for _, v := range responsesMatch.Failures {
			log.Infof("[%s] %s", v.Key, v.Message)
			r.SaveToReportLog(fmt.Sprintf("[%s] %s", v.Key, v.Message))
		}

		collectArray, ok := testCase.CollectResponse.(util.JsonArray)
		if ok {
			for _, v := range collectArray {
				jsonV, err := json.Marshal(v)
				if err != nil {
					testCase.LogReq(request)
					testCase.LogResp(apiResponse)
					return false, err
				}
				log.Warnf("Collect response not found: %s", jsonV)
				r.SaveToReportLog(fmt.Sprintf("Collect response not found: %s", jsonV))
			}
		}

		testCase.LogReq(request)
		testCase.LogResp(apiResponse)
		return false, nil
	}

	return true, nil
}

func (testCase Case) loadRequest() (req api.Request, err error) {
	req, err = testCase.loadRequestSerialization()
	if err != nil {
		return req, fmt.Errorf("error loadRequestSerialization: %s", err)
	}

	return req, err
}

func (testCase Case) loadResponse() (res api.Response, err error) {
	// unspecified response is interpreted as status_code 200
	if testCase.ResponseData == nil {
		return api.NewResponse(200, nil, bytes.NewReader([]byte("")))
	}
	spec, err := testCase.loadResponseSerialization(testCase.ResponseData)
	if err != nil {
		return res, fmt.Errorf("error loading response spec: %s", err)
	}
	res, err = api.NewResponseFromSpec(spec)
	if err != nil {
		return res, fmt.Errorf("error creating response from spec: %s", err)
	}
	return res, nil
}

func (testCase Case) responsesEqual(left, right api.Response) (equal compare.CompareResult, err error) {
	leftJSON, err := left.ToGenericJson()
	if err != nil {
		return compare.CompareResult{}, fmt.Errorf("error loading generic json: %s", err)
	}
	rightJSON, err := right.ToGenericJson()
	if err != nil {
		return compare.CompareResult{}, fmt.Errorf("error loading generic json: %s", err)
	}
	return compare.JsonEqual(leftJSON, rightJSON, compare.ComparisonContext{})
}

func (testCase Case) loadRequestSerialization() (spec api.Request, err error) {
	_, requestData, err := template.LoadManifestDataAsObject(*testCase.RequestData, testCase.manifestDir, testCase.loader)
	if err != nil {
		return spec, fmt.Errorf("error loading request data: %s", err)
	}
	specBytes, err := cjson.Marshal(requestData)
	if err != nil {
		return spec, fmt.Errorf("error marshaling req: %s", err)
	}
	err = cjson.Unmarshal(specBytes, &spec)
	spec.ManifestDir = testCase.manifestDir
	spec.DataStore = testCase.dataStore
	spec.ServerURL = testCase.ServerURL

	if len(spec.Headers) == 0 {
		spec.Headers = make(map[string]*string, 0)
	}
	for k, v := range testCase.standardHeader {
		if spec.Headers[k] == nil {
			spec.Headers[k] = v
		}
	}

	if len(spec.HeaderFromStore) == 0 {
		spec.HeaderFromStore = make(map[string]string, 0)
	}
	for k, v := range testCase.standardHeaderFromStore {
		if spec.HeaderFromStore[k] == "" {
			spec.HeaderFromStore[k] = v
		}
	}

	return
}

func (testCase Case) loadResponseSerialization(genJSON util.GenericJson) (spec api.ResponseSerialization, err error) {
	_, responseData, err := template.LoadManifestDataAsObject(genJSON, testCase.manifestDir, testCase.loader)
	if err != nil {
		return spec, fmt.Errorf("error loading response data: %s", err)
	}
	specBytes, err := cjson.Marshal(responseData)
	if err != nil {
		return spec, fmt.Errorf("error marshaling res: %s", err)
	}
	err = cjson.Unmarshal(specBytes, &spec)
	if err != nil {
		return spec, fmt.Errorf("error unmarshaling res: %s", err)
	}

	return spec, nil
}
