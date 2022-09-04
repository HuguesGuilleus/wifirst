package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
)

type SessionsCode struct {
	Radius struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	} `json:"radius"`
}

func main() {
	l := flag.String("l", "", "The login")
	p := flag.String("p", "", "The password")
	flag.Parse()
	if *l == "" || *p == "" {
		flag.Usage()
		return
	}

	boxToken := getBoxToken()
	sessionCode := getSessionCode(boxToken, *l, *p)
	sendToken(sessionCode)
}

func getBoxToken() string {
	const url = "https://wireless.wifirst.net:8099/index.txt"
	log.Println("GET", url)
	boxTokenResponse, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer boxTokenResponse.Body.Close()

	boxToken, err := io.ReadAll(boxTokenResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(boxToken)
}

func getSessionCode(boxToken, login, password string) (sessionsCode SessionsCode) {
	data, _ := json.Marshal(map[string]string{
		"email":       login,
		"password":    password,
		"box_token":   string(boxToken),
		"fragment_id": "12514",
	})

	sessionsResponse := post(
		"https://portal-front.wifirst.net/api/sessions",
		"application/json",
		data,
	)
	defer sessionsResponse.Body.Close()

	err := json.NewDecoder(sessionsResponse.Body).Decode(&sessionsCode)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func sendToken(sessionsCode SessionsCode) {
	loginResponse := post(
		"https://wireless.wifirst.net:8090/goform/HtmlLoginRequest",
		"application/x-www-form-urlencoded",
		[]byte("username="+sessionsCode.Radius.Login+
			"&password="+sessionsCode.Radius.Password+
			"&success_url=https%3A%2F%2Fwww.google.fr%2F&error_url=https%3A%2F%2Fportal-front.wifirst.net%2Fconnect-error&update_session=0"),
	)

	defer loginResponse.Body.Close()
	io.Copy(io.Discard, loginResponse.Body)
}

// Make a POST HTTP request
func post(url, contentType string, data []byte) *http.Response {
	log.Println("POST", url)
	response, err := http.Post(url, contentType, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	return response
}
