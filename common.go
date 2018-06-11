package main

import(
	"encoding/base64"
	"database/sql"
	_"github.com/go-sql-driver/mysql"
	"reflect"
	"fmt"
	"github.com/garyburd/redigo/redis"
)

const (
    base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
)

var coder = base64.NewEncoding(base64Table)

func DecryptToken(token string) (users *Users,e bool){
	st, _ := coder.DecodeString(token)
	decode := string(st)
	start := len(decode) - 11
	rs := []rune(decode)
	phone := string(rs[start:])
    db, err := sql.Open("mysql", DBHOST)
    if err != nil{
		fmt.Println("数据库链接",err,phone)
        return users,true
    }
    defer db.Close()
	users = new(Users)
	err = db.QueryRow("SELECT u_nick,u_id,u_phonenum,u_vip FROM nz_user WHERE u_phonenum = ? AND u_sta <> 1", phone).Scan(&users.Nick,&users.Id,&users.Phone,&users.Vip)
	fmt.Println("查看结果",err,users,phone)
    if err != nil {
        return users,true
	}
	return users,false
}

type Users struct{
	Id int
	Nick string
	Phone string
	Vip int 
}


//结构体转map
func Struct2Map(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}


//获取缓存
func GetCache(key string) (string, bool) {
	c, err := redis.Dial("tcp",REDISHOST)
	if err != nil {
		return "",false
	}
    defer c.Close()
	c.Do("AUTH", "jiayukeji")
	data, err := redis.String(c.Do("GET", key))
    if err != nil {
		return "",false
    } else {
        return data,true
	}
}

//设置缓存 
func SetCache(key string,value interface{},expire int64) bool {
	c, err := redis.Dial("tcp",REDISHOST)
	if err != nil {
		return false
	}
	defer c.Close()
	c.Do("AUTH", "jiayukeji")
	if expire == 0 {
		expire = 7200
	}
    _, err = c.Do("SET", key, value,"EX",expire)
    if err != nil {
		return false
	}
	return true
}
