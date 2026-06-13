package server

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/nthung2499/fake-gk/internal/config"
	"github.com/nthung2499/fake-gk/internal/db"
	"github.com/nthung2499/fake-gk/internal/notes"
)

type Server struct {
	cfg   config.Config
	store *db.Store
	notes *notes.Repository
}

type indexData struct {
	Notes         []notes.Note
	ArchivedNotes []notes.Note
	PinnedNotes   []notes.Note
	RecentNotes   []notes.Note
	TotalNotes    int
	Colors        []string
}

type noteCardData struct {
	Note   notes.Note
	Colors []string
}

func New(cfg config.Config, store *db.Store) (*gin.Engine, error) {
	s := &Server{
		cfg:   cfg,
		store: store,
		notes: notes.NewRepository(store.DB),
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
	router.POST("/notes", s.createNote)
	router.POST("/notes/:id", s.updateNote)
	router.POST("/notes/:id/pin", s.togglePinned)
	router.POST("/notes/:id/archive", s.toggleArchived)
	router.POST("/notes/:id/delete", s.deleteNote)

	return router, nil
}

func loadTemplates() (*template.Template, error) {
	templateDir := resolveTemplatesDir()
	funcs := template.FuncMap{
		"noteCard": func(note notes.Note, colors []string) noteCardData {
			return noteCardData{Note: note, Colors: colors}
		},
	}

	tmpl := template.New("").Funcs(funcs)
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
	allNotes, err := s.notes.List(c.Request.Context())
	if err != nil {
		s.renderError(c, err)
		return
	}

	var active []notes.Note
	var archived []notes.Note
	var pinned []notes.Note
	var recent []notes.Note
	for _, note := range allNotes {
		if note.IsArchived {
			archived = append(archived, note)
			continue
		}
		active = append(active, note)
		if note.IsPinned {
			pinned = append(pinned, note)
		} else {
			recent = append(recent, note)
		}
	}

	c.HTML(http.StatusOK, "index.html", indexData{
		Notes:         active,
		ArchivedNotes: archived,
		PinnedNotes:   pinned,
		RecentNotes:   recent,
		TotalNotes:    len(active) + len(archived),
		Colors:        notes.Colors(),
	})
}

func (s *Server) createNote(c *gin.Context) {
	if err := s.notes.Create(c.Request.Context(), noteInput(c)); err != nil {
		s.renderError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (s *Server) updateNote(c *gin.Context) {
	if err := s.notes.Update(c.Request.Context(), c.Param("id"), noteInput(c)); err != nil {
		s.renderError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (s *Server) togglePinned(c *gin.Context) {
	if err := s.notes.TogglePinned(c.Request.Context(), c.Param("id")); err != nil {
		s.renderError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (s *Server) toggleArchived(c *gin.Context) {
	if err := s.notes.ToggleArchived(c.Request.Context(), c.Param("id")); err != nil {
		s.renderError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (s *Server) deleteNote(c *gin.Context) {
	if err := s.notes.Delete(c.Request.Context(), c.Param("id")); err != nil {
		s.renderError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func noteInput(c *gin.Context) notes.Input {
	return notes.Input{
		Title: c.PostForm("title"),
		Body:  c.PostForm("body"),
		Color: c.PostForm("color"),
	}
}

func (s *Server) renderError(c *gin.Context, err error) {
	c.HTML(http.StatusInternalServerError, "error.html", gin.H{
		"Message": "Fake GK hit a server error.",
		"Detail":  err.Error(),
	})
}
