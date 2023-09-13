package main

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

type MessageType string

const (
	Register MessageType = "register"
	Submit   MessageType = "submit"
	Reveal   MessageType = "reveal"
	Clear    MessageType = "clear"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from all origins
		},
	}

	// Create a channel for incoming messages
	incomingMessages = make(chan Message, 1000) // Adjust buffer size as needed
	clients          = make(map[*websocket.Conn]string)
	scores           = make(map[*websocket.Conn]int)
	logger           *zap.Logger
)

// Message represents a structured message with a username and poker points.
type Message struct {
	Type MessageType `json:"type"` // Message type
	Data Data        `json:"data"`
	Conn *websocket.Conn
}

type Data struct {
	Username    *string `json:"username"`
	PokerPoints *int    `json:"pokerPoints"`
}

func main() {
	logger, _ = zap.NewDevelopment()
	defer logger.Sync()

	e := echo.New()

	// Start a goroutine to handle incoming messages
	go handleMessageUpdates()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			logger.Error("WebSocket upgrade error", zap.Error(err))
			return err
		}
		defer conn.Close()

		logger.Debug("WebSocket connection established")

		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				logger.Error("WebSocket read error", zap.Error(err))
				return err
			}
			msg.Conn = conn
			incomingMessages <- msg

		}
	})

	e.Start(":8080")
}

func handleMessageUpdates() {
	for msg := range incomingMessages {
		// Handle incoming message here, e.g., store it in a data structure
		// For simplicity, we'll just print it for now.
		logger.Debug("Received message", zap.Any("msg", msg))
		switch msg.Type {
		case Register:
			// Handle user registration
			logger.Debug("User registered", zap.String("username", *msg.Data.Username))
			clients[msg.Conn] = *msg.Data.Username
			sendToAllClients() // Broadcast registration message
		case Submit:
			// Handle poker point submission
			logger.Debug("Poker point submitted", zap.Int("pokerPoints", *msg.Data.PokerPoints))
			scores[msg.Conn] = *msg.Data.PokerPoints
			sendToAllClients() // Broadcast submission message
		case Reveal:
			// Handle reveal

		case Clear:
			// Handle clear

		default:
			logger.Warn("Unsupported message type", zap.String("type", string(msg.Type)))
		}
	}
}

func sendToAllClients() {

	response := make([]Data, 0)

	for client := range clients {
		username := clients[client]
		score := scores[client]
		data := Data{Username: &username, PokerPoints: &score}
		response = append(response, data)
	}
	logger.Debug("Current data", zap.Any("data", response))
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}
