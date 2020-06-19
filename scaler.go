package dgosharding

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

// RestartAll restarts all the shard sessions and scales up if needed
func (m *SessionManager) RestartAll() (err error) {
	m.StopAll()
	m.Lock()
	m.numShards = int(float64(m.numShards) * 1.5)
	if err != nil {
		return err
	}
	i := 0
	for _, v := range m.Sessions {
		if i != 0 {
			// One identify every 5 seconds
			time.Sleep(time.Second * 5)
		}
		log.Println("Reconnecting")
		v.ShardCount = m.numShards
		v.ShardID = i
		if e := v.Open(); e != nil {
			err = e
			return
		}
		i++
	}
	//No rescale
	if len(m.Sessions) >= m.numShards {
		m.Unlock()
		return
	}
	//Do rescale
	for i <= m.numShards {
		if i != 0 {
			// One identify every 5 seconds
			time.Sleep(time.Second * 5)
		}
		log.Println("Reconnecting")
		m.Sessions = append(m.Sessions, &discordgo.Session{})
		err := m.initSession(i)
		if err != nil {
			m.Unlock()
			return errors.WithMessage(err, "rescaling")
		}
		i++
	}

	m.Unlock()
	return
}
