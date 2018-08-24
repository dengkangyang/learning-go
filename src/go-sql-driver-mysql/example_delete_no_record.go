package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "ucloud:ucloud.cn@tcp(192.168.154.15:3306)/uaccount?charset=utf8")

	if err != nil {
		fmt.Println(err)
		return
	}

	sql := "DELETE FROM t_totp_key WHERE user_id=1;"

	result, err := db.Exec(sql)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("YES")
	}

	n, _ := result.RowsAffected() // 通过这里判断是否真的删除了记录，返回0表示没有满足条件的记录被删除
	fmt.Println(n)
}
