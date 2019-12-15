package service

import (
	"encoding/json"
)

func GetMap(data interface{}) (dataMap map[string]interface{}, err error) {
	if data == nil {
		return
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &dataMap)
	return
}
