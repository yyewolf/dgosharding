package dgosharding

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

//SessionFunc represents the session used by the manager.
type SessionFunc func(token string) (*discordgo.Session, error)

//SessionManager represent a sharding manager
type SessionManager struct {
	sync.RWMutex

	// Name of the bot, to appear before log messages as a prefix
	// and in the title of the updated status message
	Name string

	// All the shard sessions
	Sessions      []*discordgo.Session
	eventHandlers []interface{}

	// If set logs connection status events to this channel
	LogChannel string

	// If set keeps an updated satus message in this channel
	StatusMessageChannel string

	// The function that provides the guild counts per shard, used fro the updated status message
	// Should return a slice of guild counts, with the index being the shard number
	GuildCountsFunc func() []int

	// Called on events, by default this is set to a function that logs it to log.Printf
	// You can override this if you want another behaviour, or just set it to nil for nothing.
	OnEvent func(e *Event)

	// SessionFunc creates a new session and returns it, override the default one if you have your own
	// session settings to apply
	SessionFunc SessionFunc

	nextStatusUpdate     time.Time
	statusUpdaterStarted bool

	numShards int
	token     string

	bareSession *discordgo.Session
	started     bool
}

// New creates a new shard manager with the defaults set, after you have created this you call Manager.Start
// To start connecting
// dshardmanager.New("Bot asd", OptLogChannel(someChannel), OptLogEventsToDiscord(true, true))
func New(token string) *SessionManager {
	// Setup defaults
	manager := &SessionManager{
		token:     token,
		numShards: -1,
	}

	manager.OnEvent = manager.LogConnectionEventStd
	manager.SessionFunc = manager.StdSessionFunc

	manager.bareSession, _ = discordgo.New(token)

	return manager
}

// GetRecommendedCount gets the recommended sharding count from discord, this will also
// set the shard count internally if called
// Should not be called after calling Start(), will have undefined behaviour
func (m *SessionManager) GetRecommendedCount() (int, error) {
	resp, err := m.bareSession.GatewayBot()
	if err != nil {
		return 0, errors.WithMessage(err, "GetRecommendedCount()")
	}

	m.numShards = resp.Shards
	if m.numShards < 1 {
		m.numShards = 1
	}

	return m.numShards, nil
}

// GetNumShards returns the current set number of shards
func (m *SessionManager) GetNumShards() int {
	return m.numShards
}

// SetNumShards sets the number of shards to use, if you want to override the recommended count
// Should not be called after calling Start(), will panic
func (m *SessionManager) SetNumShards(n int) {
	m.Lock()
	defer m.Unlock()
	if m.started {
		panic("Can't set num shard after started")
	}

	m.numShards = n
}

// Init initializesthe manager, retreiving the recommended shard count if needed
// and initalizes all the shards
func (m *SessionManager) Init() error {
	m.Lock()
	if m.numShards < 1 {
		_, err := m.GetRecommendedCount()
		if err != nil {
			return errors.WithMessage(err, "Start")
		}
	}

	m.Sessions = make([]*discordgo.Session, m.numShards)
	for i := 0; i < m.numShards; i++ {
		err := m.initSession(i)
		if err != nil {
			m.Unlock()
			return errors.WithMessage(err, "initSession")
		}
	}

	if !m.statusUpdaterStarted {
		m.statusUpdaterStarted = true
		go m.statusRoutine()
	}

	m.nextStatusUpdate = time.Now()

	m.Unlock()

	return nil
}

// Start starts the shard manager, opening all gateway connections
func (m *SessionManager) Start() error {

	m.Lock()
	if m.Sessions == nil {
		m.Unlock()
		err := m.Init()
		if err != nil {
			return err
		}
		m.Lock()
	}

	m.Unlock()

	for i := 0; i < m.numShards; i++ {
		if i != 0 {
			// One indentify every 5 seconds
			time.Sleep(time.Second * 5)
		}

		m.Lock()
		err := m.startSession(i)
		m.Unlock()
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("Failed starting shard %d", i))
		}
	}

	return nil
}

// StopAll stops all the shard sessions and returns the last error that occured
func (m *SessionManager) StopAll() (err error) {
	m.Lock()
	for _, v := range m.Sessions {
		if e := v.Close(); e != nil {
			err = e
		}
	}
	m.Unlock()

	return
}

func (m *SessionManager) initSession(shard int) error {
	session, err := m.SessionFunc(m.token)
	if err != nil {
		return errors.WithMessage(err, "startSession.SessionFunc")
	}

	session.ShardCount = m.numShards
	session.ShardID = shard

	session.AddHandler(m.OnDiscordConnected)
	session.AddHandler(m.OnDiscordDisconnected)
	session.AddHandler(m.OnDiscordReady)
	session.AddHandler(m.OnDiscordResumed)

	// Add the user event handlers retroactively
	for _, v := range m.eventHandlers {
		session.AddHandler(v)
	}

	m.Sessions[shard] = session
	return nil
}

func (m *SessionManager) startSession(shard int) error {

	err := m.Sessions[shard].Open()
	if err != nil {
		return errors.Wrap(err, "startSession.Open")
	}
	m.handleEvent(EventOpen, shard, "")

	return nil
}

