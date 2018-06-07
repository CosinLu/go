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

//解析数据请求
type requestData struct {
    Event string `json:"event"`
    RoomId int `json:"roomid"`
    Uid int `json:"uid"`
    Lon string `json:"lon"`
    Lat string `json:"lat"`
    Flag int `json:"flag"`
}

//返回数据格式
type responseData struct {
    Event string `json:"event"`
    RoomId int `json:"roomid"`
    Uid int `json:"uid"`
    Lon string `json:"lon"`
    Lat string `json:"lat"`
    Msg string `json:"msg"`
    Error interface{} `json:"error"`
}

type roomNumber struct {
	master *websocket.Conn
	viewer []map[int]*websocket.Conn
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
    rooms map[int]roomNumber
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
    rooms: make(map[int]roomNumber),
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
        res(hub, w, r)
    })
    log.Println("Listening...")
    http.ListenAndServe(":8800",mux)
}

func (c *Client) readPump() {
    defer func() {
        fmt.Println("要断了是吗")
        if err := recover();err != nil {
            response := responseData { 
                Event : "error", 
                Msg : "网络异常", 
                Error : err,
            } 
            c.conn.WriteJSON(response)
        }
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(maxMessageSize)
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
        switch request.Event {
            case "create":  //创建一个回家的连接
                if _,ok := c.hub.rooms[request.RoomId];!ok {
                    room := roomNumber{master:c.conn}
                    c.hub.rooms[request.RoomId] = room
                }
            case "location": //用户持续上报位置
                if rooms,ok := c.hub.rooms[request.RoomId];ok {
                    fmt.Println("查看一下当前房间的连接数",rooms)
                    for _,viewer := range rooms.viewer {
                        for _,v := range viewer {
                            response := responseData { 
                                Event : "location",
                                Lat : request.Lat , 
                                Lon : request.Lon, 
                            }
                            v.WriteJSON(response)
                        }
                        fmt.Println("查看每个链接的效果",viewer,"返回的数据")
                    }
                }
            case "join" : //有人加入查看用户上报回家位置map[int][]map[string][]map[int]*websocket.Conn
                if _,ok := c.hub.rooms[request.RoomId];ok {
                    viewer := make(map[int]*websocket.Conn)
                    viewers := c.hub.rooms[request.RoomId].viewer
                    viewer[request.Flag] = c.conn
                    viewers = append(viewers,viewer)
                    master := c.hub.rooms[request.RoomId].master
                    c.hub.rooms[request.RoomId] = roomNumber{master:master,viewer:viewers}
                }else{
                    response := responseData { 
                        Event : "error", 
                        Msg : "没有该房间", 
                    } 
                    c.conn.WriteJSON(response)
                }
            case "getLocation": //有人主动查看当前用户的位置
                if _,ok := c.hub.rooms[request.RoomId];ok {
                    //检查回家人是否还在保持连接状态
                    master := c.hub.rooms[request.RoomId].master
                    if master != nil {
                        response := responseData { 
                            Event : "getLocation", 
                        } 
                        master.WriteJSON(response)
                    }else{
                        response := responseData { 
                            Event : "error", 
                            Msg : "咦?是不是来错地方了?", 
                        } 
                        c.conn.WriteJSON(response)
                    }
                }
            case "leave": //有人退出查看用户实时推送位置
                if rooms,ok := c.hub.rooms[request.RoomId]; ok {
                    for _,viewer := range rooms.viewer {
                        for k,v := range viewer {
                            if v == c.conn {
                                delete(viewer,k)
                                c.hub.unregister <- c 
                                c.conn.Close()
                            }
                        }
                        fmt.Println("查看当前的观众",viewer)    
                    }
                    fmt.Println("查看每个链接的效果",rooms)
                }
            case "end": //用户主动结束上报回家事件
                if rooms,ok := c.hub.rooms[request.RoomId]; ok {
                    master := rooms.master
                    if master != nil {
                        for _,viewer := range rooms.viewer {
                            for _,v := range viewer {
                                response := responseData { 
                                    Event : "end",
                                }
                                v.WriteJSON(response)
                            }
                        }
                        c.hub.unregister <- c
                        c.conn.Close()
                        delete(c.hub.rooms,request.RoomId)
                    }
                    fmt.Println("查看每个链接的效果",rooms)
                }
        }
        fmt.Println("查看一下现在有多少个房间",c.hub.rooms)
    }
}