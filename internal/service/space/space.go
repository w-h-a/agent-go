package space

type Space struct {
	name string
	id   string
}

func (s *Space) ID() string {
	return s.id
}

func (s *Space) Name() string {
	return s.name
}
