package queue

// Places is each line of a file's patch, by side, RIGHT or LEFT, and line.
type Places map[string]map[int]Place
