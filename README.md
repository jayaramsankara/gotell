# gotell
Cloud Components for push notification service implemented using go

gotell is a set of components that can be used to implement a push notification service.
The components support building stateless application running in PaaS, such as cloudfoundry.

gotell uses redis pub-sub in order to perform a stateless way of websocket notification across multiple instances of the app, especially when deployed in a PaaS.
 

## Functionalities
* Push notification to websocket clients
 * Websocket clients can connect to wss://gotellURL/ws/'clientid' and then any other application can do a https post to gotellURL/notify/'clientid' with body as the JSON {"message":"Value"} and the 'Value' content will be delivered to the client.
 * The Value can be another JSON, but enclosed with in double quotes (as with string) and the doublequotes that are part of the value should be escaped and no new line characters should be part of the JSON.
 * Examples: 
  * { "message": "Hello from me"}
  * { "messages": "{\"msg\":\"Hello from User1\"}"} 

## To Do
* Support for generic JSON structure as message's Value
* APNS support
* GCM support


#How to Use this library

Re-Requisite: Install Go 

* cd to GOPATH
* Run the command 'go get github.com/jayaramsankara/gotell/ws'
* Add the statement import "github.com/jayaramsankara/gotell/ws" in your .go file

## Example CloudFoundry Application
TODO


