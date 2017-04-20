package gotell

import (
	"github.com/gorilla/mux"
	"gopkg.in/redis.v5"
	"log"
	"net/http"
	"strconv"
	"github.com/jayaramsankara/gotell/ws"
)
var logs = ws.Logs

func InitServer(httpHost string, httpPort int, redisConf *redis.Options) error {

	logs.Println("Initiating Websocket server with redis pub-sub")

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/ws/{clientid}", ws.ServeWs).Methods("GET")
	r.HandleFunc("/notify/{clientid}", ws.ServeNotify).Methods("POST")
	r.HandleFunc("/apns/{devicetoken}", ws.ServeApns).Methods("POST")

    logs.Println("Initializing redis pub-sub for websocket message notification")
	err := ws.InitPubSub(redisConf)
    if err != nil {
     	log.Fatalln("Failed to initialize pubsub for websocket notification service")
	}


	logs.Println("Initializing web server for websocket and rest requests.")
	return http.ListenAndServe(httpHost+":"+strconv.Itoa(httpPort), r)
}
