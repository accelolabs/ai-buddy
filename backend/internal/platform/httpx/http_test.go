package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecoderCompatibility(t *testing.T) {
	for _, tc := range []struct {
		body, want string
		valid      bool
	}{
		{`{"title":"T"}`, "T", true}, {`{"TITLE":"T"}`, "T", true},
		{`{"title":"first","title":"last"}`, "last", true},
		{`{"title":null}`, "", true}, {`null`, "", true},
		{`{"title":"T","owner":"other"}`, "", false}, {`{} {}`, "", false},
		{`{"title":`, "", false}, {``, "", false},
		{`{"title":"` + strings.Repeat("x", 1<<20) + `"}`, "", false},
	} {
		var input struct {
			Title string `json:"title"`
		}
		r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "text/plain") // Existing decoder does not require a media type.
		err := Decode(httptest.NewRecorder(), r, &input)
		if (err == nil) != tc.valid || (tc.valid && input.Title != tc.want) {
			t.Fatalf("valid=%v title=%q err=%v", tc.valid, input.Title, err)
		}
	}
}
