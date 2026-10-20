package executiongraph

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDispatchInputsGenerations(t *testing.T) {
	base := DispatchInputs{ProjectID: "project", ParentID: "00000000-0000-4000-8000-000000000001", PreviewDigest: strings.Repeat("a", 64), Preview: json.RawMessage(`{ "stagePlan" : {} }`)}
	for _, version := range []int{1, OperationalDispatchInputsVersion} {
		d := base
		d.FormatVersion = version
		wire, err := EncodeDispatchInputs(d)
		if err != nil {
			t.Fatal(version, err)
		}
		decoded, err := DecodeDispatchInputs(wire)
		if err != nil || decoded.FormatVersion != version || !bytes.Equal(decoded.Preview, []byte(`{"stagePlan":{}}`)) {
			t.Fatal("generation not preserved", version, err)
		}
	}
	for _, version := range []int{0, 3} {
		d := base
		d.FormatVersion = version
		if _, err := EncodeDispatchInputs(d); err == nil {
			t.Fatal("unknown generation accepted", version)
		}
	}
	d := base
	d.FormatVersion = OperationalDispatchInputsVersion
	wire, _ := EncodeDispatchInputs(d)
	if _, err := DecodeDispatchInputs(bytes.Replace(wire, []byte(`"parentId"`), []byte(`"unknown":1,"parentId"`), 1)); err == nil {
		t.Fatal("unknown member accepted")
	}
}
