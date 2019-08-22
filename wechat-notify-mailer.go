package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	netUrl "net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	accessKeyId      string
	accessKeySecret  string
	fromEmailAddress string
	fromEmailAlias   string
)

// github.com/caiguanhao/wechat-notify
type Input struct {
	Timestamp   int64
	Service     string
	Event       string
	Action      string
	Host        string
	Description string
	URL         string
}

const tpl = `<b>Host:</b><br>{{.Host}}
<br><br>
<b>Description:</b><br>{{.Description}}
<br><br>
<b>Action:</b><br>{{.Action}}
<br><br>
<b>URL:</b><br>{{.URL}}
<br><br>
<b>Time:</b><br>{{.Timestamp | format}}`

func (input Input) String() string {
	t := template.Must(template.New("content").Funcs(template.FuncMap{
		"format": func(sec int64) string {
			return time.Unix(sec, 0).Format("2006-01-02 15:04:05")
		},
	}).Parse(tpl))
	var b bytes.Buffer
	err := t.Execute(&b, input)
	if err == nil {
		return b.String()
	}
	return err.Error()
}

// github.com/caiguanhao/wechat-notify
func parse(input []byte) *Input {
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimSpace(input)))
	var ret Input
	isDesc := false
	for scanner.Scan() {
		line := scanner.Text()
		if !isDesc && len(line) == 0 {
			isDesc = true
			continue
		}
		if isDesc {
			ret.Description = ret.Description + line + "\n"
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			ret.Description = ret.Description + line + "\n"
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "timestamp":
			ret.Timestamp, _ = strconv.ParseInt(value, 10, 64)
		case "service":
			ret.Service = value
		case "event":
			ret.Event = value
		case "action":
			ret.Action = value
		case "host":
			ret.Host = value
		case "url":
			ret.URL = value
		}
		ret.Description = ""
	}
	ret.Description = strings.TrimSpace(ret.Description)
	return &ret
}

func randomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func urlEncode(input string) string {
	return netUrl.QueryEscape(strings.Replace(strings.Replace(strings.Replace(input, "+", "%20", -1), "*", "%2A", -1), "%7E", "~", -1))
}

func sendMail(emailSubject, content, toEmailAddress string) error {
	v := netUrl.Values{}
	v.Set("Format", "json")
	v.Set("Version", "2015-11-23")
	v.Set("AccessKeyId", accessKeyId)
	v.Set("SignatureMethod", "HMAC-SHA1")
	v.Set("Timestamp", time.Now().UTC().Format(time.RFC3339))
	v.Set("SignatureVersion", "1.0")
	v.Set("SignatureNonce", randomString(64))
	v.Set("Action", "SingleSendMail")
	v.Set("AccountName", fromEmailAddress)
	v.Set("ReplyToAddress", "false")
	v.Set("AddressType", "0")
	v.Set("FromAlias", fromEmailAlias)
	v.Set("Subject", emailSubject)
	v.Set("HtmlBody", content)
	v.Set("ToAddress", toEmailAddress)

	h := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	h.Write([]byte("POST&%2F&" + urlEncode(v.Encode())))
	v.Set("Signature", base64.StdEncoding.EncodeToString(h.Sum(nil)))

	req, err := http.NewRequest("POST", "https://dm.aliyuncs.com/", strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := http.Client{
		Timeout: time.Duration(3 * time.Second),
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}
	return nil
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Please provide at least one Email address.")
		os.Exit(1)
	}
	stdin, stdinErr := ioutil.ReadAll(os.Stdin)
	if stdinErr != nil {
		fmt.Fprintln(os.Stderr, stdinErr)
		os.Exit(1)
	}
	input := parse(stdin)

	hasError := false
	for _, arg := range flag.Args() {
		var err = sendMail("New Error: "+input.Host, input.String(), arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			hasError = true
		}
	}

	if hasError {
		os.Exit(1)
	}
}
