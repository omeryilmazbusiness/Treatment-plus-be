package authn

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const issuer = "treatment-plus"

type Principal struct {
	UserID      id.ID
	WorkspaceID id.ID
	Role        domain.Role
}

type Claims struct {
	UserID   string `json:"user_id"`
	ClinicID string `json:"clinic_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret []byte
	ttl    time.Duration
	clock  clock.Clock
}

func NewJWT(cfg config.Auth, clk clock.Clock) (*JWT, error) {
	if len(cfg.JWTAccessSecret) < 32 {
		return nil, fmt.Errorf("authn: JWT_ACCESS_SECRET must be at least 32 bytes")
	}
	if clk == nil {
		clk = clock.System{}
	}
	ttl := cfg.AccessTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &JWT{secret: []byte(cfg.JWTAccessSecret), ttl: ttl, clock: clk}, nil
}

func (j *JWT) Issue(userID, workspaceID id.ID, role domain.Role, now time.Time) (string, time.Duration, error) {
	if now.IsZero() {
		now = j.clock.Now()
	}
	claims := Claims{
		UserID:   userID.String(),
		ClinicID: workspaceID.String(),
		Role:     string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			ID:        id.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", 0, fmt.Errorf("authn: sign: %w", err)
	}
	return signed, j.ttl, nil
}

func (j *JWT) ParseAccess(raw string) (Principal, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected method")
		}
		return j.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithTimeFunc(j.clock.Now), jwt.WithLeeway(5*time.Second))
	if err != nil || !token.Valid {
		return Principal{}, errx.ErrUnauthenticated
	}
	role, err := domain.ParseRole(claims.Role)
	if err != nil {
		return Principal{}, errx.ErrUnauthenticated
	}
	userID, err := id.Parse(firstNonEmpty(claims.UserID, claims.Subject))
	if err != nil {
		return Principal{}, errx.ErrUnauthenticated
	}
	workspaceID, err := id.Parse(firstNonEmpty(claims.ClinicID, claims.Subject))
	if err != nil {
		return Principal{}, errx.ErrUnauthenticated
	}
	return Principal{UserID: userID, WorkspaceID: workspaceID, Role: role}, nil
}

func Bearer(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
