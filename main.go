package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex
	checkboxes [1000]bool
)

type Data struct {
	Checkboxes [1000]bool
}

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/toggle", handleToggle)

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := Data{
		Checkboxes: checkboxes,
	}
	tmpl.Execute(w, data)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()

	broadcastConnectionCount()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			clientsMux.Lock()
			delete(clients, conn)
			clientsMux.Unlock()
			broadcastConnectionCount()
			break
		}
	}
}

func handleToggle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 0 || id >= 100 {
		http.Error(w, "Invalid checkbox ID", http.StatusBadRequest)
		return
	}

	checkboxes[id] = !checkboxes[id]
	broadcastUpdate(id, checkboxes[id])

	w.WriteHeader(http.StatusOK)
}

func broadcastUpdate(id int, checked bool) {
	message := fmt.Sprintf("checkbox:%d:%v", id, checked)
	broadcastMessage(message)
}

func broadcastConnectionCount() {
	count := len(clients)
	message := fmt.Sprintf("connections:%d", count)
	broadcastMessage(message)
}

func broadcastMessage(message string) {
	clientsMux.Lock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
	clientsMux.Unlock()
}

