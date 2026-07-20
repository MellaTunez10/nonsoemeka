package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
	"nonsoemeka-backend/internal/validation"
)

type AuthHandler struct {
	authService services.AuthService
	cfg         *config.Config
}

func NewAuthHandler(authService services.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NotEmpty("username", req.Username)
	v.NotEmpty("password", req.Password)
	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	user, accessToken, rawRefreshToken, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Set httpOnly refresh_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    rawRefreshToken,
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(h.cfg.JWT.RefreshTTL),
		MaxAge:   int(h.cfg.JWT.RefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
	})

	resp := dto.LoginResponse{
		AccessToken: accessToken,
		User: dto.UserProfileResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		},
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var rawRefreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawRefreshToken = cookie.Value
	}

	if rawRefreshToken == "" {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	accessToken, newRawRefreshToken, err := h.authService.Refresh(r.Context(), rawRefreshToken)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRawRefreshToken,
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(h.cfg.JWT.RefreshTTL),
		MaxAge:   int(h.cfg.JWT.RefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
	})

	h.writeJSON(w, http.StatusOK, map[string]string{
		"access_token": accessToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		_ = h.authService.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
	})

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

func (h *AuthHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *AuthHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := apperrors.ToHTTPStatus(err)
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "An unexpected error occurred"
	}

	_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
		Error: apperrors.ErrorDetail{
			Code:      code,
			Message:   msg,
			RequestID: reqID,
		},
	})
}

func (h *AuthHandler) writeValidationErrors(w http.ResponseWriter, r *http.Request, ve validation.ValidationErrors) {
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
		Error: apperrors.ErrorDetail{
			Code:      "VALIDATION_ERROR",
			Message:   ve.Error(),
			RequestID: reqID,
		},
	})
}
