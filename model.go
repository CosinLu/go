package main

import (
	"fmt"
	"database/sql"
	_"github.com/go-sql-driver/mysql"
	"time"
)

//添加回家数据
func AddGohomeData(gohomeData *RequestData) bool {
	db,err := sql.Open("mysql",DBHOST)
	if err != nil {
		panic("数据错误")
	}
	defer db.Close()
	var id int
	//先检查有没有进行中的进程
	err = db.QueryRow("SELECT id FROM nz_gohome WHERE uid = ? AND status = 0 ORDER BY id desc LIMIT 1",gohomeData.Uid).Scan(&id)
	if err != nil {//不是空就说明没查到数据返回了一些错误信息
		createTime := time.Now().Unix()
		stmt,_ := db.Prepare("INSERT INTO nz_gohome (uid,gcid,gaid,gwid,normal_time,last_time,start_localtion,create_time) VALUES (?,?,?,?,?,?,?,?)")
		res,err := stmt.Exec(gohomeData.Uid,gohomeData.Gcid,gohomeData.Gaid,gohomeData.Gwid,gohomeData.NormalTime,gohomeData.LastTime,gohomeData.StartLocation,createTime)
		if err != nil {
			panic("添加错误")
		}
		lastId,_ := res.LastInsertId()
		fmt.Println("添加数据",lastId)
	}
	return false
}