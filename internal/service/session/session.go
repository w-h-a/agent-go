package session

type Session struct {
	id      string
	spaceId string
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) SpaceId() string {
	return s.spaceId
}
