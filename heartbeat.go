package heartbeat

import (
   "encoding/json"
   "fmt"
   "log"
   "net"
   "net/http"
   "os"
   "time"
   "bytes"
)

type Delays struct {
   BeatDelay      time.Duration
   TableDelay     time.Duration
}

type Neighbors struct {
   First          string
   Second         string
}

type Heartbeater struct {
   IpStr          string
   Delays
   Neighbors                     // ip strings of neighbors I'm responsible for
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

   responseData := map[string]time.Duration {
      "BeatDelay": 1 * time.Second,
      "TableDelay": 5 * time.Second,
   }
   json.NewEncoder(w).Encode(responseData)

   go me.AssignNeighbors()
   if len(me.Members) >= 3 {
      
   }
}

func (me *Master) AssignNeighbors() {
   // me.Heartbeater.Neighbors = Neighbors {
   //    First: me.Members[1],
   //    Second: me.Members[2],
   // }
   total := len(me.Members)
   for i, ip := range me.Members[1:] {
      fmt.Println("handling neighbor " + ip)
      first := me.Members[(i + 1) % total]
      second := me.Members[(i + 2) % total]
      message := map[string]string{
         "First": first,
         "Second": second,
      }
      payload, err := json.Marshal(message)
      checkError(err)

      //url := ipToUrl(ip)
      url := ipToUrl("localhost:3001")
      fmt.Println("sending to " + url)

      resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
      checkError(err)
      resp.Body.Close()
   }
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
   json.NewDecoder(resp.Body).Decode(&me.Delays)
   fmt.Println(me.Delays.BeatDelay)
   fmt.Println(me.Delays.TableDelay)
}

func (me *Worker) ReceiveNeighbors(w http.ResponseWriter, r *http.Request) {
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&me.Heartbeater.Neighbors)
   checkError(err)
   fmt.Println(me.Heartbeater.Neighbors.First,
      me.Heartbeater.Neighbors.Second)
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
   http.HandleFunc("/", me.ReceiveNeighbors)
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
