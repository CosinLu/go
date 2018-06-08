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
	viewer map[int]*websocket.Conn
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
    register chan *Client
    unregister chan *Client
    abnormal chan *Client
    rooms map[int]roomNumber
}

type Client struct {
    hub *Hub
    conn *websocket.Conn
}

var manager = Hub{
    register: make(chan *Client),
    unregister: make(chan *Client),
    abnormal : make(chan *Client),
    clients: make(map[*Client]bool),
    rooms: make(map[int]roomNumber),
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
                        client.hub.rooms[roomsKey] = roomNumber{viewer:viewers}
                        for _,viewer := range roomsValue.viewer {
                            response := responseData { 
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
    go client.readPump()
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
        c.hub.abnormal <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
    for {
        var request requestData
        err := c.conn.ReadJSON(&request)
        if err != nil {
            fmt.Println("readmsgerror",err)
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("readerror: %v", err)
            }
            break
        }
        switch request.Event {
            case "create":  //创建一个回家的连接
                if _,ok := c.hub.rooms[request.RoomId];!ok {
                    roomMaster := roomNumber{master:c.conn}
                    c.hub.rooms[request.RoomId] = roomMaster
                    response := responseData { 
                        Event : "success",
                        Msg : "创建成功" , 
                    }
                    c.conn.WriteJSON(response)
                }else {
                    //如果连接存在的话,检查一下是不是主播断掉然后重连的
                    if c.hub.rooms[request.RoomId].master == nil {
                        viewers := c.hub.rooms[request.RoomId].viewer
                        roomMaster := roomNumber{master:c.conn,viewer:viewers}
                        c.hub.rooms[request.RoomId] = roomMaster
                        response := responseData { 
                            Event : "success",
                            Msg : "创建成功" , 
                        }
                        c.conn.WriteJSON(response)
                    }else{
                        response := responseData { 
                            Event : "error",
                            Msg : "链接已存在" , 
                        }
                        c.conn.WriteJSON(response)
                    }
                }
            case "location": //用户持续上报位置
                if rooms,ok := c.hub.rooms[request.RoomId];ok {
                    fmt.Println("查看一下当前房间的连接数",rooms)
                    for _,viewer := range rooms.viewer {
                        response := responseData { 
                            Event : "location",
                            Lat : request.Lat , 
                            Lon : request.Lon, 
                        }
                        viewer.WriteJSON(response)
                        fmt.Println("查看每个链接的效果",viewer,"返回的数据")
                    }
                    response := responseData { 
                        Event : "success",
                        Msg : "位置推送成功" , 
                    }
                    c.conn.WriteJSON(response)
                }else{
                    response := responseData { 
                        Event : "error",
                        Msg : "链接不存在,请重新设置回家" , 
                    }
                    c.conn.WriteJSON(response)
                }
            case "join" : //有人加入查看用户上报回家位置
                if _,ok := c.hub.rooms[request.RoomId];ok {
                    viewers := make(map[int]*websocket.Conn)
                    if len(c.hub.rooms[request.RoomId].viewer) == 0 {
                        viewers[request.Flag] = c.conn
                    }else{
                        viewers = c.hub.rooms[request.RoomId].viewer
                        viewers[request.Flag] = c.conn
                    }
                    master := c.hub.rooms[request.RoomId].master
                    c.hub.rooms[request.RoomId] = roomNumber{master:master,viewer:viewers}
                    response := responseData { 
                        Event : "success",
                        Msg : "链接加入成功" , 
                    }
                    c.conn.WriteJSON(response)
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
                }else{
                    response := responseData { 
                        Event : "error", 
                        Msg : "暂无回家人可联系", 
                    } 
                    c.conn.WriteJSON(response)
                }
            case "leave": //有人退出查看用户实时推送位置
                if rooms,ok := c.hub.rooms[request.RoomId]; ok {
                    for k,viewer := range rooms.viewer {
                        if viewer == c.conn {
                            delete(rooms.viewer,k)
                            c.hub.unregister <- c 
                            c.conn.Close()
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
                            response := responseData { 
                                Event : "end",
                            }
                            viewer.WriteJSON(response)
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