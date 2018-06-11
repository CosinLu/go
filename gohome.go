package main

import (
	"fmt"
    "github.com/gorilla/websocket"
    "time"
    "log"
)

func (c *Client) GohomeHub() {
    defer func() {
        fmt.Println("要断了是吗")
        if err := recover();err != nil {
            response := ResponseData { 
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
        var request RequestData
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
				if request.Token != "" {
					users,e := DecryptToken(request.Token)
					if !e {
						user := *users
						fmt.Println("用户数据",user,user.Id)
						if _,ok := c.hub.rooms[request.RoomId];!ok {
							roomMaster := RoomNumber{master:c.conn}
							c.hub.rooms[request.RoomId] = roomMaster
							response := ResponseData { 
								Event : "success",
								Msg : "创建成功" , 
							}
							c.conn.WriteJSON(response)
						}else {
							//如果连接存在的话,检查一下是不是主播断掉然后重连的
							if c.hub.rooms[request.RoomId].master == nil {
								viewers := c.hub.rooms[request.RoomId].viewer
								roomMaster := RoomNumber{master:c.conn,viewer:viewers}
								c.hub.rooms[request.RoomId] = roomMaster
								response := ResponseData { 
									Event : "success",
									Msg : "创建成功" , 
								}
								c.conn.WriteJSON(response)
							}else{
								response := ResponseData { 
									Event : "error",
									Msg : "链接已存在" , 
								}
								c.conn.WriteJSON(response)
							}
						}
					}else{
						c.hub.unregister <- c 
						c.conn.Close()
						response := ResponseData { 
							Event : "needLogin",
							Msg : "请登录" , 
						}
						c.conn.WriteJSON(response)
					}
				}else{
					c.hub.unregister <- c 
					c.conn.Close()
					response := ResponseData { 
						Event : "needLogin",
						Msg : "请登录" , 
					}
					c.conn.WriteJSON(response)
				}
            case "location": //用户持续上报位置
                if rooms,ok := c.hub.rooms[request.RoomId];ok {
                    fmt.Println("查看一下当前房间的连接数",rooms)
                    for _,viewer := range rooms.viewer {
                        response := ResponseData { 
                            Event : "location",
                            Lat : request.Lat , 
                            Lon : request.Lon, 
                        }
                        viewer.WriteJSON(response)
                        fmt.Println("查看每个链接的效果",viewer,"返回的数据")
                    }
                    response := ResponseData { 
                        Event : "success",
                        Msg : "位置推送成功" , 
                    }
                    c.conn.WriteJSON(response)
                }else{
                    response := ResponseData { 
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
                    c.hub.rooms[request.RoomId] = RoomNumber{master:master,viewer:viewers}
                    response := ResponseData { 
                        Event : "success",
                        Msg : "链接加入成功" , 
                    }
                    c.conn.WriteJSON(response)
                }else{
                    response := ResponseData { 
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
                        response := ResponseData { 
                            Event : "getLocation", 
                        } 
                        master.WriteJSON(response)
                        viewerResponse := ResponseData { 
                            Event : "success", 
                            Msg : "发送成功", 
                        } 
                        c.conn.WriteJSON(viewerResponse)
                    }else{
                        response := ResponseData { 
                            Event : "error", 
                            Msg : "咦?是不是来错地方了?", 
                        } 
                        c.conn.WriteJSON(response)
                    }
                }else{
                    response := ResponseData { 
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
                            response := ResponseData { 
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