package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	flagLogin := flag.String("l", "", "The login")
	flagPassword := flag.String("p", "", "The password")
	flag.Parse()

	if *flagLogin == "" || *flagPassword == "" {
		flag.Usage()
		os.Exit(1)
	}

	portal := getPortal()
	boxToken := getBoxToken(portal)
	login, password := getSessionCode(portal, boxToken, *flagLogin, *flagPassword)
	sendToken(portal, login, password)
}

func getPortal() *url.URL {
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("One request") }}
	response, err := client.Get("http://detectportal.firefox.com/success.txt")
	if response == nil {
		log.Fatal(err)
	}
	switch portal, err := response.Location(); err {
	case nil:
		return portal
	case http.ErrNoLocation:
		log.Fatal("No location are present in the response header, you should be already connected.")
	default:
		log.Fatal("Error when get location response header:", err)
	}
	return nil // never executed because the previous log.Fatal call.
}

func getBoxToken(portal *url.URL) string {
	boxTokenResponse, err := http.Get(portal.String())
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

func getSessionCode(portal *url.URL, boxToken, email, inputPassword string) (login, password string) {
	data, _ := json.Marshal(map[string]string{
		"email":       email,
		"password":    inputPassword,
		"box_token":   boxToken,
		"fragment_id": "12514",
	})

	sessionsResponse := post(
		replacePath(portal, "/api/sessions"),
		"application/json",
		data,
	)

	sessionsCode := struct {
		Radius struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		} `json:"radius"`
	}{}

	if err := json.Unmarshal(sessionsResponse, &sessionsCode); err != nil {
		log.Fatal(err)
	}

	return sessionsCode.Radius.Login, sessionsCode.Radius.Password
}

func sendToken(portal *url.URL, login, password string) {
	data := url.Values{
		"username":       {login},
		"password":       {password},
		"success_url":    {"https://www.google.fr/"},
		"error_url":      {replacePath(portal, "/connect-error")},
		"update_session": {"0"},
	}

	post(
		"https://wireless.wifirst.net/goform/HtmlLoginRequest",
		"application/x-www-form-urlencoded",
		[]byte(data.Encode()),
	)
}

func post(u, contentType string, data []byte) []byte {
	response, err := http.Post(u, contentType, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return body
}

/* TOOLS FUNCTION */

// Create a new URL by replacing the path and output the result as string
func replacePath(u *url.URL, path string) string {
	return u.ResolveReference(&url.URL{Path: path}).String()
}

// Log the request method and URL, then make the request with wraped http round tripper.
type LogRoundTripper struct {
	http.RoundTripper
}

func (l LogRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	log.Println(request.Method, request.URL)
	return l.RoundTripper.RoundTrip(request)
}

func init() {
	http.DefaultTransport = LogRoundTripper{http.DefaultTransport}
}
