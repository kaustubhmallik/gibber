package server

import (
	"reflect"
	"strings"
	"time"
)

func MapLowercaseKeys(v map[string]interface{}) map[string]interface{} {
	for key, val := range v {
		if reflect.TypeOf(val).Kind() == reflect.Map { // nested map
			val = MapLowercaseKeys(val.(map[string]interface{}))
		}
		delete(v, key)
		v[strings.ToLower(key)] = val
	}
	return v
}

func parseDOB(dobStr string) (dob time.Time, err error) {
	return
}
