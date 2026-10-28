package app

import (
	"context"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type WorkspaceReader interface {
	GetVisible(ctx context.Context, id id.ID) (domain.Workspace, error)
}

type AccountLookup interface {
	FindByEmail(ctx context.Context, email string) (domain.Account, error)
	FindByWorkspaceID(ctx context.Context, workspaceID id.ID) (domain.Account, error)
}

type SessionReader interface {
	GetByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error)
}

type WriteTx interface {
	CreateAccount(ctx context.Context, account domain.Account) error
	SaveSession(ctx context.Context, session domain.Session) error
	MarkReused(ctx context.Context, sessionID id.ID, at time.Time) error
	RevokeFamily(ctx context.Context, familyID id.ID, at time.Time) error
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx WriteTx) error) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) error
}

type AccessIssuer interface {
	Issue(userID, workspaceID id.ID, role domain.Role, now time.Time) (token string, ttl time.Duration, err error)
}

// ClinicProfile is the one-time PDF brand block. Identity does not own clinic tables.
type ClinicStat struct {
	Value string
	Label string
}

type ClinicDoctor struct {
	Name, Title, Subtitle, Bio, PhotoKey string
	Stats                                []ClinicStat
}

type ClinicLab struct {
	Title, Text, ImageKey string
	Stats                 []ClinicStat
}

type ClinicAwards struct {
	Text      string
	BadgeKeys []string
}

type ClinicHotel struct {
	Name, Text, ImageKey string
}

type ClinicProfile struct {
	Name         string
	Slogan       string
	Phone        string
	Email        string
	Website      string
	Address      string
	Hours        string
	Locale       string
	Currency     string
	Palette      string
	LogoKey      string
	CoverKey     string
	MapURL       string
	Intro        string
	Legal        string
	AboutText    string
	AboutKey     string
	WelcomeBody  string
	SignerName   string
	SignerTitle  string
	Highlights   []ClinicStat
	Doctor       ClinicDoctor
	Lab          ClinicLab
	Awards       ClinicAwards
	Hotel        ClinicHotel
	RequireBrand bool
}

// RequiredClinic is the minimum Wise contact + welcome block collected at clinic register.
func RequiredClinic(phone string) ClinicProfile {
	return ClinicProfile{
		Phone:       phone,
		Email:       "info@clinic.test",
		Address:     "Test Cad. No:1",
		Hours:       "09:00-18:00",
		Slogan:      "Every Smile Matters",
		SignerName:  "Dr. Test",
		SignerTitle: "Klinik Direktörü",
	}
}

type ProfileProvisioner interface {
	Provision(ctx context.Context, workspaceID id.ID, profile ClinicProfile) error
}

type ProfileFunc func(ctx context.Context, workspaceID id.ID, profile ClinicProfile) error

func (f ProfileFunc) Provision(ctx context.Context, workspaceID id.ID, profile ClinicProfile) error {
	return f(ctx, workspaceID, profile)
}
