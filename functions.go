package dgosharding

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// AddHandler adds an event handler to all shards
// All event handlers will be added to new sessions automatically.
func (m *SessionManager) AddHandler(handler interface{}) {
	m.Lock()
	defer m.Unlock()
	m.eventHandlers = append(m.eventHandlers, handler)

	if len(m.Sessions) > 0 {
		for _, v := range m.Sessions {
			v.AddHandler(handler)
		}
	}
}

// GuildCount will count the number of guilds.
func (m *SessionManager) GuildCount() []int {
	m.RLock()
	nShards := m.numShards
	result := make([]int, nShards)

	for i, session := range m.Sessions {
		if session == nil {
			continue
		}
		session.State.RLock()
		result[i] = len(session.State.Guilds)
		session.State.RUnlock()
	}

	m.RUnlock()
	return result
}

// SessionForGuildS is the same as SessionForGuild but accepts the guildID as a string for convenience
func (m *SessionManager) SessionForGuildS(guildID string) *discordgo.Session {
	// Question is, should we really ignore this error?
	// In reality, the guildID should never be invalid but...
	parsed, _ := strconv.ParseInt(guildID, 10, 64)
	return m.SessionForGuild(parsed)
}

// SessionForGuild returns the session for the specified guild
func (m *SessionManager) SessionForGuild(guildID int64) *discordgo.Session {
	// (guild_id >> 22) % num_shards == shard_id
	// That formula is taken from the sharding issue on the api docs repository on github
	m.RLock()
	defer m.RUnlock()
	shardID := (guildID >> 22) % int64(m.numShards)
	return m.Sessions[shardID]
}

// Session retrieves a session from the sessions map, rlocking it in the process
func (m *SessionManager) Session(shardID int) *discordgo.Session {
	m.RLock()
	defer m.RUnlock()
	return m.Sessions[shardID]
}

// SessionForDMs returns the session to send DMs.
func (m *SessionManager) SessionForDMs() *discordgo.Session {
	m.RLock()
	defer m.RUnlock()
	shardID := 0
	return m.Sessions[shardID]
}
