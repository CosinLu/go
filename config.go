package main

import (
	"github.com/gorilla/websocket"
)

//解析数据请求
type RequestData struct {
    Event string `json:"event"`
    RoomId int `json:"roomid"`
    Uid int `json:"uid"`
    Lon string `json:"lon"`
    Lat string `json:"lat"`
	Flag int `json:"flag"`
	Token string `json:"token"`
	Gcid int `json:"gcid"`
	Gaid int `json:"gaid"`
	Gwid int `json:"gwid"`
	NormalTime string `json:"normalTime"`
	LastTime string `json:"lastTime"`
	StartLocation string `json:"startLocation"`
}

//返回数据格式
type ResponseData struct {
    Event string `json:"event"`
    RoomId int `json:"roomid"`
    Uid int `json:"uid"`
    Lon string `json:"lon"`
    Lat string `json:"lat"`
    Msg string `json:"msg"`
    Error interface{} `json:"error"`
}

//回家人,监视人结构体
type RoomNumber struct {
	master *websocket.Conn
	viewer map[int]*websocket.Conn
}

type Client struct {
    hub *Hub
    conn *websocket.Conn
}

type Hub struct {
    clients map[*Client]bool
    register chan *Client
    unregister chan *Client
    abnormal chan *Client
    rooms map[int]RoomNumber
}


const REDISHOST = "127.0.0.1:6379"
const DBHOST = "users:jiayukeji2018@tcp(139.129.217.219:3306)/nvchezhu"