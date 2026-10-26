// Package domain is media metadata. Public ACL is forbidden; objects live in private R2/S3.
package domain

import "github.com/treatment-plus/treatment-plus-be/internal/shared/id"

type Kind string

const (
	KindLogo     Kind = "logo"
	KindTeam     Kind = "team"
	KindXray     Kind = "xray"
	KindBefore   Kind = "before"
	KindAfter    Kind = "after"
	KindVideo    Kind = "video"
)

type Asset struct {
	ID       id.ID
	ClinicID id.ID
	Key      string
	Kind     Kind
	Checksum string
}
