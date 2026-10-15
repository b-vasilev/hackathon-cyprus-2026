// Package access manages portal accounts, sessions and per-product roles.
package access

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Global roles. Admins have full access; users only reach products they are members of.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// Product roles. Leads manage a product and its members; learners read it.
const (
	RoleLead    = "lead"
	RoleLearner = "learner"
)

// SessionLifetime bounds how long a sign-in stays valid.
const SessionLifetime = 30 * 24 * time.Hour

var (
	// ErrInvalid identifies invalid account or membership input.
	ErrInvalid = errors.New("invalid input")
	// ErrCredentials hides whether the email or the password was wrong.
	ErrCredentials = errors.New("incorrect email or password")
	// ErrConflict identifies a duplicate account email.
	ErrConflict = errors.New("an account with this email already exists")
	// ErrLastAdmin prevents removing the final administrator.
	ErrLastAdmin = errors.New("at least one admin must remain")
	// ErrUnknownUser identifies a membership email without an account.
	ErrUnknownUser = errors.New("no account uses this email; ask the person to register first")
	// ErrSession identifies a missing, expired or revoked session.
	ErrSession = errors.New("sign in to continue")
)

// User is the public account projection; it never includes the password hash.
type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role" jsonschema:"admin or user"`
	CreatedAt string `json:"created_at"`
}

// Membership grants one user a role on one product.
type Membership struct {
	ProductID string `json:"product_id"`
	Role      string `json:"role" jsonschema:"lead or learner"`
}

// Me describes the signed-in user and their product roles.
type Me struct {
	User        User         `json:"user"`
	Memberships []Membership `json:"memberships"`
}

// Member is one person with access to a product.
type Member struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role" jsonschema:"lead or learner"`
	CreatedAt string `json:"created_at"`
}

// Credentials registers or signs in with an email and password.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// MemberInput grants or changes a product role by account email.
type MemberInput struct {
	Email string `json:"email"`
	Role  string `json:"role" jsonschema:"lead or learner"`
}

// RoleInput changes a user's global role.
type RoleInput struct {
	Role string `json:"role" jsonschema:"admin or user"`
}

// Store uses the application's existing SQLite connection.
type Store struct{ DB *sql.DB }

// dummyHash keeps sign-in timing similar for unknown emails.
var dummyHash = sync.OnceValue(func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("wayfinder-timing-guard"), bcrypt.DefaultCost)
	return hash
})

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 {
		return "", fmt.Errorf("%w: enter a valid email address", ErrInvalid)
	}
	return email, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Register creates an account. The first account becomes an admin.
func (s Store) Register(ctx context.Context, input Credentials) (User, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return User{}, err
	}
	// bcrypt only reads the first 72 bytes, so longer values are rejected.
	if len(input.Password) < 8 || len(input.Password) > 72 {
		return User{}, fmt.Errorf("%w: password must contain 8–72 characters", ErrInvalid)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin registration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var users, taken int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE email=?) FROM users", email).Scan(&users, &taken); err != nil {
		return User{}, fmt.Errorf("check accounts: %w", err)
	}
	if taken > 0 {
		return User{}, ErrConflict
	}
	user := User{ID: rand.Text(), Email: email, Role: RoleUser, CreatedAt: now()}
	if users == 0 {
		user.Role = RoleAdmin
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO users(id,email,password_hash,role,created_at) VALUES(?,?,?,?,?)", user.ID, user.Email, string(hash), user.Role, user.CreatedAt); err != nil {
		return User{}, fmt.Errorf("save account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit account: %w", err)
	}
	return user, nil
}

// Login verifies credentials and returns a new session token.
func (s Store) Login(ctx context.Context, input Credentials) (string, User, error) {
	var user User
	var hash string
	email := strings.ToLower(strings.TrimSpace(input.Email))
	err := s.DB.QueryRowContext(ctx, "SELECT id,email,role,created_at,password_hash FROM users WHERE email=?", email).
		Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(input.Password))
		return "", User{}, ErrCredentials
	}
	if err != nil {
		return "", User{}, fmt.Errorf("find account: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		return "", User{}, ErrCredentials
	}
	token, err := s.CreateSession(ctx, user.ID)
	return token, user, err
}

// CreateSession stores only a hash of the returned bearer token.
func (s Store) CreateSession(ctx context.Context, userID string) (string, error) {
	token := rand.Text()
	current := time.Now().UTC()
	// A fixed-width timestamp keeps string comparison chronological.
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at<=?", current.Format(time.RFC3339)); err != nil {
		return "", fmt.Errorf("remove expired sessions: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at,created_at) VALUES(?,?,?,?)",
		hashToken(token), userID, current.Add(SessionLifetime).Format(time.RFC3339), current.Format(time.RFC3339Nano)); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}
	return token, nil
}

// Authenticate returns the user of an unexpired session.
func (s Store) Authenticate(ctx context.Context, token string) (User, error) {
	var user User
	err := s.DB.QueryRowContext(ctx, `SELECT u.id,u.email,u.role,u.created_at FROM sessions s JOIN users u ON u.id=s.user_id
 WHERE s.token_hash=? AND s.expires_at>?`, hashToken(token), time.Now().UTC().Format(time.RFC3339)).
		Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrSession
	}
	if err != nil {
		return User{}, fmt.Errorf("read session: %w", err)
	}
	return user, nil
}

