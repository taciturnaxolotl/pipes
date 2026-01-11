package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/kierank/pipes/auth"
	"github.com/kierank/pipes/config"
	"github.com/kierank/pipes/engine"
	"github.com/kierank/pipes/store"
	"github.com/mmcdole/gofeed"
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
	mux.HandleFunc("/api/executions/", s.sessionManager.RequireAuth(s.handleAPIExecution))
	mux.HandleFunc("/api/feed-info", s.sessionManager.RequireAuth(s.handleAPIFeedInfo))

	// Public feed routes
	mux.HandleFunc("/feeds/", s.handlePublicFeed)

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
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Extract pipe ID from path
	pipeID := r.URL.Path[len("/pipes/"):]
	if len(pipeID) > 5 && pipeID[len(pipeID)-5:] == "/edit" {
		pipeID = pipeID[:len(pipeID)-5]
	}

	pipe, err := s.db.GetPipe(pipeID)
	if err != nil || pipe == nil {
		s.renderError(w, "Pipe Not Found", "The pipe you're looking for doesn't exist or has been deleted.", "")
		return
	}

	if pipe.UserID != user.ID {
		s.renderError(w, "Access Denied", "You don't have permission to access this pipe.", "")
		return
	}

	data := map[string]interface{}{
		"User": user,
		"Pipe": pipe,
	}

	w.Header().Set("Content-Type", "text/html")
	s.templates.ExecuteTemplate(w, "editor.html", data)
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
	path := r.URL.Path[len("/api/pipes/"):]

	// Check if it's an execute request
	if len(path) > 8 && path[len(path)-8:] == "/execute" {
		pipeID := path[:len(path)-8]
		s.handlePipeExecute(w, r, pipeID, user)
		return
	}

	// Check if it's an executions request
	if len(path) > 11 && path[len(path)-11:] == "/executions" {
		pipeID := path[:len(path)-11]
		s.handlePipeExecutions(w, r, pipeID, user)
		return
	}

	pipeID := path

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
			IsPublic    *bool                  `json:"is_public"`
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
		if req.IsPublic != nil {
			pipe.IsPublic = *req.IsPublic
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

func (s *Server) handleAPIFeedInfo(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url parameter required", http.StatusBadRequest)
		return
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(url, r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse feed: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"title":       feed.Title,
		"description": feed.Description,
		"link":        feed.Link,
		"item_count":  len(feed.Items),
	})
}

func (s *Server) handlePipeExecute(w http.ResponseWriter, r *http.Request, pipeID string, user *store.User) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pipe, err := s.db.GetPipe(pipeID)
	if err != nil || pipe == nil {
		http.Error(w, "Pipe not found", http.StatusNotFound)
		return
	}

	if pipe.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Execute the pipe
	executor := engine.NewExecutor(s.db)
	executionID, err := executor.Execute(r.Context(), pipeID, "manual")
	if err != nil {
		s.logger.Error("pipe execution failed", "pipe_id", pipeID, "error", err)
		http.Error(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"executionId": executionID,
		"status":      "started",
	})
}

func (s *Server) handlePipeExecutions(w http.ResponseWriter, r *http.Request, pipeID string, user *store.User) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pipe, err := s.db.GetPipe(pipeID)
	if err != nil || pipe == nil {
		http.Error(w, "Pipe not found", http.StatusNotFound)
		return
	}

	if pipe.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get limit from query params
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	executions, err := s.db.GetPipeExecutions(pipeID, limit)
	if err != nil {
		s.logger.Error("failed to get executions", "pipe_id", pipeID, "error", err)
		http.Error(w, "Failed to get executions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

func (s *Server) handleAPIExecution(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract execution ID from path
	path := r.URL.Path[len("/api/executions/"):]

	// Check if it's a logs request
	if len(path) > 5 && path[len(path)-5:] == "/logs" {
		executionID := path[:len(path)-5]
		s.handleExecutionLogs(w, r, executionID, user)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func (s *Server) handleExecutionLogs(w http.ResponseWriter, r *http.Request, executionID string, user *store.User) {
	// Get the execution to check ownership
	exec, err := s.db.GetExecution(executionID)
	if err != nil {
		s.logger.Error("failed to get execution", "execution_id", executionID, "error", err)
		http.Error(w, "Failed to get execution", http.StatusInternalServerError)
		return
	}

	if exec == nil {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	// Verify user owns the pipe
	pipe, err := s.db.GetPipe(exec.PipeID)
	if err != nil || pipe == nil {
		http.Error(w, "Pipe not found", http.StatusNotFound)
		return
	}

	if pipe.UserID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get logs
	logs, err := s.db.GetExecutionLogs(executionID)
	if err != nil {
		s.logger.Error("failed to get logs", "execution_id", executionID, "error", err)
		http.Error(w, "Failed to get logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handlePublicFeed(w http.ResponseWriter, r *http.Request) {
	// Parse path: /feeds/{id}.{format} or /feeds/{id}/{format}
	path := strings.TrimPrefix(r.URL.Path, "/feeds/")
	if path == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var pipeID, format string

	// Check for extension format: id.json or id.rss
	if strings.Contains(path, ".") {
		parts := strings.SplitN(path, ".", 2)
		pipeID = parts[0]
		format = parts[1]
	} else if strings.Contains(path, "/") {
		// Check for path format: id/json or id/rss
		parts := strings.SplitN(path, "/", 2)
		pipeID = parts[0]
		format = parts[1]
	} else {
		// Default to json if no format specified
		pipeID = path
		format = "json"
	}

	// Look up pipe by ID
	pipe, err := s.db.GetPipe(pipeID)
	if err != nil {
		s.logger.Error("failed to get pipe", "pipe_id", pipeID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if pipe == nil || !pipe.IsPublic {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	// Get the cached output
	output, err := s.db.GetPipeOutput(pipe.ID, format)
	if err != nil {
		s.logger.Error("failed to get pipe output", "pipe_id", pipe.ID, "format", format, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Auto-run if no output exists
	if output == nil {
		executor := engine.NewExecutor(s.db)
		_, err := executor.Execute(r.Context(), pipe.ID, "auto")
		if err != nil {
			s.logger.Error("auto-execute failed", "pipe_id", pipe.ID, "error", err)
			http.Error(w, "Failed to generate feed", http.StatusInternalServerError)
			return
		}

		// Try to get output again
		output, err = s.db.GetPipeOutput(pipe.ID, format)
		if err != nil || output == nil {
			http.Error(w, "Feed not available in requested format", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", output.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write([]byte(output.Content))
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
