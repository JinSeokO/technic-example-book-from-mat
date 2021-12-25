package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
)

type client struct {
	socket *websocket.Conn
	send   chan []byte
	room   *room
}

func (c *client) read() {
	defer c.socket.Close()

	for {
		_, msg, err := c.socket.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		c.room.forward <- msg
	}
}

func (c *client) write() {
	defer c.socket.Close()

	for msg := range c.send {
		err := c.socket.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

type room struct {
	// forward 는 수신 메시지를 보관하는 채널이며
	// 수신한 메시지는 다른 클라이언트로 전달돼야 한다
	forward chan []byte
	// join 은 방에 들어오려는 클라이언트를 위한 채널이다
	join chan *client
	// leave 는 방에서 나가길 원하는 클라이언트를 위한 채널이다
	leave chan *client
	// clients 는 현재 채팅방에 있는 모든 클라이언트를 보유한다
	clients map[*client]bool
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// 압정
			r.clients[client] = true
		case client := <-r.leave:
			// 퇴장
			delete(r.clients, client)
			close(client.send)
		case msg := <-r.forward:
			// 모든 클라이언트에게 메시지 전달
			for client := range r.clients {
				client.send <- msg
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:    socketBufferSize,
	WriteBufferSize:   messageBufferSize,
	EnableCompression: false,
}

func (r *room) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	socket, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Fatal("ServeHttp: ", err)
		return
	}

	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}

	r.join <- client
	defer func() {
		r.leave <- client
	}()

	go client.write()
	client.read()
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

type templateHandler struct {
	once     sync.Once
	fileName string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			w.WriteHeader(500)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				log.Fatal(err)
			}
		}
		t.templ = template.Must(template.ParseFiles(filepath.Join(dir, "websocket", "templates", t.fileName)))
	})
	t.templ.Execute(w, nil)
}

func main() {
	r := newRoom()
	http.Handle("/", &templateHandler{fileName: "chat.html"})
	// 웹 서버 시작
	http.Handle("/room", r)
	go r.run()
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalln(err)
	}
}
