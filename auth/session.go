package auth

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/kierank/pipes/config"
	"github.com/kierank/pipes/store"
)

type SessionManager struct {
	store *sessions.CookieStore
	db    *store.DB
	cfg   *config.Config
}

func NewSessionManager(cfg *config.Config, db *store.DB) *SessionManager {
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Env == "production",
	}

	return &SessionManager{
		store: store,
		db:    db,
		cfg:   cfg,
	}
}

func (sm *SessionManager) SetSession(w http.ResponseWriter, r *http.Request, sessionID string) error {
	session, _ := sm.store.Get(r, sm.cfg.SessionCookieName)
	session.Values["session_id"] = sessionID
	return session.Save(r, w)
}

func (sm *SessionManager) GetSessionID(r *http.Request) (string, error) {
	session, err := sm.store.Get(r, sm.cfg.SessionCookieName)
	if err != nil {
		return "", err
	}

	sessionID, ok := session.Values["session_id"].(string)
	if !ok {
		return "", nil
	}

	return sessionID, nil
}

func (sm *SessionManager) ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, _ := sm.store.Get(r, sm.cfg.SessionCookieName)
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func (sm *SessionManager) GetCurrentUser(r *http.Request) (*store.User, error) {
	sessionID, err := sm.GetSessionID(r)
	if err != nil || sessionID == "" {
		return nil, nil
	}

	session, err := sm.db.GetSessionByID(sessionID)
	if err != nil || session == nil {
		return nil, err
	}

	user, err := sm.db.GetUserByID(session.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
