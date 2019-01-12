package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	sendcloud "github.com/smartwalle/sendcloud"
	"github.com/tealeg/xlsx"
)

const (
	imagePath1    = "./image.jpg"
	imagePath2    = ""
	subjectPath   = "./subject.txt"
	bodyPath      = "./body.txt"
	excelFileName = "./email_list.xlsx"
)

type SendMail struct {
	user     string
	password string
	host     string
	port     string
	auth     smtp.Auth
}

type Attachment struct {
	name        []string
	contentType string
	withFile    bool
}

type Message struct {
	from        string
	to          string
	cc          []string
	bcc         []string
	subject     string
	body        string
	contentType string
	attachment  Attachment
}

func main() {
	from := "postmaster@fintechgood.com"
	// subs := map[string]interface{}{}

	apiUser := "fintech_send"
	apiKey := "apikey"
	sendcloud.UpdateApiInfo(apiUser, apiKey)
	tplName := "fintech_sendcloud_template"

	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		panic(err)
	}
	for index, sheet := range xlFile.Sheets {
		if index == 0 {
			for rowNum, row := range sheet.Rows {
				for index, cell := range row.Cells {
					text := cell.String()
					if index == 0 {
						fmt.Println(rowNum+1, text)
						toEmail := text
						if strings.TrimSpace(toEmail) != "" {
							var to = make([]map[string]string, 1)
							to[0] = map[string]string{"to": toEmail, "%url%": ""}
							_, err, result := sendcloud.SendTemplateMail(tplName, from, from, from, "", to, nil)
							if err != nil {
								fmt.Printf("Send Error: %s-result:%s\n", err, result)
							}
							fmt.Printf("=Send Success=result:%s", result)
						}
					}
				}
			}
		}
	}
}

func (mail SendMail) SendWithSendCloud(from, to, subject string, buf []byte) error {
	var r http.Request
	r.ParseForm()
	client := NewHTTPClient()
	url := "http://api.sendcloud.net/apiv2/mail/send"

	fmt.Println("len:", len(buf))

	paramMap := make(map[string]string)
	paramMap["apiUser"] = " "
	paramMap["apiKey"] = " "
	paramMap["from"] = from
	paramMap["fromName"] = from
	paramMap["to"] = to
	paramMap["subject"] = subject
	paramMap["html"] = string(buf)
	paramMap["respEmailId"] = "true"

	for k, v := range paramMap {
		r.Form.Add(k, v)
	}

	payload := strings.TrimSpace(r.Form.Encode())
	request, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("Response:%s\n", body)

	return nil
}

func NewHTTPClient() *http.Client {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
	return client
}
