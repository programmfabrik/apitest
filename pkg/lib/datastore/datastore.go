package datastore

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type Datastore struct {
	storage      map[string]any // custom storage
	responseJson []string       // store the responses
	logDatastore bool
	lock         *sync.RWMutex
}

func NewStore(logDatastore bool) *Datastore {
	ds := Datastore{}
	ds.storage = make(map[string]any, 0)
	ds.responseJson = make([]string, 0)
	ds.logDatastore = logDatastore
	ds.lock = &sync.RWMutex{}
	return &ds
}

type datastoreKeyNotFoundError struct {
	error string
}

func (data datastoreKeyNotFoundError) Error() string {
	return data.error
}

type datastoreIndexOutOfBoundsError struct {
	error string
}

func (data datastoreIndexOutOfBoundsError) Error() string {
	return data.error
}

type datastoreIndexError struct {
	error string
}

func (data datastoreIndexError) Error() string {
	return data.error
}

// SetWithGjson stores the given response driven by a map key => gjson
func (ds *Datastore) SetWithGjson(jsonResponse string, storeResponse map[string]string) (err error) {
	for k, qv := range storeResponse {
		setEmpty := false
		if len(qv) > 0 && qv[0] == '!' {
			setEmpty = true
			qv = qv[1:]
		}
		qValue := gjson.Get(jsonResponse, qv)
		if qValue.Value() == nil {
			if ds.logDatastore {
				logrus.Tracef("'%s' was not found in '%s'", qv, jsonResponse)
			}
			// Remove value from datastore
			if setEmpty {
				ds.delete(k)
			}
			continue
		}
		err := ds.Set(k, qValue.Value())
		if err != nil {
			return err
		}
	}
	return nil
}
func (ds *Datastore) delete(k string) {
	delete(ds.storage, k)
}

// We store the response
func (ds *Datastore) UpdateLastResponse(s string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	ds.responseJson[len(ds.responseJson)-1] = s
}

// We store the response
func (ds *Datastore) AppendResponse(s string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	ds.responseJson = append(ds.responseJson, s)
}

func (ds *Datastore) SetMap(smap map[string]any) (err error) {
	for k, v := range smap {
		err := ds.Set(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *Datastore) Set(index string, value any) (err error) {
	var dsMapRegex = regexp.MustCompile(`^(.*?)\[(.+?)\]$`)

	//typeswitch for checking if float is actually int
	switch t := value.(type) {
	case float64:
		if math.Mod(t, 1.0) == 0 {
			//is int
			value = int(t)
		}
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	//Slice in datastore
	if strings.HasSuffix(index, "[]") {
		// do a push to an array
		use_index := index[:len(index)-2]
		_, ok := ds.storage[use_index]
		if !ok {
			ds.storage[use_index] = make([]any, 0)
		}

		s, ok := ds.storage[use_index].([]any)
		if !ok {
			tmp := ds.storage[use_index]
			ds.storage[use_index] = make([]any, 0)
			s = ds.storage[use_index].([]any)

			if tmp != nil {
				ds.storage[use_index] = append(s, tmp)
			}
		}

		ds.storage[use_index] = append(s, value)

	} else if rego := dsMapRegex.FindStringSubmatch(index); len(rego) > 0 {
		// do a push to an array
		use_index := rego[1]
		_, ok := ds.storage[use_index]
		if !ok {
			ds.storage[use_index] = make(map[string]any, 0)
		}

		s, ok := ds.storage[use_index].(map[string]any)
		if !ok {
			ds.storage[use_index] = make(map[string]any, 0)
			s = ds.storage[use_index].(map[string]any)
		}
		s[rego[2]] = value
		ds.storage[use_index] = s

	} else {
		ds.storage[index] = value
	}

	if ds.logDatastore {
		logrus.Tracef("Set datastore[%q]=%#v", index, value)
	}

	return nil
}

func (ds Datastore) Get(index string) (res any, err error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	// strings are evalulated as int, so
	// that we can support "-<int>" notations

	if index == "-" {
		// return the entire custom store
		return ds.storage, nil
	}

	var dsMapRegex = regexp.MustCompile(`^(.*?)\[(.+?)\]$`)

	if rego := dsMapRegex.FindStringSubmatch(index); len(rego) > 0 {
		//we have a map or slice
		useIndex := rego[1]
		mapIndex := rego[2]

		tmpRes, ok := ds.storage[useIndex]
		if !ok {
			logrus.Errorf("datastore: key: %s not found.", useIndex)
			return "", datastoreKeyNotFoundError{error: fmt.Sprintf("datastore: key: %s not found.", useIndex)}
		}

		tmpResMap, ok := tmpRes.(map[string]any)
		if ok {
			//We have a map
			mapVal, ok := tmpResMap[mapIndex]
			if !ok {
				//Value not found in map, so return empty string
				return "", nil
			} else {
				return mapVal, nil
			}
		}

		tmpResSlice, ok := tmpRes.([]any)
		if ok {
			//We have a slice
			sliceIdx, err := strconv.Atoi(mapIndex)
			if err != nil {
				return "", datastoreIndexError{error: fmt.Sprintf("datastore: could not convert key to int: %s", mapIndex)}
			}

			if sliceIdx < 0 {
				sliceIdx = len(tmpResSlice) + sliceIdx
			}

			if len(tmpResSlice) <= sliceIdx || sliceIdx < 0 {
				return "", nil
			} else {
				return tmpResSlice[sliceIdx], nil
			}
		}

	} else {

		idx, err := strconv.Atoi(index)
		if err == nil {
			if idx < 0 {
				idx = idx + len(ds.responseJson)
			}
			if idx >= len(ds.responseJson) || idx < 0 {
				// index out of range

				return "", datastoreIndexOutOfBoundsError{error: fmt.Sprintf("datastore.Get: idx out of range: %d, current length: %d", idx, len(ds.responseJson))}
			}
			return ds.responseJson[idx], nil
		}
		var ok bool
		res, ok = ds.storage[index]
		if ok {
			return res, nil
		}
	}

	return "", nil
}
