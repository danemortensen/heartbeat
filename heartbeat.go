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

const X = 1
const Y = 5
const Z = 10

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

type Neighbors struct {
   Left          string
   Right         string
}

type DeathInfo struct {
   DeathTime      int
}

type HBStatus struct {
   Counter        int
   LastBeat       time.Time
}

type Heartbeater struct {
   IpStr          string
   Neighbors                     // ip strings of neighbors I'm responsible for
   HBTable        map[string]HBStatus
}

type Master struct {
   Heartbeater
   WorkerAddress
   Members        []string       // ip strings of heartbeaters
}

type Worker struct {
   Heartbeater
   MasterAddr     string         // ip string of master
   DeathInfo     
}

type WorkerAddress struct {
   Address       string
}

func (me *Master) AddHeartbeater(w http.ResponseWriter, r *http.Request) {
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&me.WorkerAddress)
   checkError(err)
   fmt.Println("New heartbeater at", me.WorkerAddress.Address)
   me.Members = append(me.Members, me.WorkerAddress.Address)
   fmt.Printf("Members: %v\n", me.Members)

   responseData := map[string]int {
      "DeathTime": len(me.Members) * Z,
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

   for i := 1; i < len(me.Members); i++ {
      left := me.Members[(i - 1) % total]
      right := me.Members[(i + 1) % total]
      message := map[string]string{
         "Left": left,
         "Right": right,
      }
      payload, err := json.Marshal(message)
      checkError(err)
      url := ipToUrl(me.Members[i]) + "/neighbors"
      resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
      checkError(err)
      resp.Body.Close()
      fmt.Println("Sent neighbors to", me.Members[i])
   }
}

func (me *Heartbeater) ReceiveBeat(w http.ResponseWriter, r *http.Request) {
   var ipStr WorkerAddress
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&ipStr)
   checkError(err)
   status := HBStatus {
      Counter: me.HBTable[ipStr.Address].Counter + 1,
      LastBeat: time.Now(),
   }
   fmt.Printf("%v", status)
}

func (me *Heartbeater) SendToNeighbor(neighbor string) (bool) {
   url := ipToUrl(neighbor) + "/beat"
   message := map[string]string {
      "IpStr": me.IpStr,
   }
   payload, err := json.Marshal(message)
   checkError(err)
   fmt.Println("Sending heartbeat to", url)
   resp, err := http.Post(url,"application/json", 
      bytes.NewBuffer(payload))
   defer resp.Body.Close()

   if err != nil {
      last := me.HBTable[neighbor].LastBeat
      failTime := time.Now()
      if failTime.Sub(last).Seconds() > 3 * X {
         me.HBTable[neighbor].Counter = -1
         me.HBTable[neighbor].LastBeat = failTime
         return false
      }
   } else {
      me.HBTable[neighbor].Counter++
      me.HBTable[neighbor].LastBeat = time.Now()
   }

   return true
}

func (me *Heartbeater) SendBeat() {
   for range time.Tick(X * time.Second) {
      if me.Neighbors.Left != "" {
         if (!me.SendToNeighbor(me.Neighbors.Left)) {
            me.Neighbors.Left = ""
         }
      }

      if me.Neighbors.Right != "" {
         if (!me.SendToNeighbor(me.Neighbors.Right)) {
            me.Neighbors.Right = ""
         }
      }
   }
}

func (me *Worker) connect() {
   url := ipToUrl(me.MasterAddr) + "/add"
   message := map[string]string{
      "Address": me.Heartbeater.IpStr,
   }
   payload, err := json.Marshal(message)
   checkError(err)
   resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
   checkError(err)
   defer resp.Body.Close()

   json.NewDecoder(resp.Body).Decode(&me.Heartbeater.DeathInfo)
   fmt.Println("My death time is", me.Heartbeater.DeathInfo.DeathTime)
}

func (me *Worker) ReceiveNeighbors(w http.ResponseWriter, r *http.Request) {
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&me.Heartbeater.Neighbors)
   checkError(err)
   fmt.Println("Received neighbors:",
      me.Heartbeater.Neighbors.Left,
      me.Heartbeater.Neighbors.Right)

   me.HBTable = map[string]HBStatus {
      me.Neighbors.Left: HBStatus {
         Counter: 0,
         LastBeat: time.Now(),
      },
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
   me.Members = append(me.Members, me.IpStr)
   fmt.Printf("%v\n", me.Members)
   http.HandleFunc("/add", me.AddHeartbeater)
   me.beHeartbeater()
}

func (me *Worker) BeWorker() {
   fmt.Println("New worker at", me.IpStr)
   http.HandleFunc("/neighbors", me.ReceiveNeighbors)
   go me.connect()
   me.beHeartbeater()
}

func (me *Heartbeater) beHeartbeater() {
   listener, err := net.Listen("tcp", me.IpStr)
   http.HandleFunc("/beat", me.ReceiveBeat)
   checkError(err)
   go me.SendBeat()
   log.Fatal(http.Serve(listener, nil))
}

func ipToUrl(ip string) (string) {
   return "http://" + ip
}
