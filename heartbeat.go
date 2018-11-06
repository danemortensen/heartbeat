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

// Add HB Table struct
// send table to neighbors every Y
// HB should be POST containing sender IP
// time should be time of last received HB
// HB counter is number of beats received
// http requests should be in goroutines
// add callbacks for /beat routes
// implement member removal upon death -> update all neighbors
//    - if count < 3 -> end HB protocol
// timer function will need to know which neighbor to expect HB from
// need one timer per neighbor (left, right)
// https://mmcgrana.github.io/2012/09/go-by-example-timers-and-tickers.html
// monitor /beat route
//    - if HB received, reset timers
//    - use faint timer (X), and death timer (3 * X)
// upon death:
//    - set HB counter to -1
//    - set time to time of death
// simulate a node dying every Z

type Delays struct {
   BeatDelay      time.Duration
   TableDelay     time.Duration
}

type Neighbors struct {
   Left          string
   Right         string
}

type Heartbeater struct {
   IpStr          string
   Delays
   Neighbors                     // ip strings of neighbors I'm responsible for
}

type Master struct {
   Heartbeater
   WorkerAddress
   Members        []string       // ip strings of heartbeaters
}

type Worker struct {
   Heartbeater
   MasterAddr     string         // ip string of master
}

type WorkerAddress struct {
   Address       string
}

func (me *Master) AddHeartbeater(w http.ResponseWriter, r *http.Request) {
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&me.WorkerAddress)
   fmt.Println(me.WorkerAddress.Address)
   checkError(err)
   fmt.Println("New heartbeater at", me.WorkerAddress.Address)
   me.Members = append(me.Members, me.WorkerAddress.Address)
   fmt.Printf("%v\n", me.Members)

   responseData := map[string]time.Duration {
      "BeatDelay": 1 * time.Second,
      "TableDelay": 5 * time.Second,
   }
   json.NewEncoder(w).Encode(responseData)

   if len(me.Members) >= 3 {
      go me.AssignNeighbors()
   }
}

func (me *Master) AssignNeighbors() {
   total := len(me.Members)
   me.Heartbeater.Neighbors = Neighbors {
      Left: me.Members[total - 1],
      Right: me.Members[1],
   }
   
   for i, ip := range me.Members[1:] {
      left := me.Members[(i - 1) % total]
      right := me.Members[(i + 1) % total]
      message := map[string]string{
         "Left": left,
         "Right": right,
      }
      payload, err := json.Marshal(message)
      checkError(err)
      url := ipToUrl(ip)
      resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
      checkError(err)
      resp.Body.Close()
      fmt.Println("Sent neighbors to " + url)
   }
}

func (me *Heartbeater) ReceiveBeat(w http.ResponseWriter, r *http.Request) {

}

func (me *Heartbeater) SendBeat() {
   // change http request to post
   for range time.Tick(time.Second) {
      if me.Neighbors.Left != "" && me.Neighbors.Right != "" {
         // url := ipToUrl(me.MasterAddr) + "/beat"
         // resp, err := http.Get(url)
         // checkError(err)
         // resp.Body.Close()
      }
      
   }
}

func (me *Worker) connect() {
   url := ipToUrl(me.MasterAddr)
   message := map[string]string{
      "Address": me.Heartbeater.IpStr,
   }

   payload, err := json.Marshal(message)
   checkError(err)
   resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))

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
   fmt.Println(me.Heartbeater.Neighbors.Left,
      me.Heartbeater.Neighbors.Right)
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
   go me.SendBeat()
}

func ipToUrl(ip string) (string) {
   return "http://" + ip
}
