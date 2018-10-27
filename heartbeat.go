package heartbeat

import (
   "fmt"
)

type Mode int
const (
   Create Mode = iota
   Connect Mode = iota
)

type Heartbeater struct {
   mode Mode
}

var Three int

func HelloWorld() {
   fmt.Println("Hello, world!")
}
