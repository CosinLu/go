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

func testhandler(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("错误",err)
        log.Fatal(err)
    }
    defer ws.Close()
    fmt.Println("进来了")
    for {
        t, p, err := ws.ReadMessage()
        if err != nil {
            ws.Close()
            break
        }
        fmt.Println("甚至都打印到数据了",t,"数据",string(p),"错误",err)
        ws.WriteJSON(string(p))
        //manager.broadcast <- jsonMessage
    }
}

func res(w http.ResponseWriter,r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("错误",err)
        log.Fatal(err)
    }
    defer ws.Close()
    fmt.Println("准备返回数据进来了")
    err = ws.WriteJSON("可以的")
    log.Println(err)
}

var a chan string
func ready(s string) {
    fmt.Println("现在的channel为",s)
    a <- s
}

func main(){
    a = make(chan string)
    go ready("cosin")
    i := <- a
    fmt.Println("信道",i)
	redis_example()
    defer func(){
        if err := recover();err != nil{
            fmt.Println("panic is success",err)
        }
    }()
    mux := http.NewServeMux()
    mux.HandleFunc("/test",testhandler)
    mux.HandleFunc("/res",res)
    log.Println("Listening...")
    http.ListenAndServe(":8800",mux)
}

//函数作为返回值
func test() func(int,int) int{
    return func(x,y int) int {
        return x + y
    }
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
