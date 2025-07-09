package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vimian/remote-timer/services/websocket/config"
)

type session struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Viewers []uuid.UUID
}

func (s session) Close() {
	sessionLock.Lock()
	delete(sessions, s.ID)
	sessionLock.Unlock()
}

type client struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	Conn      *websocket.Conn
	heartbeat chan struct{}
}

func (c *client) Disconnect() {
	close(c.heartbeat)
	if c.Conn != nil {
		c.Conn.Close()
	}
	if c.SessionID != uuid.Nil {
		sessionLock.RLock()
		session, exists := sessions[c.SessionID]
		sessionLock.RUnlock()
		if exists && session.OwnerID == c.ID {
			session.Close()
		}
	}
	clientLock.Lock()
	delete(clients, c.ID)
	clientLock.Unlock()
}

func (c *client) SetSessionID(sessionID uuid.UUID) {
	c.SessionID = sessionID
	clientLock.Lock()
	clients[c.ID] = *c
	clientLock.Unlock()
}

var (
	pingPeriodDuration = time.Second * time.Duration(10)
	pongWaitDuration   = pingPeriodDuration + time.Second*time.Duration(10)
)

func (c *client) StartHeartbeat(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(pongWaitDuration))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWaitDuration))
		return nil
	})

	go func() {
		ticker := time.NewTicker(pingPeriodDuration)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Ping failed for %s: %v", c.ID, err)
					c.Disconnect()
					return
				}
			case <-c.heartbeat:
				return
			}
		}
	}()
}

var (
	sessions    = make(map[uuid.UUID]session)
	sessionLock sync.RWMutex
	clients     = make(map[uuid.UUID]client)
	clientLock  sync.RWMutex
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		log.Printf("Origin: %s", r.Header.Get("Origin"))
		return true // TODO: remove this in prod
		/*origin := r.Header.Get("Origin")
		trustedOrigins := []string{
			"https://example.com",
			"https://another-trusted-origin.com",
		}
		for _, trustedOrigin := range trustedOrigins {
			if origin == trustedOrigin {
				return true
			}
		}
		return false*/
	},
}

func handleWebsocket(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	var actorID uuid.UUID
	var heartbeat chan struct{}
	for { // Reroll until unique UUID
		actorID = uuid.New()
		clientLock.RLock()
		_, exists := clients[actorID]
		clientLock.RUnlock()
		if !exists {
			clientLock.Lock()
			var newClient client = client{
				ID:        actorID,
				Conn:      conn,
				heartbeat: make(chan struct{}),
			}
			newClient.StartHeartbeat(conn)
			clients[actorID] = newClient
			clientLock.Unlock()
			break
		}
	}

	log.Printf("New connection established: %s", actorID)

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			clientLock.RLock()
			client, ok := clients[actorID]
			clientLock.RUnlock()
			if ok {
				client.Disconnect()
			}
			close(heartbeat)
			log.Println("Error reading message:", err)
			break
		}

		handleMessage(actorID, messageType, msg)
	}
}

type message struct {
	Event    string `json:"event"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsValid  bool   `json:"is_valid"`
	Duration int    `json:"duration"`
	Title    string `json:"title"`
	Value    string `json:"value"`
	OfType   string `json:"of_type"`
}

func handleMessage(authorID uuid.UUID, messageType int, msg []byte) {
	log.Printf("Received message from %s: %s", authorID, msg)

	var msgData message
	if err := json.Unmarshal(msg, &msgData); err != nil {
		log.Println("Error unmarshalling message:", err)
		return
	}

	switch msgData.Event {
	case "create_session":
		var sessionID uuid.UUID
		clientLock.RLock()
		var client client = clients[authorID]
		if client.SessionID != uuid.Nil { // Already has a session
			sessionID = client.SessionID
			clientLock.RUnlock()
		} else {
			clientLock.RUnlock()
			sessionLock.RLock()
			for { // Reroll until unique UUID
				sessionID = uuid.New()
				if _, exists := sessions[sessionID]; !exists {
					break
				}
			}
			sessionLock.RUnlock()
			newSession := session{
				ID:      sessionID,
				OwnerID: authorID,
				Viewers: []uuid.UUID{},
			}
			sessionLock.Lock()
			sessions[sessionID] = newSession
			sessionLock.Unlock()
			client.SetSessionID(sessionID)
		}

		resMsgData := message{
			Event: "session_created",
			ID:    sessionID.String(),
		}
		resMsg, err := json.Marshal(resMsgData)
		if err != nil {
			log.Println("Error marshalling response message:", err)
			return
		}
		if err := client.Conn.WriteMessage(messageType, resMsg); err != nil {
			log.Println("Error writing message:", err)
			client.Disconnect()
			return
		}
	default:
		log.Printf("Unhandled event: %s", msgData.Event)
	}

	debugState() // For debugging purposes, remove in production
}

func debugState() {
	log.Println("Current clients:")
	for id, client := range clients {
		log.Printf("Client ID: %s, Session ID: %s", id, client.SessionID)
	}
	log.Println("\nCurrent sessions:")
	for id, session := range sessions {
		log.Printf("Session ID: %s, Owner ID: %s, Viewers: %v", id, session.OwnerID, session.Viewers)
	}
}

// Listen starts the websocket server and listens for incoming connections.
func Listen(websocketConfig *config.Websocket) {
	http.HandleFunc("/ws", handleWebsocket)
	log.Fatal(http.ListenAndServe(":"+websocketConfig.Port, nil))
}
