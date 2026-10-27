package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return errx.New("invalid_body", "request body is required", errx.ErrInvalid)
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errx.New("invalid_body", "request body is required", errx.ErrInvalid)
		}
		return errx.New("invalid_body", "invalid json", fmt.Errorf("%w: %v", errx.ErrInvalid, err))
	}
	return nil
}
