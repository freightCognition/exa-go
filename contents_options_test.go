package exa

import (
	"encoding/json"
	"testing"
)

func init() {
	registerSchema("ContentsOptions", ContentsOptions{})
	registerSchema("CostDollarsOutput", CostDollars{})
	registerSchema("OutputSchemaObject", OutputSchema{})
	// The text variant is the same struct with only type/description set.
	registerSchema("OutputSchemaText", OutputSchema{}, "properties", "required", "additionalProperties")
	registerSchema("OutputSchemaTextOutput", OutputSchema{}, "properties", "required", "additionalProperties")
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEmptyOptionsMarshalAsTrue(t *testing.T) {
	got := marshal(t, ContentsOptions{Text: &TextOptions{}, Highlights: &HighlightsOptions{}, Summary: &SummaryOptions{}, Context: &ContextOptions{}})
	want := `{"text":true,"highlights":true,"summary":true,"context":true}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestSetOptionsMarshalAsObjects(t *testing.T) {
	got := marshal(t, ContentsOptions{
		Text:        &TextOptions{MaxCharacters: Ptr(500), Verbosity: TextVerbosityFull, ExcludeSections: []Section{SectionNavigation}},
		Highlights:  &HighlightsOptions{Query: "pricing", MaxCharacters: Ptr(200)},
		Summary:     &SummaryOptions{Query: "what do they sell"},
		MaxAgeHours: Ptr(0),
	})
	want := `{"text":{"maxCharacters":500,"verbosity":"full","excludeSections":["navigation"]},"highlights":{"query":"pricing","maxCharacters":200},"summary":{"query":"what do they sell"},"maxAgeHours":0}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestUnsetOptionsAreOmitted(t *testing.T) {
	if got := marshal(t, ContentsOptions{}); got != `{}` {
		t.Fatalf("got %s", got)
	}
}

func TestIncludeHTMLTagsFalseIsAnObject(t *testing.T) {
	// A set pointer to false is a real choice, not "empty".
	if got := marshal(t, TextOptions{IncludeHTMLTags: Ptr(false)}); got != `{"includeHtmlTags":false}` {
		t.Fatalf("got %s", got)
	}
}

// Responses (e.g. a Search Monitor's saved contents) echo these options back
// as true, false or an object; all three must decode and re-encode unchanged.
func TestOptionsRoundTripTrueFalseAndObjects(t *testing.T) {
	in := `{"text":true,"highlights":{"query":"pricing"},"summary":false,"context":{"maxCharacters":10}}`
	var o ContentsOptions
	if err := json.Unmarshal([]byte(in), &o); err != nil {
		t.Fatal(err)
	}
	if o.Text == nil || o.Highlights == nil || o.Highlights.Query != "pricing" || o.Summary == nil || o.Context == nil {
		t.Fatalf("decoded = %+v", o)
	}
	if got := marshal(t, o); got != in {
		t.Fatalf("round trip\n got  %s\n want %s", got, in)
	}
}

func TestMonitorContentsDecodeTrue(t *testing.T) {
	var m SearchMonitorContents
	if err := json.Unmarshal([]byte(`{"text":true,"summary":{"query":"q"},"highlights":false}`), &m); err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, m); got != `{"text":true,"highlights":false,"summary":{"query":"q"}}` {
		t.Fatalf("got %s", got)
	}
}

func TestOptionsRejectGarbage(t *testing.T) {
	var o TextOptions
	if err := json.Unmarshal([]byte(`"yes"`), &o); err == nil {
		t.Fatal("expected an error for a string")
	}
}

func TestOutputSchema(t *testing.T) {
	if got := marshal(t, OutputSchema{Type: OutputSchemaTypeText, Description: "one paragraph"}); got != `{"type":"text","description":"one paragraph"}` {
		t.Errorf("text schema = %s", got)
	}
	got := marshal(t, OutputSchema{
		Type:       OutputSchemaTypeObject,
		Properties: map[string]any{"name": map[string]any{"type": "string"}},
		Required:   []string{"name"},
	})
	if got != `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}` {
		t.Errorf("object schema = %s", got)
	}
}

func TestCostDollarsDecodes(t *testing.T) {
	var c CostDollars
	if err := json.Unmarshal([]byte(`{"total":0.012,"search":{"neural":0.005},"contents":{"text":0.001,"summary":0.006}}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Total != 0.012 || c.Search == nil || c.Search.Neural != 0.005 || c.Contents == nil || c.Contents.Summary != 0.006 {
		t.Fatalf("cost = %+v", c)
	}
}
