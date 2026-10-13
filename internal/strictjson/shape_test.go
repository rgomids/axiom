package strictjson

import "testing"

func TestExactMembersAndScalarNull(t *testing.T) {
	type record struct {
		Count    int    `json:"count"`
		Enabled  bool   `json:"enabled"`
		Optional string `json:"optional,omitempty"`
	}
	for _, wire := range []string{`{"count":1,"enabled":false}`, ` { "enabled":true,"count":2,"optional":"value" } `} {
		var r record
		if err := Decode([]byte(wire), &r, 1024); err != nil {
			t.Fatal(wire, err)
		}
	}
	for _, wire := range []string{`{"Count":1,"enabled":false}`, `{"count":1,"enabled":null}`, `{"count":1}`, `{"count":1,"enabled":false,"optional":null}`, `{"count":1,"enabled":false,"count":2}`, `{"count":1,"enabled":false,"extra":0}`} {
		var r record
		if err := Decode([]byte(wire), &r, 1024); err == nil {
			t.Fatal("accepted", wire)
		}
	}
}
