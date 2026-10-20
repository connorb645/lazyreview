package layout

import "github.com/connorb645/lazyreview/terminal/internal/models"

type summaryPair struct {
	fwd   models.ComponentLinks
	rev   *models.ComponentLinks
	lines []string
	ways  []models.ArrowWay
}

func (p summaryPair) arrow() models.ComponentArrow {
	return models.ComponentArrow{Source: p.fwd.Source, Target: p.fwd.Target, Both: p.rev != nil, Lines: p.lines, Ways: p.ways}
}
