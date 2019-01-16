package api

import (
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/test_utils"
)

func TestDataStore_Get(t *testing.T) {
	store := NewStore()
	response := Response{statusCode: 200, body: []byte(`{"foo": "bar"}`)}
	responseJson, _ := response.ToJsonString()
	store.AppendResponse(responseJson)
	responseBytes, _ := store.Get("0")
	test_utils.AssertStringEquals(t, responseBytes.(string), `{"body":{"foo":"bar"},"header":null,"statuscode":200}`)
}

func TestDataStore_Get_BodyArray(t *testing.T) {
	store := NewStore()
	response := Response{statusCode: 200, body: []byte(`["foo", "bar"]`)}
	responseJson, _ := response.ToJsonString()
	store.AppendResponse(responseJson)
	responseBytes, err := store.Get("0")
	if err != nil {
		t.Fatal(err)
	}

	test_utils.AssertStringEquals(t, responseBytes.(string), `{"body":["foo","bar"],"header":null,"statuscode":200}`)
}

func TestDataStore_Get_Err_Index_Out_Of_Bounds(t *testing.T) {
	store := NewStore()
	response := Response{statusCode: 200, body: []byte(`BROKEN`)}
	responseJson, _ := response.ToJsonString()
	store.AppendResponse(responseJson)
	_, err := store.Get("19")
	if err == nil {
		t.Errorf("expected error, got nil")
	} else {
		if _, ok := err.(DatastoreIndexOutOfBoundsError); !ok {
			t.Errorf("Wrong error type. Expected 'DatastoreIndexOutOfBoundsError' != '%T' Got", err)
		}
	}
}
