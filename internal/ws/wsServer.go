package ws

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Blaze5333/cex/internal/db"
	"golang.org/x/net/websocket"
)

type WSServer struct {
	Rdb   *db.RedisConfig
	Rooms map[string]*Room // marketId → room
	Mu    sync.RWMutex
}
type Room struct {
	marketId   string
	clients    map[*websocket.Conn]bool
	cancelFunc context.CancelFunc // to stop Redis subscription
	mu         sync.RWMutex
}

// Runs only while at least one user is in the room
func (ws *WSServer) listenRedis(ctx context.Context, room *Room) {
	pubsub := ws.Rdb.RdbClient.Subscribe(ctx,
		fmt.Sprintf("orderbook:%s", room.marketId),
		fmt.Sprintf("trades:%s", room.marketId),
	)
	defer pubsub.Close()

	for {
		select {
		case msg := <-pubsub.Channel():
			room.broadcast([]byte(msg.Payload))

		case <-ctx.Done():
			log.Printf("Redis listener stopped for %s", room.marketId)
			return
		}
	}
}

func (room *Room) broadcast(data []byte) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	for conn := range room.clients {
		err := websocket.Message.Send(conn, data)
		if err != nil {
			// Will be cleaned up when HandleConnection detects disconnect
			conn.Close()
		}
	}
}

// Called when user opens /ws/BTCUSDT
func (ws *WSServer) HandleConnection(conn *websocket.Conn, marketId string) {
	// Join or create room for this marketId
	room := ws.joinRoom(conn, marketId)

	defer func() {
		ws.leaveRoom(conn, marketId)
		conn.Close()
	}()

	// Block — keep connection open until client disconnects
	for {
		var msg []byte
		err := websocket.Message.Receive(conn, &msg)
		if err != nil {
			break
		}
	}

	_ = room
}

func (ws *WSServer) joinRoom(conn *websocket.Conn, marketId string) *Room {
	ws.Mu.Lock()
	defer ws.Mu.Unlock()

	room, exists := ws.Rooms[marketId]

	if !exists {
		// First user joining this market — create room + start Redis subscription
		ctx, cancel := context.WithCancel(context.Background())
		room = &Room{
			marketId:   marketId,
			clients:    make(map[*websocket.Conn]bool),
			cancelFunc: cancel,
		}
		ws.Rooms[marketId] = room

		// Start listening to Redis only now
		go ws.listenRedis(ctx, room)

		log.Printf("Room created for %s", marketId)
	}

	room.mu.Lock()
	room.clients[conn] = true
	room.mu.Unlock()

	log.Printf("User joined %s, total clients: %d", marketId, len(room.clients))
	return room
}

func (ws *WSServer) leaveRoom(conn *websocket.Conn, marketId string) {
	ws.Mu.Lock()
	defer ws.Mu.Unlock()

	room, exists := ws.Rooms[marketId]
	if !exists {
		return
	}

	room.mu.Lock()
	delete(room.clients, conn)
	count := len(room.clients)
	room.mu.Unlock()

	log.Printf("User left %s, remaining clients: %d", marketId, count)

	// Last user left — stop Redis subscription and destroy room
	if count == 0 {
		room.cancelFunc() // cancels Redis pubsub goroutine
		delete(ws.Rooms, marketId)
		log.Printf("Room destroyed for %s", marketId)
	}
}
