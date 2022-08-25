package sn

import "time"

var (
	ExitChan  = make(chan bool) // send to shutdown
	Running   bool              // true when skynet running, false when restart scheduled
	StartTime time.Time         // skynet start time
)

// Version is skynet version.
const Version = "1.0.0"

const ProtoVersion = 1
