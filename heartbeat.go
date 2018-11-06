package heartbeat

import (
   "fmt"
   "log"
   "net"
   "net/http"
   "os"
   "time"
)

type Heartbeater struct {
   IpStr          string
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

func (me *Worker) beatLoop() {
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

func (me *Master) BeMaster() {
   fmt.Println("New master at", me.IpStr)
   http.HandleFunc("/", me.AddHeartbeater)
   me.beHeartbeater()
}

func (me *Worker) BeWorker() {
   fmt.Println("New worker at", me.IpStr)
   go me.beatLoop()
   me.beHeartbeater()
}

func (me *Heartbeater) beHeartbeater() {
   listener, err := net.Listen("tcp", me.IpStr)
   checkError(err)
   log.Fatal(http.Serve(listener, nil))
}
