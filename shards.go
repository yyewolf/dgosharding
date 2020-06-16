package dgosharding

import (
	"fmt"
	"strings"
	"time"
)

const (
	// EventConnected is sent when the connection to the gateway was established
	EventConnected EventType = iota

	// EventDisconnected is sent when the connection is lose
	EventDisconnected

	// EventResumed is sent when the connection was sucessfully resumed
	EventResumed

	// EventReady is sent on ready
	EventReady

	// EventOpen is sent when Open() is called
	EventOpen

	// EventClose is sent when Close() is called
	EventClose

	// EventError is sent when an error occurs
	EventError
)

//Status represents the statuses of all shards.
type Status struct {
	Shards    []*ShardStatus `json:"shards"`
	NumGuilds int            `json:"num_guilds"`
}

//ShardStatus represents the status of a single shard.
type ShardStatus struct {
	Shard     int  `json:"shard"`
	OK        bool `json:"ok"`
	Started   bool `json:"started"`
	NumGuilds int  `json:"num_guilds"`
}

// Event holds data for an event
type Event struct {
	Type EventType

	Shard     int
	NumShards int

	Msg string

	// When this event occured
	Time time.Time
}

//String is used to convert an event into string (for debugging).
func (c *Event) String() string {
	prefix := ""
	if c.Shard > -1 {
		prefix = fmt.Sprintf("[%d/%d] ", c.Shard, c.NumShards)
	}

	s := fmt.Sprintf("%s%s", prefix, strings.Title(c.Type.String()))
	if c.Msg != "" {
		s += ": " + c.Msg
	}

	return s
}

//EventType is the type of an event.
type EventType int

var (
	eventStrings = map[EventType]string{
		EventOpen:         "opened",
		EventClose:        "closed",
		EventConnected:    "connected",
		EventDisconnected: "disconnected",
		EventResumed:      "resumed",
		EventReady:        "ready",
		EventError:        "error",
	}

	eventColors = map[EventType]int{
		EventOpen:         0xec58fc,
		EventClose:        0xff7621,
		EventConnected:    0x54d646,
		EventDisconnected: 0xcc2424,
		EventResumed:      0x5985ff,
		EventReady:        0x00ffbf,
		EventError:        0x7a1bad,
	}
)

func (c EventType) String() string {
	return eventStrings[c]
}
