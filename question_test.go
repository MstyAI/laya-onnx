package layaonnx

import "testing"

func TestQuestionConstructorsRejectInvalidDefinitions(t *testing.T) {
	if _, err := NewChoice("", Option{Name: "a"}, Option{Name: "b"}); err == nil {
		t.Fatal("empty instructions were accepted")
	}
	if _, err := NewChoice("pick", Option{Name: "a"}); err == nil {
		t.Fatal("single choice was accepted")
	}
	if _, err := NewChoice("pick", Option{Name: "a"}, Option{Name: "a"}); err == nil {
		t.Fatal("duplicate choice was accepted")
	}
	if _, err := NewScore("rate", "only"); err == nil {
		t.Fatal("single score level was accepted")
	}
	if _, err := NewNoul("check", BooleanCriteria{}, BooleanCriteria{}); err == nil {
		t.Fatal("multiple boolean criteria were accepted")
	}
}

func TestQuestionAccessorsReturnCopies(t *testing.T) {
	question, err := NewChoice("pick", Option{Name: "a"}, Option{Name: "b"})
	if err != nil {
		t.Fatal(err)
	}
	options := question.Options()
	options[0].Name = "changed"
	if question.Options()[0].Name != "a" {
		t.Fatal("options accessor exposed internal state")
	}
}
