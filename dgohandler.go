package dgosharding

import (
	"github.com/bwmarrin/discordgo"
)

//OnDiscordConnected will be fired when a shard connects.
func (m *SessionManager) OnDiscordConnected(s *discordgo.Session, evt *discordgo.Connect) {
	m.handleEvent(EventConnected, s.ShardID, "")
}

//OnDiscordDisconnected will be fired when a shard disconnects.
func (m *SessionManager) OnDiscordDisconnected(s *discordgo.Session, evt *discordgo.Disconnect) {
	if len(s.State.Guilds) >= 2400 {
		m.RestartAll()
	}
	m.handleEvent(EventDisconnected, s.ShardID, "")
}

//OnDiscordReady will be fired when a shard is ready.
func (m *SessionManager) OnDiscordReady(s *discordgo.Session, evt *discordgo.Ready) {
	m.handleEvent(EventReady, s.ShardID, "")
}

//OnDiscordResumed will be fired when a shard resumes.
func (m *SessionManager) OnDiscordResumed(s *discordgo.Session, evt *discordgo.Resumed) {
	m.handleEvent(EventResumed, s.ShardID, "")
}
