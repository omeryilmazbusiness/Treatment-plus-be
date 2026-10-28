package domain

import (
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Kind string

const (
	KindLogo   Kind = "logo"
	KindCover  Kind = "cover"
	KindAbout  Kind = "about"
	KindTeam   Kind = "team"
	KindDoctor Kind = "doctor"
	KindXray   Kind = "xray"
	KindLab    Kind = "lab"
	KindHotel  Kind = "hotel"
	KindBefore Kind = "before"
	KindAfter  Kind = "after"
	KindOther  Kind = "other"
)

func (k Kind) String() string { return string(k) }

func ParseKind(s string) (Kind, error) {
	switch Kind(strings.ToLower(strings.TrimSpace(s))) {
	case KindLogo, KindCover, KindAbout, KindTeam, KindDoctor, KindXray, KindLab, KindHotel, KindBefore, KindAfter, KindOther:
		return Kind(strings.ToLower(strings.TrimSpace(s))), nil
	case "":
		return KindOther, nil
	default:
		return "", errx.New("invalid_media_kind", "unsupported media kind", errx.ErrInvalid)
	}
}

const MaxBytes = 5 << 20

type Asset struct {
	ID          id.ID
	WorkspaceID id.ID
	Key         string
	Kind        Kind
	ContentType string
	Bytes       []byte
}

func DetectImage(raw []byte) string {
	if len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 {
		return "image/jpeg"
	}
	if len(raw) >= 8 && raw[0] == 0x89 && raw[1] == 0x50 && raw[2] == 0x4E && raw[3] == 0x47 {
		return "image/png"
	}
	return ""
}

func (a Asset) Validate() error {
	if a.WorkspaceID.IsZero() || a.ID.IsZero() {
		return errx.New("invalid_id", "workspace and asset id required", errx.ErrInvalid)
	}
	if strings.TrimSpace(a.Key) == "" {
		return errx.New("invalid_media_key", "object key is required", errx.ErrInvalid)
	}
	if len(a.Bytes) == 0 {
		return errx.New("empty_media", "image bytes are required", errx.ErrInvalid)
	}
	if len(a.Bytes) > MaxBytes {
		return errx.New("media_too_large", "image must be 5MB or smaller", errx.ErrInvalid)
	}
	if _, err := ParseKind(string(a.Kind)); err != nil {
		return err
	}
	switch a.ContentType {
	case "image/jpeg", "image/png":
		return nil
	default:
		return errx.New("invalid_media_type", "only jpeg and png are accepted", errx.ErrInvalid)
	}
}
