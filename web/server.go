package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/kierank/pipes/auth"
	"github.com/kierank/pipes/config"
	"github.com/kierank/pipes/engine"
	"github.com/kierank/pipes/store"
)

type Server struct {
	cfg            *config.Config
	db             *store.DB
	server         *http.Server
	sessionManager *auth.SessionManager
	oauthClient    *auth.OAuthClient
	templates      *template.Template
	logger         *log.Logger
}

func NewServer(cfg *config.Config, db *store.DB, logger *log.Logger) *Server {
	return &Server{
		cfg:            cfg,
		db:             db,
		sessionManager: auth.NewSessionManager(cfg, db),
		oauthClient:    auth.NewOAuthClient(cfg, db),
		logger:         logger,
	}
}

func (s *Server) Start() error {
	// Load templates
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}
	s.templates = tmpl

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	// Public routes
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/health", s.handleHealth)

	// Auth routes
	mux.HandleFunc("/auth/login", s.handleLogin)
	mux.HandleFunc("/auth/callback", s.handleCallback)
	mux.HandleFunc("/auth/logout", s.handleLogout)

	// Protected routes
	mux.HandleFunc("/dashboard", s.sessionManager.RequireAuth(s.handleDashboard))
	mux.HandleFunc("/pipes/", s.sessionManager.RequireAuth(s.handlePipeEditor))

	// API routes
	mux.HandleFunc("/api/me", s.sessionManager.RequireAuth(s.handleAPIMe))
	mux.HandleFunc("/api/pipes", s.sessionManager.RequireAuth(s.handleAPIPipes))
	mux.HandleFunc("/api/pipes/", s.sessionManager.RequireAuth(s.handleAPIPipe))
	mux.HandleFunc("/api/node-types", s.handleAPINodeTypes)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
		Handler: mux,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Handlers

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	user, _ := s.sessionManager.GetCurrentUser(r)
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	s.templates.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	authURL, err := s.oauthClient.GetAuthorizationURL()
	if err != nil {
		s.logger.Error("failed to generate auth URL", "error", err)
		s.renderError(w, "Configuration Error", "Failed to start authentication process. Please contact the administrator.", err.Error())
		return
	}

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		s.renderError(w, "Invalid Request", "Missing authorization code or state parameter.", "")
		return
	}

	user, session, err := s.oauthClient.HandleCallback(state, code)
	if err != nil {
		s.logger.Error("oauth callback error", "error", err)
		s.renderError(w, "Authentication Failed", "We couldn't sign you in with Indiko. Please try again.", err.Error())
		return
	}

	if err := s.sessionManager.SetSession(w, r, session.ID); err != nil {
		s.logger.Error("failed to set session", "error", err)
		s.renderError(w, "Session Error", "Authentication succeeded, but we couldn't create your session.", err.Error())
		return
	}

	s.logger.Info("user authenticated", "name", user.Name, "email", user.Email)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := s.sessionManager.GetSessionID(r)
	if sessionID != "" {
		s.db.DeleteSession(sessionID)
	}

	s.sessionManager.ClearSession(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	pipes, err := s.db.GetUserPipes(user.ID)
	if err != nil {
		s.logger.Error("failed to get pipes", "user_id", user.ID, "error", err)
		http.Error(w, "Failed to load pipes", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"User":  user,
		"Pipes": pipes,
	}

	w.Header().Set("Content-Type", "text/html")
	s.templates.ExecuteTemplate(w, "dashboard.html", data)
}

func (s *Server) handlePipeEditor(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement pipe editor
	w.Write([]byte("Pipe editor - coming soon!"))
}

func (s *Server) handleAPIMe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleAPIPipes(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		pipes, err := s.db.GetUserPipes(user.ID)
		if err != nil {
			http.Error(w, "Failed to load pipes", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipes)

	case "POST":
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Config      string `json:"config"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Config == "" {
			req.Config = `{"version":"1","nodes":[],"connections":[],"settings":{"enabled":false}}`
		}

		pipe, err := s.db.CreatePipe(user.ID, req.Name, req.Description, req.Config, false)
		if err != nil {
			http.Error(w, "Failed to create pipe", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(pipe)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAPIPipe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract pipe ID from path
	pipeID := r.URL.Path[len("/api/pipes/"):]

	switch r.Method {
	case "GET":
		pipe, err := s.db.GetPipe(pipeID)
		if err != nil || pipe == nil {
			http.Error(w, "Pipe not found", http.StatusNotFound)
			return
		}

		if pipe.UserID != user.ID && !pipe.IsPublic {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipe)

	case "PUT":
		pipe, err := s.db.GetPipe(pipeID)
		if err != nil || pipe == nil {
			http.Error(w, "Pipe not found", http.StatusNotFound)
			return
		}

		if pipe.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var req struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Config      map[string]interface{} `json:"config"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Name != "" {
			pipe.Name = req.Name
		}
		if req.Description != "" {
			pipe.Description = req.Description
		}
		if req.Config != nil {
			configJSON, _ := json.Marshal(req.Config)
			pipe.Config = string(configJSON)
		}

		if err := s.db.UpdatePipe(pipe); err != nil {
			http.Error(w, "Failed to update pipe", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case "DELETE":
		pipe, err := s.db.GetPipe(pipeID)
		if err != nil || pipe == nil {
			http.Error(w, "Pipe not found", http.StatusNotFound)
			return
		}

		if pipe.UserID != user.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if err := s.db.DeletePipe(pipeID); err != nil {
			http.Error(w, "Failed to delete pipe", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAPINodeTypes(w http.ResponseWriter, r *http.Request) {
	registry := engine.NewRegistry()
	nodes := registry.GetAll()

	var nodeTypes []map[string]interface{}
	for _, node := range nodes {
		nodeTypes = append(nodeTypes, map[string]interface{}{
			"type":        node.Type(),
			"label":       node.Label(),
			"description": node.Description(),
			"category":    node.Category(),
			"schema":      node.GetConfigSchema(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeTypes)
}

// Helper functions

func (s *Server) renderError(w http.ResponseWriter, title, message, details string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)

	data := map[string]interface{}{
		"Title":   title,
		"Message": message,
		"Details": details,
	}

	s.templates.ExecuteTemplate(w, "error.html", data)
}
