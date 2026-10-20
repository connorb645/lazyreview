package ask

import "github.com/connorb645/lazyreview/terminal/internal/models"

// Answerer has Claude answer the last question of a conversation, and
// returns once the answer, or why there is none, is kept.
type Answerer func(ref models.ReviewRef, conversation int) error
