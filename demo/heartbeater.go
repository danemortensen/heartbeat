package main

import (
   "flag"
   "fmt"
   "log"
   "net"
   "net/http"
   "os"

   "github.com/danemortensen/heartbeat"
   "github.com/gorilla/mux"
)

var (
   masterAddr string
)

func checkError(err error) {
   if err != nil {
      fmt.Println("Error: ", err)
      os.Exit(1)
   }
}

func init() {
   flag.StringVar(&masterAddr, "master", "",
                  "This heartbeater's master (if any)")
}


func main() {
   flag.Parse()
   listener, err := net.Listen("tcp", ":0")
   checkError(err)
   router := mux.NewRouter()

   if masterAddr == "" {      // I am the master node
      me := heartbeat.Master{}
      fmt.Println("New master at", listener.Addr())
      router.HandleFunc("/", me.AddHeartbeater).Methods("GET")
   } else {                  // I am a worker node
      me := heartbeat.Worker{MasterAddr: masterAddr}
      fmt.Println("New worker at", listener.Addr())
      go me.BeatLoop()
   }

   log.Fatal(http.Serve(listener, router))
}
