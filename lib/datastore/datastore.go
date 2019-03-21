package datastore

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"regexp"
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
func (this *Datastore) SetWithQjson(jsonResponse string, storeResponse map[string]string) error {
	for k, qv := range storeResponse {
		qValue := gjson.Get(jsonResponse, qv)
		if qValue.Value() == nil {
			log.Tracef("'%s' was not found in '%s'", qv, jsonResponse)
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
	var dsMapRegex = regexp.MustCompile(`^(.*?)\[(.+?)\]$`)

	//Slice in datastore
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

	} else if rego := dsMapRegex.FindStringSubmatch(index); len(rego) > 0 {
		// do a push to an array
		use_index := rego[1]
		_, ok := this.storage[use_index]
		if !ok {
			this.storage[use_index] = make(map[string]interface{}, 0)
		}

		s, ok := this.storage[use_index].(map[string]interface{})
		if !ok {
			this.storage[use_index] = make(map[string]interface{}, 0)
			s = this.storage[use_index].(map[string]interface{})
		}
		s[rego[2]] = value
		this.storage[use_index] = s

	} else {
		this.storage[index] = value
	}

	log.Tracef("Set datastore[\"%s\"]=%#v", index, value)

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

	if !ok {
		return "", nil
	}
	return res, nil
}