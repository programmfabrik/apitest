package datastore

import (
	"testing"

	"github.com/programmfabrik/go-test-utils"
)

func TestDataStore_Get(t *testing.T) {
	store := NewStore(false)
	store.AppendResponse(`{"body":{"foo":"bar"},"statuscode":200}`)
	responseBytes, _ := store.Get("0")
	test_utils.AssertStringEquals(t, responseBytes.(string), `{"body":{"foo":"bar"},"statuscode":200}`)
}

func TestDataStore_GetSlice(t *testing.T) {
	store := NewStore(false)
	store.Set("slice[]", "val1")
	store.Set("slice[]", "val2")
	store.Set("slice[]", "val2")
	responseBytes, err := store.Get("slice[2]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), `val2`)

	responseBytes, err = store.Get("slice[3]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), ``)

	responseBytes, err = store.Get("slice[-1]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), `val2`)

	responseBytes, err = store.Get("slice[-15]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), ``)
}

func TestStoreTypeInt(t *testing.T) {
	store := NewStore(false)
	store.Set("ownInt", 1.0)
	store.SetWithQjson(`{"id",1.000000}`, map[string]string{"jsonInt": "id"})

	oVal, _ := store.Get("ownInt")
	jVal, _ := store.Get("jsonInt")

	if oVal != jVal {
		t.Errorf("%d != %d", oVal, jVal)
	}

	store.Set("ownInt", 1.1)
	store.SetWithQjson(`{"id",1.100000}`, map[string]string{"jsonInt": "id"})

	oVal, _ = store.Get("ownInt")
	jVal, _ = store.Get("jsonInt")

	if oVal != jVal {
		t.Errorf("%f != %f", oVal, jVal)
	}

}

func TestDataStore_GetMap(t *testing.T) {
	store := NewStore(false)
	store.Set("map[key1]", "val1")
	store.Set("map[key2]", "val2")
	store.Set("map[key3]", "val2")
	responseBytes, err := store.Get("map[key2]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), `val2`)

	responseBytes, err = store.Get("map[key5]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), ``)

	responseBytes, err = store.Get("map[-1]")
	test_utils.AssertErrorEquals(t, err, nil)
	test_utils.AssertStringEquals(t, responseBytes.(string), ``)
}

func TestDataStore_Get_BodyArray(t *testing.T) {
	store := NewStore(false)
	store.AppendResponse(`{"body":["foo","bar"],"statuscode":200}`)
	responseBytes, err := store.Get("0")
	if err != nil {
		t.Fatal(err)
	}

	test_utils.AssertStringEquals(t, responseBytes.(string), `{"body":["foo","bar"],"statuscode":200}`)
}

func TestDataStore_MAP(t *testing.T) {
	store := NewStore(false)

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
	store := NewStore(false)
	store.AppendResponse(`BROKEN`)
	_, err := store.Get("19")
	if err == nil {
		t.Errorf("expected error, got nil")
	} else {
		if _, ok := err.(DatastoreIndexOutOfBoundsError); !ok {
			t.Errorf("Wrong error type. Expected 'DatastoreIndexOutOfBoundsError' != '%T' Got", err)
		}
	}
}
