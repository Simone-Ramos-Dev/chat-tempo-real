package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Upgrader converte uma conexão HTTP em WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // ⚠ Em produção, restrinja as origens
	},
}

// Client representa um usuário conectado
type Client struct {
	conn     *websocket.Conn
	username string
}

// Hub gerencia os clientes conectados e as mensagens
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan string
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan string),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Novo cliente conectado: %s\n", client.username)
			h.broadcast <- fmt.Sprintf("🔵 %s entrou no chat", client.username)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.conn.Close()
				log.Printf("Cliente desconectado: %s\n", client.username)
				h.broadcast <- fmt.Sprintf("🔴 %s saiu do chat", client.username)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.conn.WriteMessage(websocket.TextMessage, []byte(message))
				if err != nil {
					log.Printf("Erro ao enviar mensagem para %s: %v", client.username, err)
					client.conn.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Erro no upgrade para WebSocket:", err)
		return
	}

	// Gera username automático
	username := fmt.Sprintf("User-%d", time.Now().UnixNano()%10000)
	client := &Client{conn: conn, username: username}
	hub.register <- client

	defer func() {
		hub.unregister <- client
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Erro ao ler mensagem de %s: %v", client.username, err)
			break
		}
		hub.broadcast <- fmt.Sprintf("💬 %s: %s", client.username, string(msg))
	}
}

func main() {
	hub := newHub()
	go hub.Run()

	// Porta padrão
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Rotas
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(hub, w, r)
	})

	addr := ":" + port
	log.Printf("🚀 Servidor de chat iniciado em http://localhost:%s\n", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Erro no servidor:", err)
	}
}
