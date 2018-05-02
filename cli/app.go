package cli

import (
	"log"
	"net/url"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var app = kingpin.New("recryptportal", "RECRYPT DApp Server")

var recryptRPC = app.Flag("recrypt-rpc", "URL of recrypt RPC service").Envar("RECRYPT_RPC").Default("").String()

func Run() {
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func getRecryptRPCURL() *url.URL {
	if *recryptRPC == "" {
		log.Fatalln("Please set RECRYPT_RPC to recryptd's RPC URL")
	}

	url, err := url.Parse(*recryptRPC)
	if err != nil {
		log.Fatalln("RECRYPT_RPC URL:", *recryptRPC)
	}

	if url.User == nil {
		log.Fatalln("RECRYPT_RPC URL (must specify user & password):", *recryptRPC)
	}

	return url
}
