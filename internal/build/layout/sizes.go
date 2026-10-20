package layout

import (
	"math"
	"strings"

	"github.com/connorb645/lazyreview/internal/models"
)

const (
	packageHead         = 46
	packageHeadWithNote = 64
	componentHead       = 104
	areaHead            = 200
	farTitleSize        = 150
	farTitleOverhang    = 200
)

var charWidths = map[string]float64{
	"name": 7.9, "fileName": 7.4, "file": 6.5, "stereotype": 6.0, "edge": 6.2,
	"packageTitle": 8.6, "packagePath": 7.2, "packageNote": 6.6, "groupTitle": 7.6, "groupNote": 6.5,
	"packageArrow": 8.2, "componentTitle": 20, "componentOpen": 8.6, "componentTag": 9.1,
	"summary": 13.6, "areaTitle": 37, "areaTag": 13.3,
}

func measure(text, style string) float64 {
	return math.Ceil(float64(utf16Length(text)) * charWidths[style])
}

func utf16Length(text string) int {
	n := 0
	for _, r := range text {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

func farTitleRoom(title string) float64 {
	return math.Ceil(float64(utf16Length(title))*0.62*farTitleSize) - farTitleOverhang*2
}

func nodeSize(n models.Node) (float64, float64) {
	artifact := n.Kind == "artifact"
	style, file := "name", 0.0
	if artifact {
		style = "fileName"
	} else {
		file = measure(n.File[strings.LastIndex(n.File, "/")+1:], "file")
	}
	widest := max(measure(n.Name, style), file, measure(n.Stereotype, "stereotype"))
	w := max(120, math.Ceil(widest+28))
	if artifact {
		return w, 46
	}
	return w, 62
}