// LogConnectionEventStd is the standard connection event logger, it logs it to whatever log.output is set to.
func (m *SessionManager) LogConnectionEventStd(e *Event) {
	log.Printf("[Shard Manager] %s", e.String())
}

func (m *SessionManager) handleError(err error, shard int, msg string) bool {
	if err == nil {
		return false
	}

	m.handleEvent(EventError, shard, msg+": "+err.Error())
	return true
}

func (m *SessionManager) handleEvent(typ EventType, shard int, msg string) {
	if m.OnEvent == nil {
		return
	}

	evt := &Event{
		Type:      typ,
		Shard:     shard,
		NumShards: m.numShards,
		Msg:       msg,
		Time:      time.Now(),
	}

	go m.OnEvent(evt)

	if m.LogChannel != "" {
		go m.logEventToDiscord(evt)
	}

	go func() {
		m.Lock()
		m.nextStatusUpdate = time.Now().Add(time.Second * 2)
		m.Unlock()
	}()
}

// StdSessionFunc is the standard session provider, it does nothing to the actual session
func (m *SessionManager) StdSessionFunc(token string) (*discordgo.Session, error) {
	s, err := discordgo.New(token)
	if err != nil {
		return nil, errors.WithMessage(err, "StdSessionFunc")
	}
	return s, nil
}

func (m *SessionManager) logEventToDiscord(evt *Event) {
	if evt.Type == EventError {
		return
	}

	prefix := ""
	if m.Name != "" {
		prefix = m.Name + ": "
	}

	str := evt.String()
	embed := &discordgo.MessageEmbed{
		Description: prefix + str,
		Timestamp:   evt.Time.Format(time.RFC3339),
		Color:       eventColors[evt.Type],
	}

	_, err := m.bareSession.ChannelMessageSendEmbed(m.LogChannel, embed)
	m.handleError(err, evt.Shard, "Failed sending event to discord")
}

func (m *SessionManager) statusRoutine() {
	if m.StatusMessageChannel == "" {
		return
	}

	mID := ""

	// Find the initial message id and reuse that message if found
	msgs, err := m.bareSession.ChannelMessages(m.StatusMessageChannel, 50, "", "", "")
	if err != nil {
		m.handleError(err, -1, "Failed requesting message history in channel")
	} else {
		for _, msg := range msgs {
			// Dunno our own bot id so best we can do is bot
			if !msg.Author.Bot || len(msg.Embeds) < 1 {
				continue
			}

			nameStr := ""
			if m.Name != "" {
				nameStr = " for " + m.Name
			}

			embed := msg.Embeds[0]
			if embed.Title == "Sharding status"+nameStr {
				// Found it sucessfully
				mID = msg.ID
				break
			}
		}
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			m.RLock()
			after := time.Now().After(m.nextStatusUpdate)
			m.RUnlock()
			if after {
				m.Lock()
				m.nextStatusUpdate = time.Now().Add(time.Minute)
				m.Unlock()

				nID, err := m.updateStatusMessage(mID)
				if !m.handleError(err, -1, "Failed updating status message") {
					mID = nID
				}
			}
		}
	}
}

func (m *SessionManager) updateStatusMessage(mID string) (string, error) {
	content := ""

	status := m.GetFullStatus()
	for _, shard := range status.Shards {
		emoji := ""
		if !shard.Started {
			emoji = "🕒"
		} else if shard.OK {
			emoji = "👌"
		} else {
			emoji = "🔥"
		}
		content += fmt.Sprintf("[%d/%d]: %s (%d,%d)\n", shard.Shard, m.numShards, emoji, shard.NumGuilds, status.NumGuilds)
	}

	nameStr := ""
	if m.Name != "" {
		nameStr = " for " + m.Name
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Sharding status" + nameStr,
		Description: content,
		Color:       0x4286f4,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if mID == "" {
		msg, err := m.bareSession.ChannelMessageSendEmbed(m.StatusMessageChannel, embed)
		if err != nil {
			return "", err
		}

		return msg.ID, err
	}

	_, err := m.bareSession.ChannelMessageEditEmbed(m.StatusMessageChannel, mID, embed)
	return mID, err
}

// GetFullStatus retrieves the full status at this instant
func (m *SessionManager) GetFullStatus() *Status {
	var shardGuilds []int
	if m.GuildCountsFunc != nil {
		shardGuilds = m.GuildCountsFunc()
	} else {
		shardGuilds = m.GuildCount()
	}

	m.RLock()

	result := make([]*ShardStatus, len(m.Sessions))
	for i, shard := range m.Sessions {
		result[i] = &ShardStatus{
			Shard: i,
		}

		if shard != nil {
			result[i].Started = true

			shard.RLock()
			result[i].OK = shard.DataReady
			shard.RUnlock()
		}
	}
	m.RUnlock()

	totalGuilds := 0
	for shard, guilds := range shardGuilds {
		totalGuilds += guilds
		result[shard].NumGuilds = guilds
	}

	return &Status{
		Shards:    result,
		NumGuilds: totalGuilds,
	}
}
