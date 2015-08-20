# gotell
Cloud Components for push notification service implemented using go

gotell is a set of components that can be used to implement a push notification service.
The components support building stateless application running in PaaS, such as cloudfoundry.

gotell uses redis pub-sub in order to perform a stateless way of websocket notification across multiple instances of the app, especially when deployed in a PaaS.
 

## Functionalities
* Push notification to websocket clients
 * Websocket clients can connect to wss://gotellURL/ws/'clientid' and then any other application can do a https post to gotellURL/notify/'clientid' with body as the JSON {"message":"Value"} and the 'Value' content will be delivered to the client.
 * The Value can be another JSON, but enclosed with in double quotes (as with string) and the doublequotes that are part of the value should be escaped (replace " with \" ).
 * Examples: 
   * { "message": "Hello from me"}
    * { "message": "{\"msg\":\"Hello from User1\"}"} 
* APNS support
 * Requires app cert and unencrypted key pem files.
   * The cert and key file name is read from the environment variable 'APNS_CERT' and 'APNS_KEY' respectively.
 * Do http POST https://\<gotellURL\>/apns/\<devicetoken\>  with header Content-Type : application/json and body as   '{"message":"Message Value","badge":0,"sound":"default"}'
   * devicetoken should be the device token of the device to which the notification has to be sent.
    * Message Value can be another JSON,but enclosed with in double quotes (as with string) and the doublequotes that are part of the value should be escaped (replace " with \" ).
     * The badge  value has to be an integer and badge count set will be displayed in the app icon
      * The sound value can be any aiff file name or pass the string "default" for default notification sound.
 

## To Do
* GCM support


#How to Use this library

Re-Requisite: Install Go 

* cd to GOPATH
* Run the command 'go get github.com/jayaramsankara/gotell'
* Add the statement 'import "github.com/jayaramsankara/gotell"' in your main go file
* Initiate the notification server by invoking gotell.InitServer with appropriate params

### Example
// Start the web socket server and wait in a loop

	err := gotell.InitServer(httpHost, httpPort, redisOptions)
	
	if err != nil {
	
		log.Println("Failed to initiate notification service.", err)
		
	}

## Example CloudFoundry Application 
TODO


