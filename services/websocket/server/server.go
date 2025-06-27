package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vimian/remote-timer/services/websocket/config"
)

type session struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Viewers []uuid.UUID
}

type actor struct {
	SessionID uuid.UUID
	Conn      *websocket.Conn
}

var sessions = make(map[uuid.UUID]session)
var actors = make(map[uuid.UUID]actor)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { // TODO: remove this in prod
		return true
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
	for { // Reroll until unique UUID
		actorID = uuid.New()
		if _, exists := actors[actorID]; !exists {
			actors[actorID] = actor{
				Conn: conn,
			}
			break
		}
	}

	log.Printf("New connection established: %s", actorID)

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
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
		if actors[authorID].SessionID != uuid.Nil { // Already has a session
			sessionID = actors[authorID].SessionID
		} else {
			for { // Reroll until unique UUID
				sessionID = uuid.New()
				if _, exists := sessions[sessionID]; !exists {
					break
				}
			}
			newSession := session{
				ID:      sessionID,
				OwnerID: authorID,
				Viewers: []uuid.UUID{},
			}
			sessions[sessionID] = newSession
			actors[authorID] = actor{
				SessionID: sessionID,
				Conn:      actors[authorID].Conn,
			}
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
		if err := actors[authorID].Conn.WriteMessage(messageType, resMsg); err != nil {
			log.Println("Error writing message:", err)
			delete(actors, authorID)
			return
		}
	default:
		log.Printf("Unhandled event: %s", msgData.Event)
	}

	debugState()
}

func debugState() {
	log.Println("Current actors:")
	for id, actor := range actors {
		log.Printf("Actor ID: %s, Session ID: %s", id, actor.SessionID)
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
