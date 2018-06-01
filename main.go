package main

import (
	"fmt"
    "github.com/garyburd/redigo/redis"//redis
    _"github.com/go-sql-driver/mysql"
    "github.com/gorilla/websocket"
    "time"
    "net/http"
    "log"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:   1024,
    WriteBufferSize:  1024,
    HandshakeTimeout: 5 * time.Second,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

type Hub struct {
    clients map[*Client]bool
    broadcast chan []byte
    register chan *Client
    unregister chan *Client
}

type Client struct {
    hub *Hub
    conn *websocket.Conn
    send chan []byte
}

var manager = Hub{
    broadcast:  make(chan []byte),
    register:   make(chan *Client),
    unregister: make(chan *Client),
    clients:    make(map[*Client]bool),
}

func (h *Hub) run() {
	for {
		select {
        case client := <-h.register://有新链接加入
            fmt.Println("有新链接加入",client)
			h.clients[client] = true
		case client := <-h.unregister://有链接断开
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast://有新消息
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func res(hub *Hub,w http.ResponseWriter,r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("错误",err)
        log.Fatal(err)
    }
    defer ws.Close()
    client := &Client{hub:hub,conn:ws,send:make(chan []byte,256)}
    client.hub.register <- client
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		client.hub.broadcast <- message
        fmt.Println("数据",string(message),err)
        fmt.Println("链接",hub.clients)
	}
}

func main(){
    defer func(){
        if err := recover();err != nil{
            fmt.Println("panic is success",err)
        }
    }()
    hub := &manager
    go hub.run()
    mux := http.NewServeMux()
	mux.HandleFunc("/res", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("???")
		res(hub, w, r)
	})
    log.Println("Listening...")
    http.ListenAndServe(":8800",mux)
}

func redis_example(){
    c, err := redis.Dial("tcp", "127.0.0.1:6379")
    checkErr(err)
    defer c.Close()
    c.Do("auth","cosin")
    /* _, err = c.Do("SET", "mykey", "superWang")
    if err != nil {
        fmt.Println("redis set failed:", err)
    } */
    username, err := redis.String(c.Do("GET", "author"))
    if err != nil {
        fmt.Println("redis get failed:", err)
    } else {
        fmt.Printf("Get mykey: %v \n", username)
    }
}

func checkErr(err error){
    if err != nil {
        panic(err)
    }
}
