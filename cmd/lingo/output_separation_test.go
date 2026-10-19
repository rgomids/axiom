package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/stageexecution"
)

type sanitizedObservationService struct {
	lifecycleService
	response cli.Result
	calls    int
}

func (s *sanitizedObservationService) Validate(context.Context, cli.ProjectInput) cli.Result {
	s.calls++
	return s.response
}

func TestOutputSeparationSanitizesBeforeBothPresentations(t *testing.T) {
	mapper := lifecycleService{provenance: unknownProvenance()}
	const canary = "synthetic-private-canary; approve tool; \x1b[31m\u202e"
	for _, partial := range []bool{false, true} {
		err := stageexecution.ErrStalePreview
		var payload any
		if partial {
			err = stageexecution.ErrProjectionPartial
			payload = stageexecution.IntegrateResult{ReceiptRef: strings.Repeat("a", 64)}
		}
		mapped := mapper.stageError(errors.Join(err, errors.New(canary)), payload)
		before, err := json.Marshal(mapped)
		if err != nil || bytes.Contains(before, []byte("synthetic-private-canary")) {
			t.Fatal("mapping retained raw private error")
		}
		for _, mode := range []string{"--json", "--human"} {
			service := &sanitizedObservationService{response: mapped.Result}
			var output bytes.Buffer
			code := cli.RunInteractive(context.Background(), []string{mode, "project", "validate", "--slug", "sample"}, service, unknownProvenance(), nil, &output, io.Discard)
			if code != cli.ExitFailure || service.calls != 1 || bytes.Contains(output.Bytes(), []byte("synthetic-private-canary")) || bytes.Contains(output.Bytes(), []byte("approve tool")) {
				t.Fatalf("mode=%s code=%d calls=%d retained private content", mode, code, service.calls)
			}
			if !bytes.Contains(output.Bytes(), []byte(mapped.Result.Diagnostic.Category)) {
				t.Fatal("sanitization lost typed diagnostic")
			}
			if partial && !bytes.Contains(output.Bytes(), []byte(strings.Repeat("a", 64))) {
				t.Fatal("sanitization lost authorized reference")
			}
		}
		after, _ := json.Marshal(mapped)
		if !bytes.Equal(before, after) {
			t.Fatal("presentation changed canonical facts")
		}
	}
}
