package handler

import (
	"net/http"

	"github.com/dafibh/fortuna/fortuna-backend/internal/websocket"
	ws "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// JWTValidator validates JWT tokens and returns the workspace ID
type JWTValidator interface {
	ValidateToken(token string) (workspaceID int32, err error)
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub            *websocket.Hub
	validator      JWTValidator
	allowedOrigins map[string]bool
	upgrader       ws.Upgrader
}

// NewWebSocketHandler creates a new WebSocketHandler
func NewWebSocketHandler(hub *websocket.Hub, validator JWTValidator, allowedOrigins []string) *WebSocketHandler {
	// Build origin lookup map
	originMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		originMap[origin] = true
	}

	h := &WebSocketHandler{
		hub:            hub,
		validator:      validator,
		allowedOrigins: originMap,
	}

	h.upgrader = ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	return h
}

// checkOrigin validates the request origin against allowed origins
func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Allow requests with no Origin header (e.g., same-origin or non-browser clients)
		return true
	}

	if h.allowedOrigins[origin] {
		return true
	}

	log.Warn().
		Str("origin", origin).
		Msg("WebSocket connection rejected: origin not allowed")
	return false
}

// HandleWS handles WebSocket connection requests at GET /ws
func (h *WebSocketHandler) HandleWS(c echo.Context) error {
	// Get token from query parameter
	token := c.QueryParam("token")
	if token == "" {
		log.Debug().Msg("WebSocket connection rejected: missing token")
		return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
	}

	// Validate JWT and get workspace ID
	workspaceID, err := h.validator.ValidateToken(token)
	if err != nil {
		log.Debug().Err(err).Msg("WebSocket connection rejected: invalid token")
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return err
	}

	// Create client and register with hub
	client := websocket.NewClient(conn, workspaceID, h.hub)
	h.hub.Register(client)

	log.Info().
		Int32("workspace_id", workspaceID).
		Str("client_id", client.ID()).
		Msg("WebSocket client connected")

	// Start read/write pumps in goroutines
	go client.WritePump()
	go client.ReadPump()

	return nil
}
