package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var port = ":11811"

// Player represents a connected player's data
type Player struct {
	ID       string          `json:"id"`
	Position [2]float64      `json:"position"` // X, Y position
	conn     *websocket.Conn // WebSocket connection for each player
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Server stores all connected players and manages communication
type Server struct {
	players  map[string]*Player
	mu       sync.Mutex
	upgrader websocket.Upgrader
}

func newServer() *Server {
	return &Server{
		players: make(map[string]*Player),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// Broadcast all players' data to every client
func (s *Server) broadcastPlayers() {
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		s.mu.Lock()
		for _, player := range s.players {
			s.sendPlayerUpdates(player.ID)
		}
		s.mu.Unlock()
	}
}

// Send the position of all players to a specific player
func (s *Server) sendPlayerUpdates(playerID string) {
	player, ok := s.players[playerID]
	if !ok {
		return
	}

	// Create a connection to the player
	conn := player.conn

	// Send all players' positions to the client
	var playersData []*Player
	for _, p := range s.players {
		if p.ID == playerID {
			continue
		}
		playersData = append(playersData, p)
	}

	msg := Message{
		Type: "players_data",
		Data: playersData,
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("Error sending data to player %s: %v", playerID, err)
		conn.Close()
		delete(s.players, playerID)
	}
}

// Handle client connections and receive their position data
func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Get a unique player ID
	playerID := fmt.Sprintf("Player-%d", time.Now().UnixNano())
	player := &Player{
		ID:       playerID,
		Position: [2]float64{0, 0}, // Initial position
		conn:     conn,             // Save the WebSocket connection
	}

	// Add the player to the server's list
	s.mu.Lock()
	s.players[playerID] = player
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.players, playerID)
		s.mu.Unlock()
	}()

	log.Printf("%s connected", playerID)

	// Handle incoming messages (client sends position data)
	for {
		var res Message
		if err := conn.ReadJSON(&res); err != nil {
			log.Printf("Error reading position from %s: %v", playerID, err)
			break
		}

		switch res.Type {
		case "update_position":
			// Update player's position
			var pos [2]float64
			err := parseData(res.Data, &pos) 
			if err != nil {
				fmt.Println("Can't update player position! Parsing error!")
				continue
			}
			s.mu.Lock()
			player.Position = pos
			s.mu.Unlock()

			log.Printf("Player %s updated position to: %+v", playerID, pos)
			break
		}
	}
}

func parseData(data interface{}, v interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}

func main() {
	server := newServer()

	http.HandleFunc("/ws", server.handleConnection)

	go server.broadcastPlayers()

	log.Println("Server started on", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
