package main

import (
   "flag"
   "fmt"
   "os"

   "github.com/danemortensen/heartbeat"
)

var (
   ipStr       string
   masterAddr  string
)

func checkError(err error) {
   if err != nil {
      fmt.Println("Error: ", err)
      os.Exit(1)
   }
}

func checkArgs() {
   if ipStr == "" {
      fmt.Println("My ip string is required")
      os.Exit(1)
   }
}

func init() {
   flag.StringVar(&ipStr, "addr", "", "My ip string (<ip>:<port>)")
   flag.StringVar(&masterAddr, "master", "",
                  "Master's ip string (if any) (<ip>:<port>)")
}

func main() {
   flag.Parse()
   checkArgs()

   if masterAddr == "" {      // I am the master node
      me := heartbeat.Master {
         Heartbeater: heartbeat.Heartbeater {
            IpStr: ipStr,
         },
         Members: []string{},
      }
      me.BeMaster()
   } else {                  // I am a worker node
      me := heartbeat.Worker {
         Heartbeater: heartbeat.Heartbeater {
            IpStr: ipStr,
         },
         MasterAddr: masterAddr,
      }
      me.BeWorker()
   }
}
