package api

import (
	"testing"

	"github.com/programmfabrik/go-test-utils"
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

func TestDataStore_MAP(t *testing.T) {
	store := NewStore()

	store.Set("test[head1]", 2)
	store.Set("test[head3]", nil)
	store.Set("test[head3]", 3)
	store.Set("test[head1]", 1)

	tA, err := store.Get("test")
	if err != nil {
		t.Fatal(err)
	}

	if tA.(map[string]interface{})["head1"] != 1 {
		t.Errorf("Have '%v' != '%d' Want", tA.(map[string]interface{})["head1"], 1)
	}

	if tA.(map[string]interface{})["head3"] != 3 {
		t.Errorf("Have '%v' != '%d' Want", tA.(map[string]interface{})["head1"], 3)
	}
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
