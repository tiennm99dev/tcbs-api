package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

const sample = `{
  "paths": {
    "/accounts/{accountNo}/se": {
      "get": {
        "parameters": [
          {"name": "accountNo", "in": "path"},
          {"name": "symbol", "in": "path"},
          {"name": "page", "in": "query"}
        ],
        "responses": {"200": {"content": {"application/json": {"schema": {
          "type": "object",
          "required": ["assets", "assets[].symbol"],
          "properties": {
            "assets[].symbol": {"type": "string"},
            "assets": {"type": "array", "description": "Holdings."},
            "assets[].lots[].qty": {"type": "integer"},
            "meta.page": {"type": "integer"},
            "total": {"type": "integer"}
          }
        }}}}}
      }
    }
  }
}`

func TestFix(t *testing.T) {
	spec := decode(t, sample)
	rep, err := Fix(spec, map[string]string{"GET /accounts/{accountNo}/se": "StockAsset"})
	if err != nil {
		t.Fatal(err)
	}
	if rep != (Report{Nested: 3, DroppedParams: 1, OperationIDs: 1}) {
		t.Errorf("report = %+v", rep)
	}

	op := spec["paths"].(map[string]any)["/accounts/{accountNo}/se"].(map[string]any)["get"].(map[string]any)
	if op["operationId"] != "StockAsset" {
		t.Errorf("operationId = %v", op["operationId"])
	}
	if n := len(op["parameters"].([]any)); n != 2 {
		t.Errorf("parameters = %d, want accountNo and page", n)
	}

	schema := op["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"]
	want := decode(t, `{
	  "type": "object",
	  "required": ["assets"],
	  "properties": {
	    "assets": {"type": "array", "description": "Holdings.", "items": {"type": "object", "properties": {
	      "symbol": {"type": "string"},
	      "lots": {"type": "array", "items": {"type": "object", "properties": {"qty": {"type": "integer"}}}}
	    }}},
	    "meta": {"type": "object", "properties": {"page": {"type": "integer"}}},
	    "total": {"type": "integer"}
	  }
	}`)
	if !reflect.DeepEqual(schema, want) {
		got, _ := json.MarshalIndent(schema, "", " ")
		t.Errorf("schema:\n%s", got)
	}
}

func TestFixOperationIDMapMustMatch(t *testing.T) {
	_, err := Fix(decode(t, sample), map[string]string{"GET /gone": "Gone"})
	if err == nil || !strings.Contains(err.Error(), "GET /accounts/{accountNo}/se") || !strings.Contains(err.Error(), "GET /gone") {
		t.Fatalf("err = %v", err)
	}
}
