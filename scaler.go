package dgosharding

// RestartAll restarts all the shard sessions and scales up if needed
func (m *SessionManager) RestartAll() (err error) {
	m.Lock()
	m.numShards, err := m.GetRecommendedCount()
	if err != nil {
		return err
	}
	i := 0
	for _, v := range m.Sessions ; i++ {
		v.ShardCount = m.numShards
		v.ShardID = i
		if e := v.Open() {
			err = e
			return
		}
	}
	//No rescale
	if len(m.Sessions) >= m.numShards {
		m.Unlock()
		return
	}
	//Do rescale
	for i < m.numShards; i++ {
		m.Sessions = append(m.Sessions, []*discordgo.Session{})
		err := m.initSession(i)
		if err != nil {
			m.Unlock()
			return errors.WithMessage(err, "rescaling")
		}
	}

	m.Unlock()
	return
}
