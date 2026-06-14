package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nthung2499/fake-gk/internal/chat"
	"github.com/nthung2499/fake-gk/internal/config"
	"github.com/nthung2499/fake-gk/internal/db"
	"github.com/nthung2499/fake-gk/internal/openai"
	"github.com/nthung2499/fake-gk/internal/secrets"
)

const (
	userCookieName = "fakegk_user_id"
	defaultPrompt  = "You are FakeGK, a helpful personal AI assistant. Keep answers clear and concise. Reply in Vietnamese when the user writes Vietnamese."
)

type Server struct {
	cfg    config.Config
	store  *db.Store
	chat   *chat.Repository
	cipher *secrets.Cipher
	openai *openai.Client
}

type indexData struct {
	HasAPIKey       bool
	KeyHint         string
	Model           string
	Sessions        []chat.Session
	ActiveSessionID int64
	Messages        []chat.Message
}

type messageRequest struct {
	Message string `json:"message"`
}

type keyRequest struct {
	APIKey string `json:"apiKey"`
}

type sessionRequest struct {
	Title string `json:"title"`
}

func New(cfg config.Config, store *db.Store) (*gin.Engine, error) {
	cipher, err := secrets.NewCipher(cfg.App.Secret)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:    cfg,
		store:  store,
		chat:   chat.NewRepository(store.DB),
		cipher: cipher,
		openai: openai.NewClient(time.Duration(cfg.OpenAI.RequestTimeoutSeconds) * time.Second),
	}

	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	router.SetHTMLTemplate(tmpl)
	router.StaticFile("/styles.css", "src/public/styles.css")
	router.StaticFile("/theme.js", "src/public/theme.js")
	router.StaticFile("/ui.js", "src/public/ui.js")
	router.StaticFile("/favicon.svg", "src/public/favicon.svg")

	router.GET("/healthz", s.health)
	router.GET("/readyz", s.ready)
	router.GET("/", s.index)
	router.POST("/api/key", s.saveKey)
	router.DELETE("/api/key", s.deleteKey)
	router.POST("/api/sessions", s.createSession)
	router.GET("/api/sessions/:id/messages", s.listMessages)
	router.POST("/api/sessions/:id/messages", s.createMessage)
	router.GET("/api/sessions/:id/stream", s.streamMessage)
	router.POST("/api/sessions/:id/rename", s.renameSession)
	router.POST("/api/sessions/:id/delete", s.deleteSession)

	return router, nil
}

func loadTemplates() (*template.Template, error) {
	templateDir := resolveTemplatesDir()
	tmpl := template.New("")
	if _, err := tmpl.ParseGlob(filepath.Join(templateDir, "*.html")); err != nil {
		return nil, err
	}
	if _, err := tmpl.ParseGlob(filepath.Join(templateDir, "partials", "*.html")); err != nil {
		return nil, err
	}
	return tmpl, nil
}

func resolveTemplatesDir() string {
	candidates := []string{
		"web/templates",
		filepath.Join("..", "..", "web", "templates"),
		filepath.Join("..", "..", "..", "web", "templates"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "web/templates"
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) ready(c *gin.Context) {
	if err := s.store.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) index(c *gin.Context) {
	user, err := s.ensureUser(c)
	if err != nil {
		s.renderError(c, err)
		return
	}

	key, keyErr := s.chat.GetAPIKey(c.Request.Context(), user.ID)
	hasKey := keyErr == nil
	if keyErr != nil && !errors.Is(keyErr, chat.ErrNotFound) {
		s.renderError(c, keyErr)
		return
	}

	sessions, err := s.chat.ListSessions(c.Request.Context(), user.ID)
	if err != nil {
		s.renderError(c, err)
		return
	}

	var activeSessionID int64
	var messages []chat.Message
	if len(sessions) > 0 {
		activeSessionID = sessions[0].ID
		messages, err = s.chat.ListMessages(c.Request.Context(), user.ID, activeSessionID, 1000)
		if err != nil {
			s.renderError(c, err)
			return
		}
	}

	c.HTML(http.StatusOK, "index.html", indexData{
		HasAPIKey:       hasKey,
		KeyHint:         key.KeyHint,
		Model:           s.cfg.OpenAI.Model,
		Sessions:        sessions,
		ActiveSessionID: activeSessionID,
		Messages:        messages,
	})
}

func (s *Server) saveKey(c *gin.Context) {
	user, err := s.ensureUser(c)
	if err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}

	var req keyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.APIKey = c.PostForm("apiKey")
	}
	req.APIKey = strings.TrimSpace(req.APIKey)
	if !secrets.LooksLikeOpenAIKey(req.APIKey) {
		s.jsonError(c, http.StatusBadRequest, errors.New("Please enter a valid OpenAI API key"))
		return
	}

	encrypted, err := s.cipher.Encrypt(req.APIKey)
	if err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}
	hint := secrets.Hint(req.APIKey)
	if err := s.chat.UpsertAPIKey(c.Request.Context(), user.ID, encrypted, hint); err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "keyHint": hint})
}

