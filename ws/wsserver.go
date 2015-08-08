//wsserver.go
package ws

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gopkg.in/redis.v3"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/jayaramsankara/gotell/apns"
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
	maxRedisConn = 50
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clientConnections = map[string]*wsconnection{}

var redisSender chan *NotifyData

var receiver *redis.PubSub

var logs = log.New(os.Stdout, "INFO", log.LstdFlags)

var Logs = logs

type wsconnection struct {
	// The websocket connection.
	ws       *websocket.Conn
	clientId string

	//The redis pubsubs channel
	// Buffered channel of outbound messages.
	//receive *redis.PubSub
	send   chan ([]byte)
	active bool
}

type redisclients struct {
	receiver *redis.PubSub
	sender   *redis.Client
}

type WsMessage struct {
	Message string `json:"message"`
}

type NotifyData struct {
	ClientId string
	Message  string
}

func (conn *wsconnection) Close() {
	conn.write(websocket.CloseMessage, []byte{})
	conn.ws.Close()
}

func (conn *wsconnection) sendMessages() {
	clientId := conn.clientId
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		logs.Println("Exiting sendMessages. Removing the connection mapped to " + conn.clientId)
		conn.active = false
		logs.Println("Closing websocket connection for ", conn.clientId)
		conn.Close()
		logs.Println("Removing subscription to redis channel for ", conn.clientId)
		receiver.Unsubscribe(conn.clientId)
		logs.Println("Exiting sendMessages. Removing the connection mapped to " + conn.clientId)
		delete(clientConnections, conn.clientId)
	}()

	for {
		select {
		case message, ok := <-conn.send:
			if !ok {
				log.Println("No more messages to be sent to WS connection from channel for  ", conn.clientId)
				return
			}
			logs.Println("Sending WS message for client  " + clientId)
			if err := conn.write(websocket.TextMessage, message); err != nil {
				log.Println("Error while sending WS message  ", conn.clientId, message, err)
				return
			}
		case <-ticker.C:
			logs.Println("Sending Ping WS message for client  " + clientId)
			if err := conn.write(websocket.PingMessage, []byte{}); err != nil {
				log.Println("Error while sending WS ping message  ", conn.clientId, err)
				return
			}
		}
	}

}

func (conn *wsconnection) receiveMessages() {

	go func() {
		for {
			if _, _, err := conn.ws.NextReader(); err != nil {
				log.Println("Error while receiving WS pong message  ", conn.clientId, err)
				conn.active = false
				conn.Close()
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

func InitPubSub(redisConf *redis.Options) error {
	redisSender = make(chan *NotifyData, redisConf.PoolSize)
	receiver = redis.NewClient(redisConf).PubSub()

	var publisher = redis.NewClient(redisConf)

	logs.Println("Initialized redis clients for pub and sub.")
	
	//Init Redis receiver
	go func() {
		for {
			var msgi, err = receiver.Receive()
			if err != nil {
				log.Println("Error while receive message from redis pubsub receiver", err)
				// handle failure.. reinit redis pub sub
				log.Println("Reinitializing Redis PubSub.. and exiting the receive handler routine")
				close(redisSender)
				receiver.Close()
				publisher.Close()
				InitPubSub(redisConf)
				return
				
				
			} else {

				switch msg := msgi.(type) {
				case *redis.Subscription:
					logs.Println("Messge from channel : ", msg.Kind, msg.Channel)
				case *redis.Message:
					logs.Println("Received message from Redis channel ", msg.Payload, msg.Channel)
					go func(clientId string, message string) {

						logs.Println("Sending the message to WS send channel ", message, clientId)
						clientConnections[clientId].send <- []byte(message)
					}(msg.Channel, msg.Payload)

				case *redis.Pong:
					logs.Println("Pong message from channel : ", msg)
				default:
					log.Printf("Error unknown message: %#v", msgi)
				}

			}

		}

	}()

	go func(msgsForRedis chan *NotifyData) {
		for {
			data, ok := <-msgsForRedis
			if !ok {
				log.Println("RedisSender channel seems to be closed. Exiting the routine")
				return
			}
			logs.Println("Received notification data from redisSender ", data.ClientId)
			go func() {
				logs.Println("Publishing message to the channel  ", data.Message, data.ClientId)
				err := publisher.Publish(data.ClientId, data.Message).Err()
				if err != nil {
					log.Println("Error in publishing event to redis", err)
				}
			}()

		}

	}(redisSender)
	return nil
}



func ServeApns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceToken := vars["devicetoken"]

	//extract body

	decoder := json.NewDecoder(r.Body)
	var data apns.ApnsMessage
	err := decoder.Decode(&data)

	if err != nil {
		log.Println("Error while extracting the apns message.", err)
		w.WriteHeader(http.StatusInternalServerError)

	} else {
	// send data to conn
		logs.Println("Handling apns:  The extracted message is : ", data.Message,data.Badge,data.Sound)
	    apns.Notify(&data,deviceToken)
		w.WriteHeader(http.StatusOK)
	}
	
}


//serveNotify receives the API, parses the body and sends the message to the corresponding
// websocket. Returns error if no websocket conn exists for a client id or send fails
func ServeNotify(w http.ResponseWriter, r *http.Request) {

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
		logs.Println("Handling notify:  The extracted message is : ", data.Message, clientId)
		notifyData := &NotifyData{
			ClientId: clientId,
			Message:  data.Message,
		}
		logs.Println("Handling notify:  Sending notification data to redisSender ", clientId)
		redisSender <- notifyData
		w.WriteHeader(http.StatusOK)
	}

}

// serverWs handles websocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {

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
