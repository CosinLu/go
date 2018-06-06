package main

import (
	// "encoding/json"
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
	master map[int]*websocket.Conn
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
    // rooms map[int][]map[string][]map[int]*websocket.Conn
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
    // rooms: make(map[int][]map[string][]map[int]*websocket.Conn),
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
		case message := <-h.broadcast://有新消息
            fmt.Println("有数据的",message)
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
        //判断发送的事件
        // linkArr := make(map[string][]map[int]*websocket.Conn,0)
        // link := make(map[int]*websocket.Conn)
        switch request.Event {
            case "create":  //创建一个回家的连接map[int]map[string][]map[int]*websocket.Conn
                if _,ok := c.hub.rooms[request.RoomId];!ok {
                    /* link[request.Flag] = c.conn
                    linkArr["master"] = append(linkArr["master"],link)
                    c.hub.rooms[request.RoomId] = append(c.hub.rooms[request.RoomId],linkArr) */
                    master := map[int]*websocket.Conn{request.Flag:c.conn}
                    room := roomNumber{master:master}
                    c.hub.rooms[request.RoomId] = room
                }
            case "location": //用户持续上报位置
                /* if rooms,ok := c.hub.rooms[request.RoomId];ok {
                    fmt.Println("查看一下当前房间的连接数",rooms)
                    for _,v := range rooms {
                        if viewer,ok := v["other"];ok {
                            response := responseData { 
                                Event : "location",
                                Lat : request.Lat , 
                                Lon : request.Lon, 
                            }
                            fmt.Println(response,"循环里面的观众结构",viewer)
                            viewer[0][request.Flag].WriteJSON(response)
                        }
                        fmt.Println("查看每个链接的效果",v,"返回的数据")
                    }
                } */
            case "join" : //有人加入查看用户上报回家位置map[int][]map[string][]map[int]*websocket.Conn
                if _,ok := c.hub.rooms[request.RoomId];ok {
                    /* link[request.Flag] = c.conn
                    linkArr["other"] = append(linkArr["other"],link)
                    c.hub.rooms[request.RoomId] = append(c.hub.rooms[request.RoomId],linkArr) */
                    viewers := c.hub.rooms[request.RoomId].viewer
                    viewer := make(map[int]*websocket.Conn)
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
                /* if _,ok := c.hub.rooms[request.RoomId];ok {
                    //检查回家人是否还在保持连接状态
                    if master,ok := c.hub.rooms[request.RoomId][0]["master"];ok {
                        if len(master) < 1 {
                            response := responseData { 
                                Event : "error", 
                                Msg : "糟糕,你的小伙伴失联了,要不要电话联系一下?", 
                            } 
                            c.conn.WriteJSON(response)
                        }else{
                            response := responseData { 
                                Event : "getLocation", 
                            } 
                            master[0][request.Flag].WriteJSON(response)
                        }
                    }else{
                        response := responseData { 
                            Event : "error", 
                            Msg : "咦?是不是来错地方了?", 
                        } 
                        c.conn.WriteJSON(response)
                    }
                } */
            case "leave": //有人退出查看用户实时推送位置map[int][] map[string][]map[int]*websocket.Conn
                /* if viewer,ok := c.hub.rooms[request.RoomId]; ok {
                    fmt.Println("查看一下当前房间的连接数",viewer)
                    for k,v := range viewer {
                        if other,ok := v["other"];ok {
                            if other[0][request.Flag] == c.conn {
                                delete(viewer[k],v["other"])
                            }
                        }
                        fmt.Println("查看每个链接的效果",v,"返回的数据",k,"...",viewer[k])
                    }
                } */
            case "end": //用户主动结束上报回家事件
        }
        // c.hub.broadcast <- request
        fmt.Println("查看一下现在有多少个房间",c.hub.rooms,"看看看现在的主播",c.hub.rooms[request.RoomId])
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