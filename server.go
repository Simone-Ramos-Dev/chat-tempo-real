package main

import (
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

// Define o upgrader para converter requisições HTTP em conexões WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permite qualquer origem
	},
}

// Hub para gerenciar clientes (conexões)
type Hub struct {
	// Mapeia conexões de clientes
	clients map[*websocket.Conn]bool
	// Canal para registrar novas mensagens
	broadcast chan []byte
	// Canais para registrar e desregistrar clientes
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

// Inicializa e executa o hub
func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run gerencia os canais e o estado do hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// Adiciona um novo cliente
			h.clients[client] = true
			log.Println("Novo cliente conectado!")
		case client := <-h.unregister:
			// Remove o cliente
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				log.Println("Cliente desconectado.")
			}
		case message := <-h.broadcast:
			// Envia a mensagem para todos os clientes
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Erro ao enviar mensagem: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
		}
	}
}

// HandleWebSocket lida com as conexões WebSocket
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade de WebSocket falhou:", err)
		return
	}

	// Registra o cliente no hub
	hub.register <- conn

	// Garante que o cliente seja removido ao sair
	defer func() {
		hub.unregister <- conn
	}()

	// Lê as mensagens do cliente e as envia para o canal de broadcast
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Erro de leitura:", err)
			break
		}
		hub.broadcast <- msg
	}
}

func main() {
	hub := newHub()
	go hub.Run()

	// Roteamento
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(hub, w, r)
	})

	log.Println("Servidor de chat iniciado em :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}