// Logout revokes one session.
func (s Store) Logout(ctx context.Context, token string) error {
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash=?", hashToken(token)); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// Memberships lists a user's product roles.
func (s Store) Memberships(ctx context.Context, userID string) ([]Membership, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT product_id,role FROM product_members WHERE user_id=? ORDER BY created_at", userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()
	result := make([]Membership, 0)
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.ProductID, &m.Role); err != nil {
			return nil, fmt.Errorf("read membership: %w", err)
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read memberships: %w", err)
	}
	return result, nil
}

// ProductRole returns the user's role on a product, or an empty string without access.
func (s Store) ProductRole(ctx context.Context, userID, productID string) (string, error) {
	var role string
	err := s.DB.QueryRowContext(ctx, "SELECT role FROM product_members WHERE user_id=? AND product_id=?", userID, productID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read product role: %w", err)
	}
	return role, nil
}

// LeadsAny reports whether the user leads at least one product.
func (s Store) LeadsAny(ctx context.Context, userID string) (bool, error) {
	var leads bool
	if err := s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM product_members WHERE user_id=? AND role=?)", userID, RoleLead).Scan(&leads); err != nil {
		return false, fmt.Errorf("read lead role: %w", err)
	}
	return leads, nil
}

// Users lists all accounts.
func (s Store) Users(ctx context.Context) ([]User, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,email,role,created_at FROM users ORDER BY email")
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()
	result := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("read account: %w", err)
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read accounts: %w", err)
	}
	return result, nil
}

// lockAdmin loads a user and rejects changes that would leave no admin.
func lockAdmin(ctx context.Context, tx *sql.Tx, userID string, keepsAdmin bool) (User, error) {
	var u User
	if err := tx.QueryRowContext(ctx, "SELECT id,email,role,created_at FROM users WHERE id=?", userID).Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt); err != nil {
		return u, fmt.Errorf("find account: %w", err)
	}
	if u.Role != RoleAdmin || keepsAdmin {
		return u, nil
	}
	var admins int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role=?", RoleAdmin).Scan(&admins); err != nil {
		return u, fmt.Errorf("count admins: %w", err)
	}
	if admins <= 1 {
		return u, ErrLastAdmin
	}
	return u, nil
}

// SetRole changes a user's global role.
func (s Store) SetRole(ctx context.Context, userID, role string) (User, error) {
	if role != RoleAdmin && role != RoleUser {
		return User{}, fmt.Errorf("%w: role must be admin or user", ErrInvalid)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin role change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	user, err := lockAdmin(ctx, tx, userID, role == RoleAdmin)
	if err != nil {
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET role=? WHERE id=?", role, userID); err != nil {
		return User{}, fmt.Errorf("save role: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit role: %w", err)
	}
	user.Role = role
	return user, nil
}

// DeleteUser removes an account with its sessions and product roles.
func (s Store) DeleteUser(ctx context.Context, userID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account removal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := lockAdmin(ctx, tx, userID, false); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id=?", userID); err != nil {
		return fmt.Errorf("remove account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account removal: %w", err)
	}
	return nil
}

// Members lists the people with access to a product.
func (s Store) Members(ctx context.Context, productID string) ([]Member, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT u.id,u.email,m.role,m.created_at FROM product_members m JOIN users u ON u.id=m.user_id
 WHERE m.product_id=? ORDER BY m.role, u.email`, productID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	result := make([]Member, 0)
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.UserID, &m.Email, &m.Role, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("read member: %w", err)
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read members: %w", err)
	}
	return result, nil
}

// SetMember grants a registered user a product role or changes it.
func (s Store) SetMember(ctx context.Context, productID string, input MemberInput) (Member, error) {
	if input.Role != RoleLead && input.Role != RoleLearner {
		return Member{}, fmt.Errorf("%w: role must be lead or learner", ErrInvalid)
	}
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return Member{}, err
	}
	var exists bool
	if checkErr := s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id=?)", productID).Scan(&exists); checkErr != nil {
		return Member{}, fmt.Errorf("find product: %w", checkErr)
	}
	if !exists {
		return Member{}, sql.ErrNoRows
	}
	member := Member{Email: email, Role: input.Role}
	err = s.DB.QueryRowContext(ctx, "SELECT id FROM users WHERE email=?", email).Scan(&member.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrUnknownUser
	}
	if err != nil {
		return Member{}, fmt.Errorf("find account: %w", err)
	}
	err = s.DB.QueryRowContext(ctx, `INSERT INTO product_members(product_id,user_id,role,created_at) VALUES(?,?,?,?)
 ON CONFLICT(product_id,user_id) DO UPDATE SET role=excluded.role RETURNING created_at`, productID, member.UserID, member.Role, now()).Scan(&member.CreatedAt)
	if err != nil {
		return Member{}, fmt.Errorf("save member: %w", err)
	}
	return member, nil
}

// RemoveMember revokes a user's product role.
func (s Store) RemoveMember(ctx context.Context, productID, userID string) error {
	result, err := s.DB.ExecContext(ctx, "DELETE FROM product_members WHERE product_id=? AND user_id=?", productID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
