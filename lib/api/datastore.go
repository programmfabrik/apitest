package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	// "reflect"
	// "github.com/programmfabrik/fylr-apitest/lib/cjson"
)

type Datastore struct {
	storage      map[string]interface{} // custom storage
	responseJson []string               // store the responses
}

// returns a new store, with the storage copied
func NewStoreShare(datastore *Datastore) *Datastore {
	ds := Datastore{}
	ds.storage = datastore.storage
	ds.responseJson = make([]string, 0)
	return &ds
}

func NewStore() *Datastore {
	ds := Datastore{}
	ds.storage = make(map[string]interface{}, 0)
	ds.responseJson = make([]string, 0)
	return &ds
}

type DatastoreIndexOutOfBoundsError struct {
	error string
}

func (data DatastoreIndexOutOfBoundsError) Error() string {
	return data.error
}

// SetWithQjson stores the given response driven by a map key => qjson
func (this *Datastore) SetWithQjson(response Response, storeResponse map[string]string) error {
	json, err := response.ToJsonString()
	if err != nil {
		return err
	}
	for k, qv := range storeResponse {
		qValue := gjson.Get(json, qv)
		if qValue.Value() == nil {
			continue
		}
		setValue := qValue.Value()
		if qValue.Type == gjson.Number {
			//Check if float is int
			if fmt.Sprintf("%.0f", qValue.Float()) == fmt.Sprintf("%d", qValue.Int()) {
				setValue = qValue.Int()
			}
		}

		err := this.Set(k, setValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// We store the response
func (this *Datastore) UpdateLastResponse(s string) {
	this.responseJson[len(this.responseJson)-1] = s
}

// We store the response
func (this *Datastore) AppendResponse(s string) {
	this.responseJson = append(this.responseJson, s)
}

func (this *Datastore) SetMap(smap map[string]interface{}) error {
	for k, v := range smap {
		err := this.Set(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Datastore) Set(index string, value interface{}) error {
	if strings.HasSuffix(index, "[]") {
		// do a push to an array
		use_index := index[:len(index)-2]
		_, ok := this.storage[use_index]
		if !ok {
			this.storage[use_index] = make([]interface{}, 0)
		}

		s, ok := this.storage[use_index].([]interface{})
		if !ok {
			tmp := this.storage[use_index]
			this.storage[use_index] = make([]interface{}, 0)
			s = this.storage[use_index].([]interface{})

			if tmp != nil {
				this.storage[use_index] = append(s, tmp)
			}
		}

		this.storage[use_index] = append(s, value)

		//logging.Debugf("datastore[\"%s\"][]=%#v", use_index, value)
	} else {
		this.storage[index] = value
		//logging.Debugf("datastore[\"%s\"]=%#v", index, value)
	}
	//logging.Debugf("datastore %#v", this)
	return nil
}

func (this Datastore) Get(index string) (interface{}, error) {
	// strings are evalulated as int, so
	// that we can support "-<int>" notations

	if index == "-" {
		// return the entire custom store
		return this.storage, nil
	}

	idx, err := strconv.Atoi(index)
	if err == nil {
		if idx < 0 {
			idx = idx + len(this.responseJson)
		}
		if idx >= len(this.responseJson) || idx < 0 {
			// index out of range
			return "", DatastoreIndexOutOfBoundsError{error: fmt.Sprintf("datastore.Get: idx out of range: %d, current length: %d", idx, len(this.responseJson))}
		}
		return this.responseJson[idx], nil
	}

	res, ok := this.storage[index]

	// logging.Warnf("datastore: %s", store.storage)
	if !ok {
		// logging.Warnf("datastore: key: %s not found.", index)
		return "", nil
	}
	return res, nil
}
