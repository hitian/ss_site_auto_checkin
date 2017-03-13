package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	configFilePath = ".ss.site.checkin.account.json"
	cookieFilePath = ".ss.site.checkin.cookie"
	maxTryTimes    = 3
)

type Result int

const (
	resultOk Result = 1 << iota
	resultNeedLogin
	resultError
)

var siteInfo SiteInfo
var cookie string

func main() {
	siteInfo = readAccountInfo()
	//fmt.Print(siteInfo)
	cookie = readCookie()

	if len(cookie) < 1 {
		newCookie := doLogin()
		if len(newCookie) < 1 {
			fmt.Println("login failed. cookie not right.")
			os.Exit(1)
		}
		saveCookieToFile(newCookie)
		cookie = newCookie
	}

	currentRetryTimes := 0
	for currentRetryTimes < maxTryTimes {
		res := doCheckin()
		if res == resultError {
			os.Exit(1)
		}
		if res == resultNeedLogin {
			cookie = doLogin()
			fmt.Printf("doLogin result: %s", cookie)
			currentRetryTimes++
		}
		if res == resultOk {
			saveCookieToFile(cookie)
			os.Exit(0)
		}
	}
}

type CheckinResponse struct {
	Msg string `json:"msg"`
}

func doCheckin() Result {
	now := time.Now()
	fmt.Printf("doCheckin @ %s \n", now.Format("2006-01-02 15:04:05"))
	var url = siteInfo.SiteURL + "user/checkin"
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("Cookie", cookie)
	client := &http.Client{
		CheckRedirect: doNotRedirect,
	}
	//fmt.Printf("%v", req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print(err)
		return resultError
	}
	defer resp.Body.Close()
	if resp.StatusCode == 302 {
		fmt.Println("need login <<")
		return resultNeedLogin
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err)
		return resultError
	}
	var result CheckinResponse
	json.Unmarshal(body, &result)
	fmt.Printf("签到成功 %s\n", result.Msg)
	return resultOk
}

type SiteInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	SiteURL  string `json:"site_url"`
}

type LoginResponse struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

func doLogin() string {
	fmt.Print("start login \n")
	//fmt.Printf("account: %#v", siteInfo)
	url := siteInfo.SiteURL + "auth/login"
	loginInfo := map[string]string{"email": siteInfo.Username, "passwd": siteInfo.Password, "remember_me": "week"}
	jsonValue, _ := json.Marshal(loginInfo)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, */*")
	client := &http.Client{
		CheckRedirect: doNotRedirect,
	}
	//fmt.Printf("%v", req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("do login error: %v", err)
		os.Exit(1)
	}

	var result LoginResponse
	json.Unmarshal(body, &result)
	if result.Ret == 0 {
		fmt.Printf("Login failed: %s \n", result.Msg)
		os.Exit(1)
	}
	cookies := resp.Cookies()
	//fmt.Printf("cookies: %v", cookies)
	var cookieStringArr string
	for _, cookie := range cookies {
		cookieStringArr += fmt.Sprintf("%s=%s;", cookie.Name, cookie.Value)
	}
	return cookieStringArr
}

func doNotRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func readCookie() string {
	filePath := getCookieFilePath()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("cookie file not exist")
		return ""
	}
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Print(err.Error())
		return ""
	}
	//fmt.Println(file)
	return string(file)
}

func saveCookieToFile(cookie string) {
	filePath := getCookieFilePath()
	err := ioutil.WriteFile(filePath, []byte(cookie), 0600)
	if err != nil {
		fmt.Println("save cookie to file failed")
	}
}

func getCookieFilePath() string {
	homeDir := os.Getenv("HOME")
	filePath := homeDir + "/" + cookieFilePath
	return filePath
}

func readAccountInfo() SiteInfo {
	homeDir := os.Getenv("HOME")
	filePath := homeDir + "/" + configFilePath
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("account file not exists, please create %s in your home dir\n", configFilePath)
		fmt.Print("Content like: {\"username\": \"email\", \"password\":\"password\", \"site_url\": \"https://www.example.com/\"}\n")
		os.Exit(1)
	}
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Print("read account info file error", err.Error())
		os.Exit(1)
	}
	var account SiteInfo
	json.Unmarshal(file, &account)
	return account
}
