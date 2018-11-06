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
   me.Members = append(me.Members, r.RemoteAddr)
   fmt.Printf("%v\n", me.Members)
}

func (me *Heartbeater) ReceiveBeat(w http.ResponseWriter, r *http.Request) {

}

func (me *Worker) beatLoop() {
   for range time.Tick(time.Second) {
      url := ipToUrl(me.MasterAddr) + "/beat"
      resp, err := http.Get(url)
      checkError(err)
      defer resp.Body.Close()
   }
}

func (me *Worker) connect() {
   url := ipToUrl(me.MasterAddr)
   resp, err := http.Get(url)
   checkError(err)
   defer resp.Body.Close()
}

func checkError(err error) {
   if err != nil {
      fmt.Println("Error: ", err)
      os.Exit(1)
   }
}

func (me *Master) BeMaster() {
   fmt.Println("New master at", me.IpStr)
   me.Members = append(me.Members, me.IpStr)
   fmt.Printf("%v\n", me.Members)
   http.HandleFunc("/", me.AddHeartbeater)
   me.beHeartbeater()
}

func (me *Worker) BeWorker() {
   fmt.Println("New worker at", me.IpStr)
   go me.connect()

   me.beHeartbeater()
}

func (me *Heartbeater) beHeartbeater() {
   listener, err := net.Listen("tcp", me.IpStr)
   checkError(err)
   log.Fatal(http.Serve(listener, nil))
}

func ipToUrl(ip string) (string) {
   return "http://" + ip
}
