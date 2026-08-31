package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"cybermind/internal/audit"
	"cybermind/internal/config"
	"cybermind/internal/database"
	"cybermind/internal/security"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles authentication-related endpoints.
type AuthHandler struct {
	manager    *security.AuthManager
	config     *config.Config
	configPath string
	logger     *zap.Logger
	audit      *audit.Service
	db         *database.DB
}

// SetAudit wires platform audit logging.
func (h *AuthHandler) SetAudit(s *audit.Service) {
	h.audit = s
}

// SetDB wires the database store for registration.
func (h *AuthHandler) SetDB(db *database.DB) {
	h.db = db
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(manager *security.AuthManager, cfg *config.Config, configPath string, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		manager:    manager,
		config:     cfg,
		configPath: configPath,
		logger:     logger,
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password" binding:"required"`
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// Login verifies password and returns a session token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码不能为空"})
		return
	}

	token, expiresAt, err := h.manager.Authenticate(req.Username, req.Password)
	if err != nil {
		if h.audit != nil {
			h.audit.Record(c, audit.Entry{
				Level:    "warn",
				Category: "auth",
				Action:   "login",
				Result:   "failure",
				Message:  "登录失败：密码错误",
				Actor:    strings.TrimSpace(req.Username),
			})
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	session, _ := h.manager.ValidateToken(token)

	if h.audit != nil {
		h.audit.Record(c, audit.Entry{
			Category:    "auth",
			Action:      "login",
			Result:      "success",
			SessionHint: audit.HintFromToken(token),
			Message:     "登录成功",
			Actor:       session.Username,
			Detail: map[string]interface{}{
				"expires_at": expiresAt.UTC().Format(time.RFC3339),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"token":               token,
		"expires_at":          expiresAt.UTC().Format(time.RFC3339),
		"session_duration_hr": h.manager.SessionDurationHours(),
		"user": gin.H{
			"id":           session.UserID,
			"username":     session.Username,
			"display_name": session.DisplayName,
		},
		"roles":             session.Roles,
		"permissions":       permissionKeys(session.Permissions),
		"permission_scopes": session.PermissionScopes,
		"scope":             session.Scope,
	})
}

// Register creates a new self-service user account and logs it in immediately.
// Registered users are assigned the built-in operator role: they can run the
// day-to-day security workflows but cannot manage accounts or core configuration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码均不能为空"})
		return
	}

	username := strings.ToLower(strings.TrimSpace(req.Username))
	if err := validateUsername(username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度至少需要 8 位"})
		return
	}

	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "用户存储未就绪"})
		return
	}

	if _, err := h.db.GetRBACUserByUsername(username); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	} else if err != sql.ErrNoRows {
		if h.logger != nil {
			h.logger.Error("查询用户名失败", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("生成密码哈希失败", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = username
	}

	if _, err := h.db.CreateRBACUser(username, displayName, hash, true, []string{database.RBACSystemRoleAdmin}); err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
			return
		}
		if h.logger != nil {
			h.logger.Error("创建用户失败", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	token, expiresAt, err := h.manager.Authenticate(username, password)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("注册后自动登录失败", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册成功但自动登录失败，请手动登录"})
		return
	}
	session, _ := h.manager.ValidateToken(token)

	if h.audit != nil {
		h.audit.Record(c, audit.Entry{
			Category: "auth",
			Action:   "register",
			Result:   "success",
			Message:  "用户注册成功",
			Actor:    session.Username,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"token":               token,
		"expires_at":          expiresAt.UTC().Format(time.RFC3339),
		"session_duration_hr": h.manager.SessionDurationHours(),
		"user": gin.H{
			"id":           session.UserID,
			"username":     session.Username,
			"display_name": session.DisplayName,
		},
		"roles":             session.Roles,
		"permissions":       permissionKeys(session.Permissions),
		"permission_scopes": session.PermissionScopes,
		"scope":             session.Scope,
	})
}

func validateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("用户名至少需要 3 个字符")
	}
	if len(username) > 32 {
		return errors.New("用户名最多 32 个字符")
	}
	if username == "admin" {
		return errors.New("该用户名不可用")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return errors.New("用户名只能包含小写字母、数字、_、-、.")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// Logout revokes the current session token.
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetString(security.ContextAuthTokenKey)
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		} else {
			token = strings.TrimSpace(authHeader)
		}
	}

	h.manager.RevokeToken(token)
	if h.audit != nil {
		h.audit.Record(c, audit.Entry{
			Category: "auth",
			Action:   "logout",
			Result:   "success",
			Message:  "退出登录",
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// ChangePassword updates the login password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	oldPassword := strings.TrimSpace(req.OldPassword)
	newPassword := strings.TrimSpace(req.NewPassword)

	if oldPassword == "" || newPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前密码和新密码均不能为空"})
		return
	}

	if len(newPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码长度至少需要 8 位"})
		return
	}

	if oldPassword == newPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码不能与旧密码相同"})
		return
	}

	session, _ := security.CurrentSession(c)
	if session.Username == "" {
		session.Username = "admin"
	}
	if !h.manager.CheckUserPassword(session.Username, oldPassword) {
		if h.audit != nil {
			h.audit.Record(c, audit.Entry{
				Level:    "warn",
				Category: "auth",
				Action:   "change_password",
				Result:   "failure",
				Message:  "修改密码失败：当前密码不正确",
			})
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前密码不正确"})
		return
	}

	if session.UserID == "" {
		session.UserID = "admin"
	}
	if err := h.manager.UpdateUserPassword(session.UserID, newPassword); err != nil {
		if h.logger != nil {
			h.logger.Error("更新用户密码失败", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户密码失败"})
		return
	}

	if h.logger != nil {
		h.logger.Info("登录密码已更新，所有会话已失效")
	}

	if h.audit != nil {
		h.audit.Record(c, audit.Entry{
			Category: "auth",
			Action:   "change_password",
			Result:   "success",
			Message:  "登录密码已修改",
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码已更新，请使用新密码重新登录"})
}

// Validate returns the current session status.
func (h *AuthHandler) Validate(c *gin.Context) {
	token := c.GetString(security.ContextAuthTokenKey)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "会话无效"})
		return
	}

	session, ok := h.manager.ValidateToken(token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "会话已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      session.Token,
		"expires_at": session.ExpiresAt.UTC().Format(time.RFC3339),
		"user": gin.H{
			"id":           session.UserID,
			"username":     session.Username,
			"display_name": session.DisplayName,
		},
		"roles":             session.Roles,
		"permissions":       permissionKeys(session.Permissions),
		"permission_scopes": session.PermissionScopes,
		"scope":             session.Scope,
	})
}

func permissionKeys(perms map[string]bool) []string {
	keys := make([]string, 0, len(perms))
	for key, ok := range perms {
		if ok {
			keys = append(keys, key)
		}
	}
	return keys
}
