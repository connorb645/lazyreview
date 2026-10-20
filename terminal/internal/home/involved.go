package home

type involved struct {
	asked     []string
	yours     bool
	commented bool
}

func (a involved) joined(b involved) involved {
	asked := append([]string{}, a.asked...)
	for _, name := range b.asked {
		held := false
		for _, have := range asked {
			held = held || have == name
		}
		if !held {
			asked = append(asked, name)
		}
	}
	return involved{asked: asked, yours: a.yours || b.yours, commented: a.commented || b.commented}
}
