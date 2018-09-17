package gotell

import (
	"log"
	"net/http"
	"net/http/pprof"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jayaramsankara/gotell/ws"
	"gopkg.in/redis.v5"
)

var logs = ws.Logs

func attachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Manually add support for paths linked to by index page at /debug/pprof/
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
}

//InitServer ... Initialize the server
func InitServer(httpHost string, httpPort int, redisConf *redis.Options) error {

	logs.Println("Initiating Websocket server with redis pub-sub")

	r := mux.NewRouter().StrictSlash(true)
	attachProfiler(r)
	r.HandleFunc("/ws/{clientid}", ws.ServeWs).Methods("GET")
	r.HandleFunc("/notify/{clientid}", ws.ServeNotify).Methods("POST")
	r.HandleFunc("/notify/{clientid}", ws.ServeNotifyCORS).Methods("OPTIONS")
	r.HandleFunc("/apns/{devicetoken}", ws.ServeApns).Methods("POST")

	logs.Println("Initializing redis pub-sub for websocket message notification")
	err := ws.InitPubSub(redisConf)
	if err != nil {
		log.Fatalln("Failed to initialize pubsub for websocket notification service")
	}

	logs.Println("Initializing web server for websocket and rest requests.")
	return http.ListenAndServe(httpHost+":"+strconv.Itoa(httpPort), r)
}
