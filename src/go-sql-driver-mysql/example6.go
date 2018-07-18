package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/gommon/log"
)

func main() {
	db, err := sql.Open("mysql", "ucloud:ucloud.cn@tcp(192.168.154.15:3306)/uaccount")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		fmt.Println("Ping err: ", err)
	}

	idAry := []string{"yeheng003", "yeheng001", "wddg1k"}
	ids := strings.Join(idAry, "','")
	sqlRaw := fmt.Sprintf(`SELECT id, resource_id, resource_type FROM t_resource WHERE resource_id IN ('%s') OR id IN ('%s')`, ids, ids)
	rows, err := db.Query(sqlRaw)

	// log.Print("cols len: ", len(cols))

	if err != nil {
		log.Errorf("SQL t_resource error:%s", err)
	} else {
		fmt.Println("here")
		for rows.Next() {
			cols, _ := rows.Columns()
			buff := make([]interface{}, len(cols)) // 临时slice
			vals := make([]string, len(cols))      // 存数据slice
			for i, _ := range buff {
				buff[i] = &vals[i]
			}
			err = rows.Scan(buff...)
			if err != nil {
				log.Errorf("collect rows.Scan error:%s", err)
			}
			fmt.Printf("Vals:%v\n", vals)

			id := vals[0]
			resourceID := vals[1]
			resourceType := vals[2]

			fmt.Printf("id:%s, resourceID:%s, resourceType:%s", id, resourceID, resourceType)
		}
	}

}
