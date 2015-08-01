//wsserver.go
package ws

import (
	"os"
	"encoding/json"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gopkg.in/redis.v3"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
	
	// Max NUmber of Redis connections maintained for publishing events
	// This is same as the Go Channel buffer size for sending events for redis
	maxRedisConn =  50
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clientConnections = map[string]*wsconnection{}

var redisSender = make(chan *NotifyData, maxRedisConn)

var receiver *redis.PubSub

var logs = log.New(os.Stdout,"INFO",log.LstdFlags)

type wsconnection struct {
	// The websocket connection.
	ws       *websocket.Conn
	clientId string

	//The redis pubsubs channel
	// Buffered channel of outbound messages.
	//receive *redis.PubSub
	send    chan ([]byte)
	active  bool
}

type redisclients struct {
	receiver *redis.PubSub
	sender   *redis.Client
}

type WsMessage struct {
	Message string `json:"message"`
}

type NotifyData struct {
	ClientId  string
	Message string
}

func (conn *wsconnection) sendMessages() {
	clientId := conn.clientId
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		log.Println("Exiting sendMessages. Removing the connection mapped to " + conn.clientId)
		conn.active = false
		log.Println("Closing websocket connection for ", conn.clientId)
		conn.ws.Close()
		log.Println("Removing subscription to redis channel for ",conn.clientId)
		receiver.Unsubscribe(conn.clientId)
		log.Println("Exiting sendMessages. Removing the connection mapped to " + conn.clientId)
  		delete(clientConnections, conn.clientId)
	}()
	
	for {
		select {
		case message, ok := <-conn.send:
			if !ok {

				conn.write(websocket.CloseMessage, []byte{})

				log.Println("Error while fetching message from channel for  ", conn.clientId)
				return
			}
			logs.Println("Sending WS message for client  " + clientId)
			if err := conn.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			logs.Println("Sending Ping WS message for client  " + clientId)
			if err := conn.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}

}

func (conn *wsconnection) receiveMessages() {
	
	go func () {
		    for {
		        if _, _, err := conn.ws.NextReader(); err != nil {
		            
					conn.active=false
					conn.ws.Close()
		            break
		        }
		    }
		}()
		
	conn.ws.SetReadLimit(maxMessageSize)
	conn.ws.SetReadDeadline(time.Now().Add(pongWait))
	conn.ws.SetPongHandler(func(string) error {
		logs.Println("PongHandler for client  " + conn.clientId)
		conn.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
}

// write writes a message with the given message type and payload.
func (conn *wsconnection) write(messageType int, payload []byte) error {
	conn.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.ws.WriteMessage(messageType, payload)
}

func InitServer() error {
	
	logs.Println("Initiating Websocket server with redis pub-sub")
	
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/ws/{clientid}", serveWs).Methods("GET")
	r.HandleFunc("/notify/{clientid}", serveNotify).Methods("POST")

	var httpHost = "localhost"
	var httpPort = 8080

	var redisHost = "72.163.168.10"
	var redisPort = "6379"

	appEnv, err := cfenv.Current()
	if err != nil {
		log.Println("WSServer: Failed to read CF Environment variables ", err)
		//os.Exit(15)
	} else {
		httpHost = appEnv.Host
		httpPort = appEnv.Port
		
		services, err := appEnv.Services.WithName("redis")
		if err != nil {
			log.Println("Failed to read redis details from VCAP_SERVICES . ", err)
			//os.Exit(20)
		} else {
			redisconn := services.Credentials
			redisHost, _ = redisconn["host"].(string)
			redisPort, _ = redisconn["port"].(string)
		}
	}

	

	receiver = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: "", // no password set
		DB:       0,  // use default DB
		MaxRetries : 3, // Max retries on operation errors
	}).PubSub()

	var publisher = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: "", // no password set
		DB:       0,  // use default DB
		PoolTimeout : 3, // Pool timeout
		MaxRetries : 3,
		PoolSize : maxRedisConn,
	})

	logs.Println("Initialized redis clients for pub and sub.")
	
	//Init Redis receiver 
	go func() {
		for {
			var msgi, err = receiver.Receive()
			if err != nil {
				log.Println("Error while receive message from redis pubsub receiver", err)
				//Todo handle failure
			} else {
				
				switch msg := msgi.(type) {
			    case *redis.Subscription:
			        logs.Println("Messge from channel : ",msg.Kind, msg.Channel)
			    case *redis.Message:
					logs.Println("Received message from Redis channel ", msg.Payload,msg.Channel)
				    go func (clientId string, message string) {
						
						logs.Println("Sending the message to WS send channel ", message, clientId)
						clientConnections[clientId].send <- []byte(message)
					}(msg.Channel,msg.Payload)
				    
			    case *redis.Pong:
			        logs.Println("Pong message from channel : ",msg)
			    default:
			        log.Printf("unknown message: %#v", msgi)
			    }
	
				
			}

		}

	}()
	
	go func() {
		for {
			data := <-redisSender
			logs.Println("Received notification data from redisSender " ,data.ClientId)
			go func () {
				logs.Println("Publishing message to the channel  ", data.Message,data.ClientId)
				err := publisher.Publish(data.ClientId, data.Message).Err()
				if err != nil {
					log.Println("Error in publishing event to redis", err)
				}
			}()
 			
			
		}
		
	}()

	logs.Println("Initializing web server for websocket and rest requests.")
	return http.ListenAndServe(httpHost+":"+strconv.Itoa(httpPort), r)
}

//serveNotify receives the API, parses the body and sends the message to the corresponding
// websocket. Returns error if no websocket conn exists for a client id or send fails
func serveNotify(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	clientId := vars["clientid"]

	//extract body

	decoder := json.NewDecoder(r.Body)
	var data WsMessage
	err := decoder.Decode(&data)

	if err != nil {
		log.Println("Error while extracting the message.", err)
		w.WriteHeader(http.StatusInternalServerError)

	} else {
		// send data to conn
		logs.Println("Handling notify:  The extracted message is : " , data.Message, clientId)
		notifyData := &NotifyData {
			ClientId: clientId,
			Message : data.Message, 
		}
		logs.Println("Handling notify:  Sending notification data to redisSender " ,clientId)
		redisSender <- notifyData
		w.WriteHeader(http.StatusOK)
	}

}

// serverWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error in accepting websocket connection", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//new websocket connection
	vars := mux.Vars(r)
	clientId := vars["clientid"]
	logs.Println("Creating new wsconnection for client : " + clientId)
	// Initialize redis client as subscriber

	err = receiver.Subscribe(clientId)
	if err != nil {
		errMsg := "Failed to subscribe to message channel for " + clientId
		log.Println(errMsg, err)
		ws.WriteMessage(websocket.CloseInternalServerErr, []byte(errMsg))
		ws.Close()
		return
	}

	conn := &wsconnection{active: true, clientId: clientId, ws: ws, send: make(chan []byte, 256)}
	logs.Println("Adding clientId-WsConn mapping for client  " + clientId)
	clientConnections[clientId] = conn
	go conn.sendMessages()
	conn.receiveMessages()
}