func (s *Server) deleteKey(c *gin.Context) {
	user, err := s.ensureUser(c)
	if err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}
	if err := s.chat.DeleteAPIKey(c.Request.Context(), user.ID); err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) createSession(c *gin.Context) {
	user, err := s.ensureUser(c)
	if err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}
	var req sessionRequest
	_ = c.ShouldBindJSON(&req)
	session, err := s.chat.CreateSession(c.Request.Context(), user.ID, req.Title)
	if err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"session": session})
}

func (s *Server) listMessages(c *gin.Context) {
	user, sessionID, ok := s.userAndSessionID(c)
	if !ok {
		return
	}
	messages, err := s.chat.ListMessages(c.Request.Context(), user.ID, sessionID, 1000)
	if err != nil {
		s.handleChatError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (s *Server) createMessage(c *gin.Context) {
	user, sessionID, ok := s.userAndSessionID(c)
	if !ok {
		return
	}
	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.jsonError(c, http.StatusBadRequest, errors.New("message is required"))
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		s.jsonError(c, http.StatusBadRequest, errors.New("message is required"))
		return
	}

	apiKey, err := s.userAPIKey(c, user.ID)
	if err != nil {
		s.handleChatError(c, err)
		return
	}

	userMessage, err := s.chat.AddMessage(c.Request.Context(), user.ID, sessionID, chat.RoleUser, req.Message, "", chat.StatusComplete, "")
	if err != nil {
		s.handleChatError(c, err)
		return
	}
	s.maybeRenameSession(c, user.ID, sessionID, req.Message)

	contextMessages, err := s.openAIContext(c, user.ID, sessionID)
	if err != nil {
		s.handleChatError(c, err)
		return
	}
	answer, err := s.openai.Generate(c.Request.Context(), apiKey, s.cfg.OpenAI.Model, defaultPrompt, contextMessages)
	if err != nil {
		assistantMessage, _ := s.chat.AddMessage(c.Request.Context(), user.ID, sessionID, chat.RoleAssistant, "", s.cfg.OpenAI.Model, chat.StatusError, err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"userMessage": userMessage, "assistantMessage": assistantMessage, "error": err.Error()})
		return
	}
	assistantMessage, err := s.chat.AddMessage(c.Request.Context(), user.ID, sessionID, chat.RoleAssistant, answer, s.cfg.OpenAI.Model, chat.StatusComplete, "")
	if err != nil {
		s.handleChatError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"userMessage": userMessage, "assistantMessage": assistantMessage})
}

func (s *Server) streamMessage(c *gin.Context) {
	user, err := s.ensureUser(c)
	beginSSE(c)
	send := func(event string, payload any) error {
		return sendSSE(c, event, payload)
	}
	if err != nil {
		_ = send("app_error", gin.H{"error": err.Error()})
		return
	}

	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		_ = send("app_error", gin.H{"error": "invalid session id"})
		return
	}

	message := strings.TrimSpace(c.Query("message"))
	if message == "" {
		_ = send("app_error", gin.H{"error": "message is required"})
		return
	}

	apiKey, err := s.userAPIKey(c, user.ID)
	if err != nil {
		_ = send("app_error", gin.H{"error": err.Error()})
		return
	}

	if _, err := s.chat.AddMessage(c.Request.Context(), user.ID, sessionID, chat.RoleUser, message, "", chat.StatusComplete, ""); err != nil {
		_ = send("app_error", gin.H{"error": err.Error()})
		return
	}
	s.maybeRenameSession(c, user.ID, sessionID, message)

	contextMessages, err := s.openAIContext(c, user.ID, sessionID)
	if err != nil {
		_ = send("app_error", gin.H{"error": err.Error()})
		return
	}

	answer, err := s.openai.Stream(c.Request.Context(), apiKey, s.cfg.OpenAI.Model, defaultPrompt, contextMessages, func(delta string) error {
		return send("delta", gin.H{"delta": delta})
	})
	if err != nil {
		assistantMessage, _ := s.chat.AddMessage(c.Request.Context(), user.ID, sessionID, chat.RoleAssistant, "", s.cfg.OpenAI.Model, chat.StatusError, err.Error())
		_ = send("app_error", gin.H{"error": err.Error(), "message": assistantMessage})
		return
	}
	assistantMessage, err := s.chat.AddMessage(c.Request.Context(), user.ID, sessionID, chat.RoleAssistant, answer, s.cfg.OpenAI.Model, chat.StatusComplete, "")
	if err != nil {
		_ = send("app_error", gin.H{"error": err.Error()})
		return
	}
	_ = send("done", gin.H{"message": assistantMessage})
}

