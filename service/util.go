package service

import (
	"encoding/json"
)

func GetMap(data interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	bytes, _ := json.Marshal(data)
	dataMap := make(map[string]interface{})
	_ = json.Unmarshal(bytes, &dataMap)
	return dataMap
}
