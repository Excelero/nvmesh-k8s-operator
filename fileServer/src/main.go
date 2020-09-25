package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	auth "github.com/abbot/go-http-auth"
	"gopkg.in/yaml.v2"
)

//Secret ...
//Use https://unix4lyfe.org/crypt to generate an md5 hash + salt
func Secret(user, realm string) string {
	users := secrets.Users

	if password, ok := users[user]; ok {
		return password
	} else {
		return ""
	}
}

type AuthSecrets struct {
	Salt  string            `yaml:"salt"`
	Users map[string]string `yaml:"users"`
}

var secrets AuthSecrets

func exitIfErr(err error, msg string) {
	if err != nil {
		log.Printf("%s. Caused by: %s", msg, err)
		os.Exit(1)
	}
}

func parseAuthFile(filename string) AuthSecrets {
	bytes, err := ioutil.ReadFile(filename)
	exitIfErr(err, fmt.Sprintf("Could not find auth.yaml file at %s", filename))

	err = yaml.Unmarshal(bytes, &secrets)
	exitIfErr(err, fmt.Sprintf("Failed to parse yaml in auth yaml file at %s", filename))

	return secrets
}

func main() {
	port := flag.String("p", "8100", "port to serve on")
	directory := flag.String("d", ".", "the directory of static file to host")
	authFile := flag.String("auth", "auth.yaml", "location of the auth.yaml file")
	cert := flag.String("cert", "server.crt", "location of the server.crt certificate file")
	key := flag.String("cert-key", "server.key", "location of the server.key file")

	flag.Parse()

	parseAuthFile(*authFile)

	authenticator := auth.NewBasicAuthenticator(secrets.Salt, Secret)
	http.HandleFunc("/", authenticator.Wrap(func(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
		http.FileServer(http.Dir(*directory)).ServeHTTP(w, &r.Request)
		log.Printf("%s %s\n", r.Request.Method, r.Request.URL.Path)
	}))

	log.Printf("Serving %s on HTTPS port: %s\n", *directory, *port)
	err := http.ListenAndServeTLS(":"+*port, *cert, *key, nil)
	log.Fatal(err)
}
