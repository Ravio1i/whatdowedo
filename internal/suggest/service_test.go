package suggest

import "testing"

func TestExtractJSON(t *testing.T) {
	in := "Sure! ```json\n{\"a\":1}\n```"
	if got := extractJSON(in); got != `{"a":1}` {
		t.Fatalf("got %q", got)
	}
}

func TestValidate(t *testing.T) {
	r := Request{Location: " Berlin ", Date: "2026-12-24", Tags: []string{" Kids", "", "outdoor"}}
	if _, err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if r.Location != "Berlin" || len(r.Tags) != 2 || r.Tags[0] != "kids" {
		t.Fatalf("normalisation failed: %+v", r)
	}
	if _, err := (&Request{Location: "x", Date: "24.12.2026"}).Validate(); err == nil {
		t.Fatal("expected date error")
	}
	if _, err := (&Request{}).Validate(); err == nil {
		t.Fatal("expected location error")
	}
}
