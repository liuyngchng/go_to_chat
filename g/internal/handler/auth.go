package handler

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/store"

	"github.com/gin-gonic/gin"
)

// defaultTokenSecret 默认 HMAC 签名密钥（cfg.yml 未配置时使用）
var defaultTokenSecret = []byte("kb-chat-flow_secret_2026")

// token 有效期 2 小时
const tokenTTL = 2 * time.Hour

// getTokenSecret 获取当前 token 签名密钥
func (h *AuthHandler) getTokenSecret() []byte {
	if h.cfg.Server.TokenSecret != "" {
		return []byte(h.cfg.Server.TokenSecret)
	}
	return defaultTokenSecret
}

// AuthHandler 认证处理器
type AuthHandler struct {
	cfg      *model.Config
	store    store.MetaStore
	presence PresenceStore
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(cfg *model.Config, metaStore store.MetaStore, presence PresenceStore) *AuthHandler {
	return &AuthHandler{
		cfg:      cfg,
		store:    metaStore,
		presence: presence,
	}
}

// OnlineAgent 在线座席信息
type OnlineAgent struct {
	UserName  string `json:"user_name"`
	LoginTime string `json:"login_time"`
	Note      string `json:"note"`
}

// LoginPage 登录页面
func (h *AuthHandler) LoginPage(c *gin.Context) {
	pageTitle := h.cfg.Sys.Name
	if h.cfg.Server.Role == model.SvcRoleAdmin {
		pageTitle = h.cfg.Sys.Name + "系统管理"
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"page_title":   pageTitle,
		"default_user": "user0",
		"default_pwd":  "user0",
		"error_msg":    "",
	})
}

// generateToken 生成 HMAC 签名 token
// 格式: base64(user_name|expiry_timestamp|hmac_signature)
func generateToken(userName string, role int, expiry time.Time, secret []byte) string {
	expiryUnix := strconv.FormatInt(expiry.Unix(), 10)
	payload := fmt.Sprintf("%s|%d|%s", userName, role, expiryUnix)

	// HMAC-SHA256 签名
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))[:16] // 取前 16 位

	full := fmt.Sprintf("%s|%s", payload, sig)
	return base64.RawURLEncoding.EncodeToString([]byte(full))
}

// parseToken 解析并验证 token，返回 user 或 nil
func (h *AuthHandler) parseToken(tokenStr string) *model.User {
	// Base64 解码
	data, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil
	}

	parts := strings.SplitN(string(data), "|", 4)
	if len(parts) != 4 {
		return nil
	}

	userName := parts[0]
	role, _ := strconv.Atoi(parts[1])
	expiryUnix := parts[2]
	sig := parts[3]

	// 检查过期
	expiry, err := strconv.ParseInt(expiryUnix, 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return nil
	}

	// 验证签名
	payload := fmt.Sprintf("%s|%d|%s", userName, role, expiryUnix)
	mac := hmac.New(sha256.New, h.getTokenSecret())
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))[:16]

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil
	}

	return &model.User{
		UserName: userName,
		Role:     role,
	}
}

// Login 处理登录请求（JSON）
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.UserName == "" || req.UserPwd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	// MD5 密码
	md5Pwd := md5Hash(req.UserPwd)

	user, err := h.store.GetUserByLogin(req.UserName, md5Pwd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败: " + err.Error()})
		return
	}

	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// admin 实例：仅管理员可登录
	if h.cfg.Server.Role == model.SvcRoleAdmin && user.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "此账号无法访问管理后台"})
		return
	}

	expiry := time.Now().Add(tokenTTL)
	token := generateToken(user.UserName, user.Role, expiry, h.getTokenSecret())

	// 如果是客服座席，加入在线列表
	if user.Role == model.RoleAgent {
		h.presence.SetPresence(user.UserName, time.Now())
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"token":     token,
		"user_name": user.UserName,
		"role":      user.Role,
	})
}

// Logout 处理注销
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从 token 中解析用户
	if tokenStr := extractToken(c); tokenStr != "" {
		if user := h.parseToken(tokenStr); user != nil {
			h.presence.RemovePresence(user.UserName)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// extractToken 从 URL 参数 t 或 Authorization 头提取 token
func extractToken(c *gin.Context) string {
	// 优先从 URL 参数 t 读取
	if t := c.Query("t"); t != "" {
		return t
	}
	// 其次从 Authorization 头读取
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// AuthMiddleware 认证中间件：验证 token
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user := h.parseToken(tokenStr)
		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("token_str", tokenStr)
		c.Next()
	}
}

// GetTokenStr 从 context 提取原始 token 字符串
func GetTokenStr(c *gin.Context) string {
	if ts, exists := c.Get("token_str"); exists {
		if s, ok := ts.(string); ok {
			return s
		}
	}
	return ""
}

// AdminOnlyMiddleware 仅允许管理员访问
func (h *AuthHandler) AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "禁止访问"})
			c.Abort()
			return
		}

		user, ok := userVal.(*model.User)
		if !ok || user.Role != model.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可访问"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ApiAuthMiddleware API 认证中间件：受 sys.api_auth 开关控制
func (h *AuthHandler) ApiAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 始终尝试从请求中提取 token，设置 user（后续 AdminOnlyMiddleware 等依赖此值）
		tokenStr := extractToken(c)
		if tokenStr != "" {
			if user := h.parseToken(tokenStr); user != nil {
				c.Set("user", user)
				c.Set("token_str", tokenStr)
			}
		}

		// 接口认证关闭时，跳过认证检查（但 user 已设置）
		if !h.cfg.Sys.ApiAuth {
			c.Next()
			return
		}

		// 接口认证开启时，必须提供有效 token
		if _, exists := c.Get("user"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证 token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetOnlineAgents 获取在线座席列表
func (h *AuthHandler) GetOnlineAgents(c *gin.Context) {
	agents := h.presence.GetOnlineAgents()
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// Me 返回当前登录用户信息（从 Authorization header 或 URL t 参数解析 token）
func (h *AuthHandler) Me(c *gin.Context) {
	tokenStr := extractToken(c)
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	user := h.parseToken(tokenStr)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token 无效或已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_name": user.UserName,
		"role":      user.Role,
	})
}

// md5Hash 计算字符串的 MD5
func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
