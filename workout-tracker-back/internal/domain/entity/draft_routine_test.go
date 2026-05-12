package entity

import (
	"encoding/json"
	"testing"
)

func TestFlexString_UnmarshalFromString(t *testing.T) {
	var f FlexString
	if err := json.Unmarshal([]byte(`"1-2"`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != "1-2" {
		t.Errorf("want %q got %q", "1-2", f)
	}
}

func TestFlexString_UnmarshalFromInteger(t *testing.T) {
	var f FlexString
	if err := json.Unmarshal([]byte(`2`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != "2" {
		t.Errorf("want %q got %q", "2", f)
	}
}

func TestFlexString_UnmarshalFromFloat(t *testing.T) {
	var f FlexString
	if err := json.Unmarshal([]byte(`1.5`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != "1.5" {
		t.Errorf("want %q got %q", "1.5", f)
	}
}

func TestFlexString_UnmarshalFromNull(t *testing.T) {
	var f FlexString
	if err := json.Unmarshal([]byte(`null`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != "" {
		t.Errorf("want empty got %q", f)
	}
}

func TestFlexString_MarshalAlwaysString(t *testing.T) {
	f := FlexString("2")
	out, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal err: %v", err)
	}
	if string(out) != `"2"` {
		t.Errorf("want %q got %q", `"2"`, string(out))
	}
}

func TestDraftExercise_Unmarshal_RIR_AsInt(t *testing.T) {
	payload := []byte(`{
		"exercise_id":"ex_032","spanish_name":"Sentadilla","pattern":"squat","role":"compound",
		"sets":3,"reps":"8-10","rir":2,"rest_seconds":120,"exercise_order":1,"alternatives":[]
	}`)
	var ex DraftExercise
	if err := json.Unmarshal(payload, &ex); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if ex.RIR != "2" {
		t.Errorf("RIR want %q got %q", "2", ex.RIR)
	}
}

func TestDraftExercise_Unmarshal_RIR_AsString(t *testing.T) {
	payload := []byte(`{
		"exercise_id":"ex_032","spanish_name":"Sentadilla","pattern":"squat","role":"compound",
		"sets":3,"reps":"8-10","rir":"1-2","rest_seconds":120,"exercise_order":1,"alternatives":[]
	}`)
	var ex DraftExercise
	if err := json.Unmarshal(payload, &ex); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if ex.RIR != "1-2" {
		t.Errorf("RIR want %q got %q", "1-2", ex.RIR)
	}
}

func TestDraftAlternative_Unmarshal_LegacyLinkKey(t *testing.T) {
	payload := []byte(`{
		"exercise_id":"ex_002","spanish_name":"Sentadilla Hack",
		"main_muscle":"Quads","link":"https://example.com/v"
	}`)
	var alt DraftAlternative
	if err := json.Unmarshal(payload, &alt); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if alt.VideoLink != "https://example.com/v" {
		t.Errorf("VideoLink want %q got %q", "https://example.com/v", alt.VideoLink)
	}
}

func TestDraftAlternative_Unmarshal_CanonicalVideoLinkKey(t *testing.T) {
	payload := []byte(`{
		"exercise_id":"ex_002","spanish_name":"Sentadilla Hack",
		"main_muscle":"Quads","video_link":"https://example.com/canonical"
	}`)
	var alt DraftAlternative
	if err := json.Unmarshal(payload, &alt); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if alt.VideoLink != "https://example.com/canonical" {
		t.Errorf("VideoLink want %q got %q", "https://example.com/canonical", alt.VideoLink)
	}
}

func TestDraftAlternative_Unmarshal_PrefersCanonicalWhenBothPresent(t *testing.T) {
	payload := []byte(`{
		"exercise_id":"ex_002","spanish_name":"X",
		"video_link":"https://canonical","link":"https://legacy"
	}`)
	var alt DraftAlternative
	if err := json.Unmarshal(payload, &alt); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if alt.VideoLink != "https://canonical" {
		t.Errorf("expected canonical key to win, got %q", alt.VideoLink)
	}
}

func TestDraftData_Unmarshal_RealWorldBadPayload(t *testing.T) {
	// Mirrors the actual broken draft "0249b8": rir as int, alternatives with `link`.
	payload := []byte(`{
		"week_schedule":"fb_2","goal":"Ganar masa muscular","level":"Intermedio",
		"days":[{
			"day_number":1,"title":"Full Body A","exercises":[{
				"exercise_id":"ex_032","spanish_name":"Sentadilla búlgara","pattern":"squat",
				"role":"compound","sets":3,"reps":"8-10","rir":2,"rest_seconds":120,
				"exercise_order":1,
				"alternatives":[{
					"link":"https://musclewiki.com/es-es/exercise/machine-hack-squat",
					"exercise_id":"ex_002","main_muscle":"Quads","spanish_name":"Sentadilla Hack"
				}]
			}]
		}]
	}`)
	var d DraftData
	if err := json.Unmarshal(payload, &d); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	ex := d.Days[0].Exercises[0]
	if ex.RIR != "2" {
		t.Errorf("RIR want %q got %q", "2", ex.RIR)
	}
	if got := ex.Alternatives[0].VideoLink; got != "https://musclewiki.com/es-es/exercise/machine-hack-squat" {
		t.Errorf("alternative VideoLink not populated from legacy `link`, got %q", got)
	}
}
