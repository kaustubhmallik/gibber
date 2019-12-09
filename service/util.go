package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

func MapLowercaseKeys(v map[string]interface{}) map[string]interface{} {
	for key, val := range v {
		if val != nil && reflect.TypeOf(val).Kind() == reflect.Map { // nested map
			val = MapLowercaseKeys(val.(map[string]interface{}))
		}
		delete(v, key)
		if reflect.TypeOf(key).Kind() == reflect.String { // ObjectID key is not string, it is primitive.ObjectID
			v[strings.ToLower(key)] = val
		}
	}
	return v
}

func parseDOB(dobStr string) (dob time.Time, err error) {
	return
}

func GetMap(data interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	bytes, _ := json.Marshal(data)
	dataMap := make(map[string]interface{})
	_ = json.Unmarshal(bytes, &dataMap)
	return dataMap
}
