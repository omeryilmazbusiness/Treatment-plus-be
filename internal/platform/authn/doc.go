// Package authn authenticates clinic accounts (Argon2id + JWT access + opaque refresh).
// Go never writes cookies; the BFF stores tokens.
package authn
