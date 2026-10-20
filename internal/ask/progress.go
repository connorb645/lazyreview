package ask

import (
	"sync"
	"time"

	"github.com/connorb645/lazyreview/internal/commands/claude"
	"github.com/connorb645/lazyreview/internal/models"
)

type progress struct {
	mu      sync.Mutex
	steps   []stepHeard
	words   func(claude.Heard) string
	write   func([]models.Step)
	changed chan struct{}
	over    chan struct{}
	done    chan struct{}
}

func newProgress(words func(claude.Heard) string, write func([]models.Step)) *progress {
	p := &progress{words: words, write: write, changed: make(chan struct{}, 1), over: make(chan struct{}), done: make(chan struct{})}
	go p.writing()
	return p
}

func (p *progress) heard(h claude.Heard) {
	p.mu.Lock()
	if h.Refused != "" {
		for i := range p.steps {
			if p.steps[i].id == h.ID {
				p.steps[i].refused = h.Refused
			}
		}
	} else {
		p.steps = append(p.steps, stepHeard{id: h.ID, words: p.words(h)})
	}
	p.mu.Unlock()
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

func (p *progress) writing() {
	defer close(p.done)
	for {
		select {
		case <-p.over:
			return
		case <-p.changed:
		}
		select {
		case <-p.over:
			return
		case <-time.After(writeEvery):
		}
		p.write(p.kept())
	}
}

func (p *progress) settled() {
	close(p.over)
	<-p.done
}

func (p *progress) kept() []models.Step {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := []models.Step{}
	for _, s := range p.steps[max(0, len(p.steps)-keptSteps):] {
		out = append(out, models.Step{Words: s.words, Refused: s.refused})
	}
	return out
}
