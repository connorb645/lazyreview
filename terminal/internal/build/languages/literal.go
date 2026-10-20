package languages

type literal struct {
	start   int
	end     int
	inner   [2]int
	comment bool
}
