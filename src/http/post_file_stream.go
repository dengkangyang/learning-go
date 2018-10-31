package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

func main() {
	url := "http://localhost:8999/tech-sdkwrapper/timevale/sign/userStreamSign"
	// url := "http://127.0.0.1:5011"

	params := make(map[string]string)
	params["signType"] = "Single"
	params["accountId"] = "CF71BD99B7F243148008F1F92D63A663"
	params["sealData"] = `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`
	params["signPos"] = `{
		"posType": 0,
		"posPage": "1",
		"posX": 500,
		"posY": 100
	}`
	params["dstPdfFile"] = "/Users/pathbox/edst_file/0001_ok_done.pdf"

	client := NewHTTPClient()
	file, _ := os.Open("/Users/pathbox/edst_file/0001_done.pdf")
	defer file.Close()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)

	fw, err := mw.CreateFormFile("file", "0001_done.pdf")
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(fw, file)

	for key, val := range params {
		_ = mw.WriteField(key, val)
	}
	err = mw.Close() // 要在 http.NewRequest client.Do 之前就Close，不能 defer mw.Close() 这样会导致产生不合法协议的multipart/form-data 请求
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	fmt.Println("After request: ", resp.StatusCode)
	fmt.Println(string(rb))

}

func NewHTTPClient() *http.Client {
	// transport := &http.Transport{
	// 	Dial: (&net.Dialer{
	// 		Timeout: 30 * time.Second,
	// 	}).Dial,
	// 	TLSHandshakeTimeout:   15 * time.Second,
	// 	ResponseHeaderTimeout: 30 * time.Second,
	// }

	// client := &http.Client{
	// 	Timeout:   30 * time.Second,
	// 	Transport: transport,
	// }
	client := &http.Client{}
	return client
}
