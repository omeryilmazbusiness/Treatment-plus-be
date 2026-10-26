package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

func TestWriteErrorMapsDomainErrors(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{errx.ErrNotFound, http.StatusNotFound, "not_found"},
		{errx.ErrForbidden, http.StatusForbidden, "forbidden"},
		{errx.ErrConflict, http.StatusConflict, "conflict"},
		{errx.ErrInvalid, http.StatusUnprocessableEntity, "invalid"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		WriteError(rec, tc.err)
		if rec.Code != tc.status {
			t.Fatalf("%v: status %d", tc.err, rec.Code)
		}
		var body errorBody
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Error.Code != tc.code {
			t.Fatalf("code %s", body.Error.Code)
		}
	}
}
