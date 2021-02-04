// This code is based on:
// https://github.com/cfz42/wifirst-connect/blob/master/wifirst-connect.sh

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"time"
)

func main() {
	t := flag.String("t", "https://github.com/", "The test page")
	l := flag.String("l", "", "The login")
	p := flag.String("p", "", "The password")
	force := flag.Bool("f", false, "Force the connexion if multiple connexion exist.")
	dur := flag.Duration("dur", 0, "Duration between two authentification, zero for only one authentification")
	flag.Parse()

	run(*t, *l, *p, *force)
	if *dur > 0 {
		for range time.Tick(*dur) {
			run(*t, *l, *p, *force)
		}
	}
}

func run(t, l, p string, force bool) {
	cl := newClient()
	if err := recup(&cl, t, nil, nil); err == nil {
		log.Println("[STATUS] Already connected")
		return
	}
	log.Println("[STATUS] no connected")

	cl = newClient()
	m1 := gm("authenticity_token")
	if force {
		if err := recup(&cl, "https://connect.wifirst.net/?perform=true&ignore_conflicts=true&reason=Device", nil, m1); err != nil {
			log.Println("[ERROR]", err)
			return
		}
	} else {
		if err := recup(&cl, "https://connect.wifirst.net/?perform=true", nil, m1); err != nil {
			log.Println("[ERROR]", err)
			return
		}
	}

	m2 := gm("username", "password")
	if err := recup(&cl, "https://selfcare.wifirst.net/sessions", url.Values{
		"utf8":               []string{"✓"},
		"authenticity_token": []string{m1["authenticity_token"]},
		"login":              []string{l},
		"password":           []string{p},
	}, m2); err != nil {
		log.Println("[ERROR]", err)
		return
	}

	if err := recup(&cl, "https://wireless.wifirst.net:8090/goform/HtmlLoginRequest", url.Values{
		"commit":      []string{"Se connecter"},
		"username":    []string{m2["username"]},
		"password":    []string{m2["password"]},
		"qos_class":   []string{""},
		"success_url": []string{"https://apps.wifirst.net/?redirected=true"},
		"error_url":   []string{"https://connect.wifirst.net/login_error"},
	}, nil); err != nil {
		log.Println("[ERROR]", err)
		return
	}
	log.Println("end of the connexion opration")
}

// Create a new Client with Cookie Jar.
func newClient() http.Client {
	j, _ := cookiejar.New(nil)
	return http.Client{Jar: j}
}

// Generate the string map
func gm(keys ...string) map[string]string {
	m := make(map[string]string, len(keys))
	for _, k := range keys {
		m[k] = ""
	}
	return m
}

// With the client cl, Get or POST (value is not nil) a request and set the keys
// from the response body.
func recup(cl *http.Client, u string, form url.Values, keys map[string]string) error {
	var rep *http.Response
	var err error
	if form == nil {
		rep, err = cl.Get(u)
	} else {
		rep, err = cl.PostForm(u, form)
	}
	if err != nil {
		return fmt.Errorf("Request to %q fail: %w", u, err)
	} else if rep.StatusCode != http.StatusOK {
		return fmt.Errorf("Request to %q fail: %q", u, rep.Status)
	}

	_body, err := ioutil.ReadAll(rep.Body)
	body := string(_body)
	rep.Body.Close()
	if err != nil {
		return fmt.Errorf("Read body from %q fail:\r\n\t%w", u, err)
	}

	for k := range keys {
		r := regexp.MustCompile(`.*` + k + `.*value="(.*)".*`)
		v := r.ReplaceAllString(r.FindString(body), "$1")
		if v == "" {
			return fmt.Errorf("Get the info %q not found in %q", k, u)
		}
		keys[k] = v
	}

	return nil
}
