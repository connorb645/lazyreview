package languages

type header struct {
	open         int
	end          int
	inherits     []string
	parameters   []string
	parametersAt int
}
