# gotell
A push notification service implemented using go

gotell is implemented as a cloudfoundry app , but can be run as a standalone app as well once the CFEnv dependencies are cleared out.

gotell uses redis pub-sub in order to perform a stateless way of websocket notification across multiple instances of the app. 
This brings in depedency to redis service which, when deployed in cloudfoundry, is injected as an user provided service.

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
* Carve out the cloudfoundry app part separately so that this one becomes a generic lib

#How to Build This Project and Push to CF
Follow the instructions below, typically the first time you are attempting to get the source and build.

* Install Go by following the instructions @ https://golang.org/doc/install
* Review and Understand https://golang.org/doc/code.html
* Choose a folder to be the root of all the go sources, packages and binaries. 
 * Typically, this will be ~/Projects/go 
* Set the env variable GOPATH to the folder chosen in the step above. 
 * e.g: export GOPATH=~/Projects/go (add this line in your ~/.bash_profile)
* Set the env variable GOROOT to the folder where go is installed. 
 * Do a 'which go' after Go package installation and see the folder, typically /usr/local/go on a MAC.
* Add $GPATH/bin and $GOROOT/bin to system PATH
* cd to GOPATH
* create the folder src/github.com/jayaramsankara
* cd to $GOPATH/src/github.com/jayaramsankara
* Clone the repo gotell to the folder  $GOPATH/src/github.com/jayaramsankara
* cd gotell
* Run the command 'godep save'.  This will copy all the dependent packages and sources to $GOPATH.
 * In case any existing package has to be updated do "go get -u " or "go get -u all" for all packages including the go system packages.
 * Run the command "godep save"


Now , you are all set to build the project. To build run the following command.

* go install


Now to push the application Cloud Foundry, just run the command ' cf push '.

Note that this app requires the user provided service 'redis' in the format '{"host":"redishost","port":"redisport"}' to be created in the cloudfoundry space to which the app has to be pushed.


