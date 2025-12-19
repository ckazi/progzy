package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"proxy-server/database"
	"proxy-server/middleware"
	"proxy-server/models"
	"proxy-server/utils"
)

type UsersHandler struct {
	db *database.Database
}

var allowedProxyTypes = map[string]struct{}{
	"default":   {},
	"whitelist": {},
	"blacklist": {},
}

func normalizeProxyType(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedProxyTypes[v]; ok {
		return v
	}
	return "default"
}

func NewUsersHandler(db *database.Database) *UsersHandler {
	return &UsersHandler{db: db}
}

func (h *UsersHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.GetAllUsers()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}

func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	req.ProxyType = normalizeProxyType(req.ProxyType)

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user, err := h.db.CreateUser(&req, hashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	if !user.IsAdmin && (req.Whitelist != nil || req.Blacklist != nil) {
		whitelist := req.Whitelist
		blacklist := req.Blacklist
		if whitelist == nil {
			whitelist = []string{}
		}
		if blacklist == nil {
			blacklist = []string{}
		}
		wl, bl, err := h.db.SetUserProxyLists(user.ID, whitelist, blacklist)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save proxy lists")
			return
		}
		user.Whitelist = wl
		user.Blacklist = bl
	}

	if actor := middleware.GetUserFromContext(r); actor != nil {
		details := fmt.Sprintf("Created user %s (id=%d) state=%s", user.Username, user.ID, formatAuditJSON(buildUserAuditSnapshot(user)))
		h.db.LogAdminAction(&actor.ID, "USER_CREATE", details, getRequestIP(r))
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	original, err := h.db.GetUserByID(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	var req models.UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Password != nil {
		hashedPassword, err := utils.HashPassword(*req.Password)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		req.Password = &hashedPassword
	}

	if req.ProxyType != nil {
		normalized := normalizeProxyType(*req.ProxyType)
		req.ProxyType = &normalized
	}

	if err := h.db.UpdateUser(id, &req); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch updated user")
		return
	}

	if actor := middleware.GetUserFromContext(r); actor != nil {
		payload := map[string]interface{}{
			"before":           buildUserAuditSnapshot(original),
			"after":            buildUserAuditSnapshot(user),
			"password_changed": req.Password != nil,
		}
		details := fmt.Sprintf("Updated user %s (id=%d) diff=%s", user.Username, user.ID, formatAuditJSON(payload))
		h.db.LogAdminAction(&actor.ID, "USER_UPDATE", details, getRequestIP(r))
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	if err := h.db.DeleteUser(id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	if actor := middleware.GetUserFromContext(r); actor != nil {
		details := fmt.Sprintf("Deleted user %s (id=%d) previous_state=%s", user.Username, user.ID, formatAuditJSON(buildUserAuditSnapshot(user)))
		h.db.LogAdminAction(&actor.ID, "USER_DELETE", details, getRequestIP(r))
	}

	respondWithJSON(w, http.StatusOK, models.SuccessResponse{Message: "User deleted successfully"})
}

func buildUserAuditSnapshot(user *models.User) map[string]interface{} {
	if user == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"username":   user.Username,
		"email":      user.Email,
		"comment":    user.Comment,
		"is_admin":   user.IsAdmin,
		"is_active":  user.IsActive,
		"proxy_type": user.ProxyType,
		"twofa":      user.TwoFAEnabled,
		"whitelist":  user.Whitelist,
		"blacklist":  user.Blacklist,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}
}