func (s *Server) renameSession(c *gin.Context) {
	user, sessionID, ok := s.userAndSessionID(c)
	if !ok {
		return
	}
	var req sessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.jsonError(c, http.StatusBadRequest, errors.New("title is required"))
		return
	}
	if err := s.chat.RenameSession(c.Request.Context(), user.ID, sessionID, req.Title); err != nil {
		s.handleChatError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteSession(c *gin.Context) {
	user, sessionID, ok := s.userAndSessionID(c)
	if !ok {
		return
	}
	if err := s.chat.DeleteSession(c.Request.Context(), user.ID, sessionID); err != nil {
		s.handleChatError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) ensureUser(c *gin.Context) (chat.User, error) {
	if cookie, err := c.Cookie(userCookieName); err == nil && cookie != "" {
		user, err := s.chat.FindUserByPublicID(c.Request.Context(), cookie)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, chat.ErrNotFound) {
			return chat.User{}, err
		}
	}

	publicID, err := chat.NewPublicID()
	if err != nil {
		return chat.User{}, err
	}
	user, err := s.chat.CreateUser(c.Request.Context(), publicID)
	if err != nil {
		return chat.User{}, err
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     userCookieName,
		Value:    publicID,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return user, nil
}

func (s *Server) userAndSessionID(c *gin.Context) (chat.User, int64, bool) {
	user, err := s.ensureUser(c)
	if err != nil {
		s.jsonError(c, http.StatusInternalServerError, err)
		return chat.User{}, 0, false
	}
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		s.jsonError(c, http.StatusBadRequest, errors.New("invalid session id"))
		return chat.User{}, 0, false
	}
	return user, sessionID, true
}

func (s *Server) userAPIKey(c *gin.Context, userID int64) (string, error) {
	record, err := s.chat.GetAPIKey(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, chat.ErrNotFound) {
			return "", errors.New("Please add your OpenAI API key first")
		}
		return "", err
	}
	return s.cipher.Decrypt(record.EncryptedKey)
}

func (s *Server) openAIContext(c *gin.Context, userID, sessionID int64) ([]openai.Message, error) {
	messages, err := s.chat.ListMessages(c.Request.Context(), userID, sessionID, s.cfg.OpenAI.ContextMessages)
	if err != nil {
		return nil, err
	}
	result := make([]openai.Message, 0, len(messages))
	for _, message := range messages {
		if message.Status != chat.StatusComplete || strings.TrimSpace(message.Content) == "" {
			continue
		}
		result = append(result, openai.Message{Role: message.Role, Content: message.Content})
	}
	return result, nil
}

func (s *Server) maybeRenameSession(c *gin.Context, userID, sessionID int64, message string) {
	session, err := s.chat.GetSession(c.Request.Context(), userID, sessionID)
	if err != nil || session.Title != "New chat" {
		return
	}
	title := strings.TrimSpace(message)
	runes := []rune(title)
	if len(runes) > 48 {
		title = string(runes[:48]) + "..."
	}
	_ = s.chat.RenameSession(c.Request.Context(), userID, sessionID, title)
}

func (s *Server) handleChatError(c *gin.Context, err error) {
	s.jsonError(c, statusForError(err), err)
}

func statusForError(err error) int {
	if errors.Is(err, chat.ErrNotFound) {
		return http.StatusNotFound
	}
	if strings.Contains(err.Error(), "API key") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func beginSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
}

func sendSSE(c *gin.Context, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func (s *Server) jsonError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func (s *Server) renderError(c *gin.Context, err error) {
	c.HTML(http.StatusInternalServerError, "error.html", gin.H{
		"Message": "Fake GK hit a server error.",
		"Detail":  err.Error(),
	})
}
