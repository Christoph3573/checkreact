package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	apimw "schulapp/internal/middleware"
)

const (
	refreshTokenDuration = 7 * 24 * time.Hour
	accessTokenDuration  = 15 * time.Minute
	refreshCookieName    = "refresh_token"
)

type Auth struct {
	db        *pgxpool.Pool
	jwtSecret []byte
}

func NewAuth(db *pgxpool.Pool, jwtSecret []byte) *Auth {
	return &Auth{db: db, jwtSecret: jwtSecret}
}

type userRow struct {
	ID           pgtype.UUID
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Role         string
	IsActive     bool
}

func uuidStr(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

func (h *Auth) generateAccessToken(u userRow) (string, error) {
	claims := apimw.Claims{
		UserID:    uuidStr(u.ID),
		Email:     u.Email,
		Role:      u.Role,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   uuidStr(u.ID),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

func (h *Auth) createRefreshToken(ctx context.Context, userID pgtype.UUID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	tokenVal := base64.URLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(tokenVal))
	hashHex := hex.EncodeToString(sum[:])

	_, err := h.db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, hashHex, time.Now().Add(refreshTokenDuration))
	if err != nil {
		return "", err
	}
	return tokenVal, nil
}

func (h *Auth) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   false, // set true behind HTTPS in production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(refreshTokenDuration.Seconds()),
	})
}

func (h *Auth) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// POST /api/v1/auth/login
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password required")
		return
	}

	var u userRow
	err := h.db.QueryRow(r.Context(),
		`SELECT id, email, password_hash, first_name, last_name, role, is_active
		 FROM users WHERE email = $1`, req.Email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !u.IsActive {
		respondError(w, http.StatusUnauthorized, "account deactivated")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := h.generateAccessToken(u)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate token")
		return
	}
	refreshToken, err := h.createRefreshToken(r.Context(), u.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	h.setRefreshCookie(w, refreshToken)
	respond(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"user": map[string]any{
			"id":         uuidStr(u.ID),
			"email":      u.Email,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
			"role":       u.Role,
		},
	})
}

// POST /api/v1/auth/refresh
func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "no refresh token")
		return
	}

	sum := sha256.Sum256([]byte(cookie.Value))
	hashHex := hex.EncodeToString(sum[:])

	var tokenID pgtype.UUID
	var userID pgtype.UUID
	var expiresAt time.Time
	err = h.db.QueryRow(r.Context(),
		`SELECT id, user_id, expires_at FROM refresh_tokens WHERE token_hash = $1`, hashHex).
		Scan(&tokenID, &userID, &expiresAt)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if time.Now().After(expiresAt) {
		h.db.Exec(r.Context(), `DELETE FROM refresh_tokens WHERE id = $1`, tokenID) //nolint:errcheck
		respondError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	// Token rotation: delete old, issue new
	h.db.Exec(r.Context(), `DELETE FROM refresh_tokens WHERE id = $1`, tokenID) //nolint:errcheck

	var u userRow
	err = h.db.QueryRow(r.Context(),
		`SELECT id, email, password_hash, first_name, last_name, role, is_active
		 FROM users WHERE id = $1`, userID).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive)
	if err != nil || !u.IsActive {
		respondError(w, http.StatusUnauthorized, "user not found")
		return
	}

	accessToken, err := h.generateAccessToken(u)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate token")
		return
	}
	newRefreshToken, err := h.createRefreshToken(r.Context(), u.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	h.setRefreshCookie(w, newRefreshToken)
	respond(w, http.StatusOK, map[string]any{"access_token": accessToken})
}

// POST /api/v1/auth/logout
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		sum := sha256.Sum256([]byte(cookie.Value))
		hashHex := hex.EncodeToString(sum[:])
		h.db.Exec(r.Context(), `DELETE FROM refresh_tokens WHERE token_hash = $1`, hashHex) //nolint:errcheck
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/auth/me
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	claims := apimw.ClaimsFromCtx(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var u userRow
	err := h.db.QueryRow(r.Context(),
		`SELECT id, email, password_hash, first_name, last_name, role, is_active
		 FROM users WHERE id = $1::uuid`, claims.Subject).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"id":         uuidStr(u.ID),
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"role":       u.Role,
	})
}

// PATCH /api/v1/auth/me
func (h *Auth) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims := apimw.ClaimsFromCtx(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	if req.FirstName == "" || req.LastName == "" {
		respondError(w, http.StatusBadRequest, "first_name and last_name required")
		return
	}

	var u userRow
	err := h.db.QueryRow(r.Context(),
		`UPDATE users SET first_name = $1, last_name = $2, updated_at = NOW()
		 WHERE id = $3::uuid
		 RETURNING id, email, password_hash, first_name, last_name, role, is_active`,
		req.FirstName, req.LastName, claims.Subject).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusNotFound, "user not found")
		} else {
			respondError(w, http.StatusInternalServerError, "update failed")
		}
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"id":         uuidStr(u.ID),
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"role":       u.Role,
	})
}
