package dgosharding

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
