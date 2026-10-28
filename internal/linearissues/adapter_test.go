package linearissues

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workitem"
)

func TestValidResourceAndExternal(t *testing.T) {
	a := Adapter{}
	if !a.ValidResource("AXM") || a.ValidResource("axm") || a.ValidResource("A-X") || a.ValidResource("") {
		t.Fatal("team key validation")
	}
	external := workitem.External{ID: "AXM-39", URL: "https://linear.app/rgomids/issue/AXM-39", State: workitem.OpenState}
	if !a.ValidExternal("AXM", "AXM-39", external) {
		t.Fatal("expected valid external")
	}
	external.URL = "https://linear.app/rgomids/issue/AXM-39/title"
	if !a.ValidExternal("AXM", "AXM-39", external) {
		t.Fatal("suffix path should be valid")
	}
	external.URL = "https://github.com/rgomids/axiom/issues/39"
	if a.ValidExternal("AXM", "AXM-39", external) {
		t.Fatal("foreign host must fail")
	}
	if a.ValidExternal("AXM", "39", external) || a.ValidExternal("ENG", "AXM-39", workitem.External{ID: "AXM-39", URL: "https://linear.app/w/issue/AXM-39", State: workitem.OpenState}) {
		t.Fatal("mismatched resource/id must fail")
	}
}

func TestCreateReadViaFakeBin(t *testing.T) {
	bin := writeLinearBin(t, map[string]any{
		`teams(filter: { key: { eq: $key } }, first: 2)`: map[string]any{
			"teams": map[string]any{"nodes": []any{map[string]any{"id": "team-1", "key": "AXM"}}},
		},
		`issueCreate`: map[string]any{
			"issueCreate": map[string]any{
				"success": true,
				"issue": map[string]any{
					"id": "issue-1", "identifier": "AXM-39",
					"url":         "https://linear.app/rgomids/issue/AXM-39",
					"state":       map[string]any{"type": "unstarted"},
					"description": "<!-- axiom:work-item-draft:" + strings.Repeat("a", 64) + " -->\n",
				},
			},
		},
		`issue(id: $id)`: map[string]any{
			"issue": map[string]any{
				"id": "issue-1", "identifier": "AXM-39",
				"url":   "https://linear.app/rgomids/issue/AXM-39",
				"state": map[string]any{"type": "started"},
				"title": "T", "description": "body",
				"labels": map[string]any{"nodes": []any{}},
			},
		},
	})
	adapter, err := New(bin)
	if err != nil {
		t.Fatal(err)
	}
	created, err := adapter.Create(context.Background(), workitem.CreateRequest{
		Resource:    "AXM",
		Correlation: strings.Repeat("a", 64),
		Document:    workitem.ProviderDocument{Title: "T", Body: "body"},
	})
	if err != nil || created.ID != "AXM-39" || created.State != workitem.OpenState {
		t.Fatalf("create = %#v err=%v", created, err)
	}
	read, err := adapter.Read(context.Background(), "AXM", "AXM-39")
	if err != nil || read.ID != "AXM-39" || read.URL != created.URL {
		t.Fatalf("read = %#v err=%v", read, err)
	}
}

func TestHTTPRequiresAPIKey(t *testing.T) {
	adapter, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	adapter.getenv = func(string) string { return "" }
	adapter.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("http must not run without key")
		return nil, nil
	})}
	_, err = adapter.Read(context.Background(), "AXM", "AXM-39")
	var provider *workitem.ProviderError
	if err == nil || !asProvider(err, &provider) || provider.Kind != workitem.ProviderUnauthenticated {
		t.Fatalf("err=%v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func asProvider(err error, target **workitem.ProviderError) bool {
	if pe, ok := err.(*workitem.ProviderError); ok {
		*target = pe
		return true
	}
	return false
}

func writeLinearBin(t *testing.T, routes map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	routesPath := filepath.Join(dir, "routes.json")
	payload, err := json.Marshal(routes)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(routesPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "linear-bin")
	content := "#!/usr/bin/env python3\nimport json,sys\nroutes=json.load(open(" + strconv.Quote(routesPath) + "))\nreq=json.load(sys.stdin)\nq=req.get(\"query\",\"\")\nfor key, data in routes.items():\n    if key in q:\n        json.dump({\"data\": data}, sys.stdout)\n        sys.exit(0)\njson.dump({\"errors\":[{\"message\":\"unmatched query\"}]}, sys.stdout)\nsys.exit(1)\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}
func TestDecodeEnvelopeLimits(t *testing.T) {
	_, err := decodeEnvelope([]byte(`{"data":{"ok":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = decodeEnvelope(bytes.Repeat([]byte("x"), 10))
	if err == nil {
		t.Fatal("expected invalid")
	}
	_ = io.EOF
}
