package httpserver

import (
	"encoding/json"
	"net/http"
)

func DecodeContextParams(req *http.Request, object any) error {
	params := GetContextParams(req)

	paramsJson, err := json.Marshal(params)
	if err != nil {
		return err
	}

	err = json.Unmarshal(paramsJson, object)
	if err != nil {
		return err
	}

	return nil
}

func GetContextParams(req *http.Request) map[string]any {
	params := req.Context().Value(paramsKey)
	if params == nil {
		return map[string]any{}
	}

	paramsMap, ok := params.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return paramsMap
}
