package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

// AdminAccount is one entry of ADMIN_ACCOUNTS (see config.AdminConfig).
type AdminAccount struct {
	Email        string
	PasswordHash string // PHC-formatted, as HashPassword makes it
}

// ParseAdminAccounts reads ADMIN_ACCOUNTS: a comma-separated list of
// email:hash, the hash being base64 of a HashPassword result (what
// `go run ./cmd/hashpassword` prints). Empty input means no admins.
func ParseAdminAccounts(raw string) ([]AdminAccount, error) {
	var accounts []AdminAccount
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		email, encoded, ok := strings.Cut(entry, ":")
		if !ok {
			return nil, fmt.Errorf("entry %q: want email:hash", entry)
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, fmt.Errorf("entry %q: invalid email", email)
		}
		hash, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("entry %q: hash is not base64", email)
		}
		if !strings.HasPrefix(string(hash), "$argon2id$") {
			return nil, fmt.Errorf("entry %q: not a password hash from cmd/hashpassword", email)
		}
		accounts = append(accounts, AdminAccount{Email: email, PasswordHash: string(hash)})
	}
	return accounts, nil
}

// EnsureAdmins makes sure each account exists, signs in with its configured
// password and has the admin role: missing ones are registered (starting
// diamonds included), existing ones are updated. It runs on every start, so
// changing ADMIN_ACCOUNTS and restarting is all a password change takes.
// Accounts taken out of the list keep their role.
func (s *Service) EnsureAdmins(ctx context.Context, accounts []AdminAccount) error {
	for _, a := range accounts {
		user, err := s.repo.GetUserByEmail(ctx, a.Email)
		if errors.Is(err, ErrUserNotFound) {
			user, err = s.repo.CreateUser(ctx, a.Email, a.PasswordHash)
			if err == nil && s.bonus != nil {
				err = s.bonus.GrantSignupBonus(ctx, user.ID, s.bonusAmount)
			}
		}
		if err != nil {
			return fmt.Errorf("admin %s: %w", a.Email, err)
		}
		if err := s.repo.MakeAdmin(ctx, user.ID, a.PasswordHash); err != nil {
			return fmt.Errorf("admin %s: %w", a.Email, err)
		}
	}
	return nil
}
