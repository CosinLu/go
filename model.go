package main

import (
	"fmt"
	"database/sql"
	_"github.com/go-sql-driver/mysql"
	"time"
)

//添加最后一次的位置
func SetLastlocation(id int,lastLocation string) {
	db,err := sql.Open("mysql",DBHOST)
	if err != nil {
		panic("链接错误")
	}
	defer db.Close()
	stmt,err := db.Prepare("UPDATE nz_gohome SET last_location = ? WHERE id = ?")
	if err != nil {
		panic("预定义error")
	}
	res,err := stmt.Exec(lastLocation,id)
	if err != nil {
		panic("绑定error")
	}
	affect,_ := res.RowsAffected()
	fmt.Println("修改结果",affect)
}

//根据用户id获取用户名
func GetNickById(uid int) (nick string,phone string) {
	db,err := sql.Open("mysql",DBHOST)
	if err != nil {
		panic("数据错误")
	}
	defer db.Close()
	err = db.QueryRow("SELECT u_nick,u_phonenum FROM nz_user WHERE u_id = ?",uid).Scan(&nick,&phone)
	if err != nil {
		nick = "xxx"
	}
	return nick,phone
}

//结束回家进程
func EndGohomeCourse(id int){
	db,err := sql.Open("mysql",DBHOST)
	if err != nil {
		panic("数据错误")
	}
	defer db.Close()
	endTime := time.Now().Unix()
	stmt,err := db.Prepare("UPDATE nz_gohome SET status = 1,end_time = ? WHERE id = ?")
	if err != nil {
		panic("预定义error")
	}
	res,err := stmt.Exec(endTime,id)
	if err != nil {
		panic("绑定error")
	}
	affect,_ := res.RowsAffected()
	fmt.Println("修改结果",affect)
}