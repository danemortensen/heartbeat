package heartbeat

import (
   "fmt"
   "net"
   "net/http"
   "os"
   "time"
)

type Heartbeater struct {
   BeatDelay      time.Time      // delay between heartbeats
   TableDelay     time.Time      // delay between sending table to neighbors
   Neighbors      []string       // ip strings of neighbors I'm responsible for
}

type Master struct {
   Heartbeater
   Members        []string       // ip strings of heartbeaters
}

type Worker struct {
   Heartbeater
   MasterAddr     string         // ip string of master
}

func (me *Master) AddHeartbeater(w http.ResponseWriter, r *http.Request) {
   ip, port, err := net.SplitHostPort(r.RemoteAddr)
   checkError(err)
   fmt.Println("New heartbeater at", ip, "on port", port)
}

func (me *Worker) BeatLoop() {
   for range time.Tick(time.Second) {
      resp, err := http.Get(me.MasterAddr)
      checkError(err)
      defer resp.Body.Close()
   }
}

func checkError(err error) {
   if err != nil {
      fmt.Println("Error: ", err)
      os.Exit(1)
   }
}
