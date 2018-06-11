package main

import (
	"fmt"
    _"github.com/go-sql-driver/mysql"
    "github.com/gorilla/websocket"
    "time"
    "net/http"
    "log"
)

const (
    writeWait = 10 * time.Second    // Time allowed to write a message to the peer.
    pongWait = 60 * time.Second     // Time allowed to read the next pong message from the peer.
    pingPeriod = (pongWait * 9) / 10    // Send pings to peer with this period. Must be less than pongWait.
    maxMessageSize = 512    // Maximum message size allowed from peer.
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:   1024,
    WriteBufferSize:  1024,
    HandshakeTimeout: 5 * time.Second,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

var manager = Hub{
    register: make(chan *Client),
    unregister: make(chan *Client),
    abnormal : make(chan *Client),
    clients: make(map[*Client]bool),
    rooms: make(map[int]RoomNumber),
}

func (h *Hub) run() {
	for {
		select {
            case client := <-h.register://有新链接加入
                fmt.Println("有新链接加入",client)
                h.clients[client] = true
            case client := <-h.unregister://有链接正常退出
                fmt.Println("链接断掉了",client)
                if _, ok := h.clients[client]; ok {
                    delete(h.clients, client)
                }
            case client := <- h.abnormal://有链接异常退出
                if _, ok := h.clients[client]; ok {
                    delete(h.clients, client)
                }
                fmt.Println(h.rooms,"查看一下区别",client.hub.rooms)
                for roomsKey,roomsValue := range client.hub.rooms {
                    fmt.Println("房间key",roomsKey,"房间value",roomsValue)
                    //看是不是主播断了
                    if roomsValue.master == client.conn {
                        viewers := roomsValue.viewer
                        client.hub.rooms[roomsKey] = RoomNumber{viewer:viewers}
                        for _,viewer := range roomsValue.viewer {
                            response := ResponseData { 
                                Event : "disconnect",
                            }
                            viewer.WriteJSON(response)
                        }
                    }else{//否则就是观众断掉了
                        for k,viewer := range roomsValue.viewer {
                            if client.conn == viewer{
                                delete(roomsValue.viewer,k)
                            }
                        }
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
    client := &Client{hub:hub,conn:ws}
    client.hub.register <- client
    go client.GohomeHub()
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
    mux.HandleFunc("/gohome", func(w http.ResponseWriter, r *http.Request) {
        res(hub, w, r)
    })
    log.Println("Listening...")
    http.ListenAndServe(":8800",mux)
}
