package config

var darkColours = namedColours{
	roles: []namedValue{
		{"defaultFgColor", "#f1f0fb"}, {"mutedFgColor", "#a09dbf"}, {"faintFgColor", "#6d6a8a"}, {"accentColor", "#a796ff"}, {"activeBorderColor", "#a796ff"},
		{"inactiveBorderColor", "#4a4766"}, {"addedColor", "#7ce2a7"}, {"removedColor", "#ff9fb6"}, {"dimAddedColor", "#4d9a74"}, {"dimRemovedColor", "#b0647a"},
		{"warningColor", "#f2b26a"}, {"selectedLineBgColor", "#3a3270"}, {"inactiveViewSelectedLineBgColor", "#26233d"}, {"visualSelectionBgColor", "#45475a"},
	},
	pulls: []string{"#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"},
	components: []namedValue{
		{"blue", "#7fa8f8"}, {"teal", "#4cd3c2"}, {"purple", "#b893fb"}, {"amber", "#f2b26a"}, {"olive", "#b8cc6a"}, {"rose", "#f08cb6"}, {"indigo", "#a4a9ff"},
		{"slate", "#a3afc8"}, {"grey", "#9d9bb8"},
	},
}

var lightColours = namedColours{
	roles: []namedValue{
		{"defaultFgColor", "#17152f"}, {"mutedFgColor", "#615e7f"}, {"faintFgColor", "#9b98b5"}, {"accentColor", "#5233e6"}, {"activeBorderColor", "#5233e6"},
		{"inactiveBorderColor", "#c4c0dc"}, {"addedColor", "#146b3a"}, {"removedColor", "#b21f43"}, {"dimAddedColor", "#5c9a78"}, {"dimRemovedColor", "#c8768a"},
		{"warningColor", "#9c5212"}, {"selectedLineBgColor", "#dcd4ff"}, {"inactiveViewSelectedLineBgColor", "#efebfb"}, {"visualSelectionBgColor", "#c9cbd6"},
	},
	pulls: []string{"#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6", "#e34948"},
	components: []namedValue{
		{"blue", "#2c66a8"}, {"teal", "#0b6e6e"}, {"purple", "#7448ad"}, {"amber", "#9c5212"}, {"olive", "#5f6e12"}, {"rose", "#a8325e"}, {"indigo", "#4a4fb0"},
		{"slate", "#556170"}, {"grey", "#6f7987"},
	},
}
