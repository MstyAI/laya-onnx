package layaonnx

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodeRequestPreservesChoiceOrder(t *testing.T) {
	request, err := DecodeRequest(strings.NewReader(`{
        "state":{"message":"hello"},
        "questions":{"route":{"type":"choice","instructions":"Pick","criteria":{"z":"last","a":null,"m":"middle"}}}
    }`))
	if err != nil {
		t.Fatal(err)
	}
	options := request.Questions["route"].Options()
	if len(options) != 3 || options[0].Name != "z" || options[1].Name != "a" || options[2].Name != "m" {
		t.Fatalf("choice order = %+v", options)
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"criteria":{"z":"last","a":null,"m":"middle"}`)) {
		t.Fatalf("round trip changed criteria: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"state":{"message":"hello"}`)) {
		t.Fatalf("round trip changed state: %s", raw)
	}
}

func TestDecodeRequestPreservesStructuredStateOrder(t *testing.T) {
	request, err := DecodeRequest(strings.NewReader(`{
        "state":{"z":[1,true],"a":"<keep>"},
        "questions":{"q":{"type":"noul","instructions":"Check"}}
    }`))
	if err != nil {
		t.Fatal(err)
	}
	state, err := serializeState(request.State)
	if err != nil {
		t.Fatal(err)
	}
	if state != `{"z": [1, true], "a": "<keep>"}` {
		t.Fatalf("state = %q", state)
	}
}

func TestDecodeRequestKeepsTextStateUnquoted(t *testing.T) {
	request, err := DecodeRequest(strings.NewReader(`{
        "state":"plain text",
        "questions":{"q":{"type":"noul","instructions":"Check"}}
    }`))
	if err != nil {
		t.Fatal(err)
	}
	state, err := serializeState(request.State)
	if err != nil {
		t.Fatal(err)
	}
	if state != "plain text" {
		t.Fatalf("state = %q", state)
	}
}

func TestDecodeRequestRejectsInvalidWireShapes(t *testing.T) {
	for _, input := range []string{
		`{}`,
		`{"state":"x","questions":{"q":{"type":"unknown","instructions":"x"}}}`,
		`{"state":"x","questions":{"q":{"type":"score","instructions":"x","criteria":"bad"}}}`,
		`{"state":"x","questions":{"q":{"type":"noul","instructions":"x","extra":true}}}`,
	} {
		if _, err := DecodeRequest(strings.NewReader(input)); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("input %s returned %v", input, err)
		}
	}
}
