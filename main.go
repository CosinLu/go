package main

import (
	"fmt"
    "github.com/garyburd/redigo/redis"//redis
    "database/sql"
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

func UpgradeSocket(w http.ResponseWriter, r *http.Request) {

    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("升级socket",ws)
}

func main(){
    defer func(){
        if err := recover();err != nil{
            fmt.Println("panic is success")
        }
    }()
	redis_example()
    //函数作为返回值
    add := test()(1,3)
    fmt.Println("现在的值是",add)
    func (s string) {
        fmt.Println("现在的值是",s)
    }("hello world")
}

//函数作为返回值
func test() func(int,int) int{
    return func(x,y int) int {
        return x + y
    }
}

type UserRes struct{
    nick string
    phone string
}

func redis_example(){
    c, err := redis.Dial("tcp", "127.0.0.1:6379")
    checkErr(err)
    defer c.Close()
    /* _, err = c.Do("SET", "mykey", "superWang")
    if err != nil {
        fmt.Println("redis set failed:", err)
    } */
    c.Do("auth","cosin")
    username, err := redis.String(c.Do("GET", "author"))
    if err != nil {
        fmt.Println("redis get failed:", err)
    } else {
        fmt.Printf("Get mykey: %v \n", username)
    }

    db,err := sql.Open("mysql","root:@(127.0.0.1:3306)/test?charset=utf8")
    if err != nil{
        fmt.Printf("databases connected error:%v\n",err)
    }
    defer db.Close()
    //插入数据
    /* stmt,err := db.Prepare("INSERT user SET nick = ?,phone = ?")
    if err != nil{
        fmt.Printf("insert data error:%v\n",err)
    }   
    res,err := stmt.Exec("cosin","13730444340")
    if err != nil {
        fmt.Printf("bind data error:%v\n",err)
    }else{
        id,err := res.LastInsertId()
        checkErr(err)
        fmt.Printf("insert data id is %d\n",id)
    } */
    users := UserRes{}
    //查询数据
    /* stmt,err := db.Prepare("SELECT nick,phone FROM user WHERE id = ?")
    checkErr(err)
    fmt.Print(res)
    res,err := stmt.Query(1)
    checkErr(err)
    for res.Next(){
        err = res.Scan(&users.nick,&users.phone)
        checkErr(err)
        fmt.Printf("nick is %s and phone is %s\n",users.nick,users.phone)
    } */
    err = db.QueryRow("SELECT nick,phone FROM user where id = ?",1).Scan(&users.nick,&users.phone)
    checkErr(err)
    fmt.Printf("nick is %s and phone is %s\n",users.nick,users.phone)
}

func checkErr(err error){
    if err != nil {
        panic(err)
    }
}
