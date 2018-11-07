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

const X = 2
const Y = 1
const Z = 8

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

type DeathNotice struct {
   Death          bool
}

type HBStatus struct {
   Counter        int
   LastBeat       time.Time
}

type TableHolder struct {
   HBTable        map[string]HBStatus
}

type Heartbeater struct {
   IpStr          string
   Neighbors                     // ip strings of neighbors I'm responsible for
   HBTable        map[string]HBStatus
   DeathFlag      bool
   DeathLeft      DeathNotice
   DeathRight     DeathNotice
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
   fmt.Println("New worker at", me.WorkerAddress.Address)
   me.Members = append(me.Members, me.WorkerAddress.Address)
   responseData := map[string]int {
      "DeathTime": len(me.Members) * Z,
   }
   json.NewEncoder(w).Encode(responseData)

   if len(me.Members) >= 3 {
      me.AssignNeighbors()
   }
}

func (me *Master) AssignNeighbors() {
   total := len(me.Members)
   me.Heartbeater.Neighbors = Neighbors {
      Left: me.Members[total - 1],
      Right: me.Members[1],
   }

   me.Heartbeater.initTable();

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
   }
}

func (me *Heartbeater) ReceiveBeat(w http.ResponseWriter, r *http.Request) {
   var ipStr WorkerAddress
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&ipStr)
   checkError(err)

   responseData := map[string]bool {
      "Death": me.DeathFlag,
   }
   json.NewEncoder(w).Encode(responseData)
}

func (me *Heartbeater) PrintTable() {
   fmt.Println("\nTable:")
   for key, val := range me.HBTable {
      fmt.Println("ID:", key)
      fmt.Println("HB Counter:", val.Counter)
      fmt.Println("Last Beat:", val.LastBeat)
   }
}

func (me *Heartbeater) ReceiveTable(w http.ResponseWriter, r *http.Request) {
   var table TableHolder
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&table)
   checkError(err)

   for key, val := range table.HBTable {
      if key != me.Neighbors.Left && key != me.Neighbors.Right {
         me.HBTable[key] = val
      }
   }

   me.PrintTable()   
}

func (me *Heartbeater) SendBeatToLeft() {
   neighbor := me.Neighbors.Left
   url := ipToUrl(neighbor) + "/beat"
   message := map[string]string {
      "IpStr": me.IpStr,
   }
   payload, err := json.Marshal(message)
   checkError(err)
   resp, err := http.Post(url,"application/json",
      bytes.NewBuffer(payload))
   checkError(err)
   defer resp.Body.Close()

   json.NewDecoder(resp.Body).Decode(&me.DeathLeft)
   if me.DeathLeft.Death {
      last := me.HBTable[neighbor].LastBeat
      failTime := time.Now()
      if failTime.Sub(last).Seconds() > 3 * X {
         fmt.Println("Left neighbor OFFLINE")
         me.PrintTable()
         me.HBTable[neighbor] = HBStatus {
            -1,
            failTime,
         }
         me.Neighbors.Left = ""
      }
   } else {
      oldCounter := me.HBTable[neighbor].Counter
      me.HBTable[neighbor] = HBStatus {
         oldCounter + 1,
         time.Now(),
      }
   }
}

func (me *Heartbeater) SendBeatToRight() {
   neighbor := me.Neighbors.Right
   url := ipToUrl(neighbor) + "/beat"
   message := map[string]string {
      "IpStr": me.IpStr,
   }
   payload, err := json.Marshal(message)
   checkError(err)
   resp, err := http.Post(url,"application/json",
      bytes.NewBuffer(payload))
   checkError(err)
   defer resp.Body.Close()

   json.NewDecoder(resp.Body).Decode(&me.DeathRight)
   if me.DeathRight.Death {
      last := me.HBTable[neighbor].LastBeat
      failTime := time.Now()
      if failTime.Sub(last).Seconds() > 3 * X {
         fmt.Println("Right neighbor OFFLINE")
         me.PrintTable()
         me.HBTable[neighbor] = HBStatus {
            -1,
            failTime,
         }
         me.Neighbors.Right = ""
      }
   } else {
      oldCounter := me.HBTable[neighbor].Counter
      me.HBTable[neighbor] = HBStatus {
         oldCounter + 1,
         time.Now(),
      }
   }
}

func (me *Heartbeater) SendTableToLeft() {
   neighbor := me.Neighbors.Left
   url := ipToUrl(neighbor) + "/table"

   message := map[string]map[string]HBStatus {
      "HBTable": me.HBTable,
   }

   payload, err := json.Marshal(message)
   checkError(err)
   resp, err := http.Post(url,"application/json",
      bytes.NewBuffer(payload))
   checkError(err)
   defer resp.Body.Close()
}

func (me *Heartbeater) SendTableToRight() {
   neighbor := me.Neighbors.Right
   url := ipToUrl(neighbor) + "/table"

   message := map[string]map[string]HBStatus {
      "HBTable": me.HBTable,
   }

   payload, err := json.Marshal(message)
   checkError(err)
   resp, err := http.Post(url,"application/json",
      bytes.NewBuffer(payload))
   checkError(err)
   defer resp.Body.Close()
}

func (me *Heartbeater) SendBeat() {
   for range time.Tick(X * time.Second) {
      if !me.DeathFlag {
         if me.Neighbors.Left != "" {
            me.SendBeatToLeft()
         }

         if me.Neighbors.Right != "" {
            me.SendBeatToRight()
         }
      }
   }
}

func (me *Heartbeater) SendTable() {
   for range time.Tick(Y * time.Second) {
      if !me.DeathFlag {
         if me.Neighbors.Left != "" {
            me.SendTableToLeft()
         }

         if me.Neighbors.Right != "" {
            me.SendTableToRight()
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

   json.NewDecoder(resp.Body).Decode(&me.DeathInfo)
   timer := time.NewTimer(time.Duration(me.DeathInfo.DeathTime) * time.Second)
   go func() {
      <- timer.C
      me.Heartbeater.DeathFlag = true
      fmt.Println("Simulating failure...")
   }()
}

func (me *Heartbeater) initTable() {
   initStatusL := HBStatus {
      Counter: 0,
      LastBeat: time.Now(),
   }

   initStatusR := HBStatus {
      Counter: 0,
      LastBeat: time.Now(),
   }

   me.HBTable = map[string]HBStatus {
      me.Neighbors.Left: initStatusL,
      me.Neighbors.Right: initStatusR,
   }
}

func (me *Worker) ReceiveNeighbors(w http.ResponseWriter, r *http.Request) {
   decoder := json.NewDecoder(r.Body)
   err := decoder.Decode(&me.Heartbeater.Neighbors)
   checkError(err)
   me.initTable()
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
   http.HandleFunc("/table", me.ReceiveTable)
   checkError(err)
   go me.SendBeat()
   go me.SendTable()
   log.Fatal(http.Serve(listener, nil))
}

func ipToUrl(ip string) (string) {
   return "http://" + ip
}
