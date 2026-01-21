package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a system user or service account
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"` // admin, user, viewer, service
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Token represents an authentication token
type Token struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"-"` // Never send to client
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

// APIKey represents a service account API key
type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	UserID      string    `json:"user_id"`
	KeyPrefix   string    `json:"key_prefix"` // First 8 chars for display
	KeyHash     string    `json:"-"`          // Never send to client
	Permissions []string  `json:"permissions"`
	IsActive    bool      `json:"is_active"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used,omitempty"`
}

// Role defines permissions for users
type Role struct {
	Name        string   `json:"name"` // admin, user, viewer, service
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Permission represents an action that can be granted
type Permission struct {
	Name        string `json:"name"` // e.g., "agents:read", "beads:write"
	Description string `json:"description"`
	Resource    string `json:"resource"` // agents, beads, providers, projects, decisions
	Action      string `json:"action"`   // read, write, delete, admin
}

// Claims represents JWT claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// Implement jwt.Claims interface
func (c *Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.ExpiresAt, nil
}

func (c *Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.IssuedAt, nil
}

func (c *Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.NotBefore, nil
}

func (c *Claims) GetIssuer() (string, error) {
	return c.Issuer, nil
}

func (c *Claims) GetSubject() (string, error) {
	return c.Subject, nil
}

func (c *Claims) GetAudience() (jwt.ClaimStrings, error) {
	return c.Audience, nil
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"` // seconds
	User      User   `json:"user"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	Token string `json:"token"`
}

// CreateAPIKeyRequest represents API key creation request
type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	ExpiresIn   int64    `json:"expires_in,omitempty"` // seconds, 0 = no expiry
}

// CreateAPIKeyResponse returns the new API key (only shown once)
type CreateAPIKeyResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"` // Full key - only shown once!
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// PreDefinedRoles contains the standard roles
var PreDefinedRoles = map[string]Role{
	"admin": {
		Name:        "admin",
		Description: "Full system access",
		Permissions: []string{
			"*:*", // All permissions
		},
	},
	"user": {
		Name:        "user",
		Description: "Read and write access to most resources",
		Permissions: []string{
			"agents:read",
			"agents:write",
			"beads:read",
			"beads:write",
			"providers:read",
			"providers:write",
			"projects:read",
			"projects:write",
			"decisions:read",
			"decisions:write",
			"repl:use",
		},
	},
	"viewer": {
		Name:        "viewer",
		Description: "Read-only access",
		Permissions: []string{
			"agents:read",
			"beads:read",
			"providers:read",
			"projects:read",
			"decisions:read",
		},
	},
	"service": {
		Name:        "service",
		Description: "Service account with API key auth",
		Permissions: []string{
			// Service accounts grant specific permissions
			// Set per API key
		},
	},
}

// PreDefinedPermissions contains all available permissions
var PreDefinedPermissions = []Permission{
	// Agents
	{Name: "agents:read", Resource: "agents", Action: "read", Description: "Read agent information"},
	{Name: "agents:write", Resource: "agents", Action: "write", Description: "Create/modify agents"},
	{Name: "agents:delete", Resource: "agents", Action: "delete", Description: "Delete agents"},
	{Name: "agents:admin", Resource: "agents", Action: "admin", Description: "Admin access to agents"},

	// Beads
	{Name: "beads:read", Resource: "beads", Action: "read", Description: "Read bead information"},
	{Name: "beads:write", Resource: "beads", Action: "write", Description: "Create/modify beads"},
	{Name: "beads:delete", Resource: "beads", Action: "delete", Description: "Delete beads"},
	{Name: "beads:admin", Resource: "beads", Action: "admin", Description: "Admin access to beads"},

	// Providers
	{Name: "providers:read", Resource: "providers", Action: "read", Description: "Read provider information"},
	{Name: "providers:write", Resource: "providers", Action: "write", Description: "Create/modify providers"},
	{Name: "providers:delete", Resource: "providers", Action: "delete", Description: "Delete providers"},
	{Name: "providers:admin", Resource: "providers", Action: "admin", Description: "Admin access to providers"},

	// Projects
	{Name: "projects:read", Resource: "projects", Action: "read", Description: "Read project information"},
	{Name: "projects:write", Resource: "projects", Action: "write", Description: "Create/modify projects"},
	{Name: "projects:delete", Resource: "projects", Action: "delete", Description: "Delete projects"},
	{Name: "projects:admin", Resource: "projects", Action: "admin", Description: "Admin access to projects"},

	// Decisions
	{Name: "decisions:read", Resource: "decisions", Action: "read", Description: "Read decision information"},
	{Name: "decisions:write", Resource: "decisions", Action: "write", Description: "Create/modify decisions"},
	{Name: "decisions:delete", Resource: "decisions", Action: "delete", Description: "Delete decisions"},
	{Name: "decisions:admin", Resource: "decisions", Action: "admin", Description: "Admin access to decisions"},

	// System
	{Name: "repl:use", Resource: "repl", Action: "write", Description: "Use CEO REPL"},
	{Name: "system:admin", Resource: "system", Action: "admin", Description: "Full system administration"},

	// Catch-all
	{Name: "*:*", Resource: "*", Action: "*", Description: "All permissions"},
}
