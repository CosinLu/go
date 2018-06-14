package main

import (
	"fmt"
	"database/sql"
	_"github.com/go-sql-driver/mysql"
	"time"
	_"net/http"
	_"io/ioutil"
	"net/url"
)

//添加最后一次的位置
func SetLastlocation(id int,lastLocation string) {
	db,err := sql.Open("mysql",DBHOST)
	if err != nil {
		fmt.Println("SetLastlocation链接错误",err)
	}
	defer db.Close()
	stmt,err := db.Prepare("UPDATE nz_gohome SET last_location = ? WHERE id = ?")
	if err != nil {
		fmt.Println("SetLastlocation预定义error",err)
	}
	res,err := stmt.Exec(lastLocation,id)
	if err != nil {
		fmt.Println("SetLastlocation绑定error",err)
	}
	affect,_ := res.RowsAffected()
	fmt.Println("修改结果",affect)
}

//根据用户id获取用户名
func GetNickById(uid int) (nick string,phone string) {
	db,err := sql.Open("mysql",DBHOST)
	if err != nil {
		fmt.Println("GetNickById链接错误",err)
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
		fmt.Println("EndGohomeCourse链接错误",err)
	}
	defer db.Close()
	endTime := time.Now().Unix()
	stmt,err := db.Prepare("UPDATE nz_gohome SET status = 1,end_time = ? WHERE id = ?")
	if err != nil {
		fmt.Println("EndGohomeCourse预定义error",err)
	}
	res,err := stmt.Exec(endTime,id)
	if err != nil {
		fmt.Println("EndGohomeCourse绑定error",err)
	}
	affect,_ := res.RowsAffected()
	fmt.Println("修改结果",affect)
	//给用户发送短信
	var gcid,uid int
	err = db.QueryRow("SELECT gcid,uid FROM nz_gohome WHERE id = ?",id).Scan(&gcid,&uid)
	if err != nil {
		fmt.Println("EndGohomeCourse查询数据error",err)
	}
	//获取联系人手机号
	var contactsPhone string
	err = db.QueryRow("SELECT phone FROM nz_gohome_contacts WHERE id = ?",gcid).Scan(&contactsPhone)
	if err != nil {
		fmt.Println("EndGohomeCourse查询联系人手机号数据error",err)
	}
	//获取用户的昵称和手机号
	var nick,mobile string
	err = db.QueryRow("SELECT u_nick,u_phonenum FROM nz_user WHERE u_id = ?",uid).Scan(&nick,&mobile)
	//发送短信通知
	mobile = string([]byte(mobile)[7:])
	msg := "#name#="+nick+"&#num#="+mobile
	msg = url.QueryEscape(msg)
	requestUrl := "http://v.juhe.cn/sms/send?mobile="+contactsPhone+"&tpl_id=82533&tpl_value="+msg+"&key=c86578753fa4f1ed8968ca9584b3cefb"
	fmt.Println("请求地址",requestUrl)
	/* resp,err := http.Get(requestUrl)
	if err != nil {
		fmt.Println("短信发送err",err)
	}
	defer resp.Body.Close()
	body,_ := ioutil.ReadAll(resp.Body)
	fmt.Println("短信发送结果",string(body)) */
}