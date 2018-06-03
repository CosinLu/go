package main

import (
	"fmt"
    _"github.com/go-sql-driver/mysql"
    "github.com/gorilla/websocket"
    "time"
    "net/http"
    "log"
    "reflect"
)

const (
    // Time allowed to write a message to the peer.
    writeWait = 10 * time.Second

    // Time allowed to read the next pong message from the peer.
    pongWait = 60 * time.Second

    // Send pings to peer with this period. Must be less than pongWait.
    pingPeriod = (pongWait * 9) / 10

    // Maximum message size allowed from peer.
    maxMessageSize = 512
)

//设定一个解析数据请求的格式
type requestData struct {
    Event string `json:"event"`
    RoomId int `json:"roomid"`
    Uid int `json:"uid"`
    Lon string `json:"lon"`
    Lat string `json:"lat"`
}

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
    broadcast chan requestData
    register chan *Client
    unregister chan *Client
    rooms map[int][]int
}

type Client struct {
    hub *Hub
    conn *websocket.Conn
    send chan requestData
}

var manager = Hub{
    broadcast: make(chan requestData),
    register: make(chan *Client),
    unregister: make(chan *Client),
    clients: make(map[*Client]bool),
    rooms: make(map[int][]int),
}

func (h *Hub) run() {
	for {
		select {
        case client := <-h.register://有新链接加入
            fmt.Println("有新链接加入",client)
			h.clients[client] = true
		case client := <-h.unregister://有链接断开
		    fmt.Println("发完就断开了？")
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast://有新消息
		fmt.Println("有数据的",message,reflect.TypeOf(message))
			for client := range h.clients {
				select {
				case client.send <- message:
				    fmt.Println("检查发送的消息是什么",message)
				default:
				    fmt.Println("会每次都关闭吗")
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
    client := &Client{hub:hub,conn:ws,send:make(chan requestData)}
    fmt.Println("应该是从这边先进入的链接")
    client.hub.register <- client
    go client.readPump()
    //go client.writePump()
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

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
    for {
        var request requestData
        err := c.conn.ReadJSON(&request)
        if err != nil {
            fmt.Println("难道是读取时候出错了吗",err)
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break
        }
        if err != nil {
            fmt.Println("读取json数据失败",err)
        }
        fmt.Println("循环读取数据",request,request.Event,err,reflect.TypeOf(request))
        c.hub.broadcast <- request
    }
}

/*func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // The hub closed the channel.
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // Add queued chat messages to the current websocket message.
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}*/
