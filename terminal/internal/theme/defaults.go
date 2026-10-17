package theme

// DarkColours are the colours for a terminal with a dark background.
func DarkColours() Colours {
	return Colours{
		DefaultFgColor:                  "#f1f0fb",
		MutedFgColor:                    "#a09dbf",
		FaintFgColor:                    "#6d6a8a",
		AccentColor:                     "#a796ff",
		ActiveBorderColor:               "#a796ff",
		InactiveBorderColor:             "#4a4766",
		AddedColor:                      "#7ce2a7",
		RemovedColor:                    "#ff9fb6",
		WarningColor:                    "#f2b26a",
		SelectedLineBgColor:             "#3a3270",
		InactiveViewSelectedLineBgColor: "#26233d",
		VisualSelectionBgColor:          "#45475a",
		PullRequestColors:               []string{"#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"},
		ComponentColors:                 map[string]string{"blue": "#7fa8f8", "teal": "#4cd3c2", "purple": "#b893fb", "amber": "#f2b26a", "olive": "#b8cc6a", "rose": "#f08cb6", "indigo": "#a4a9ff", "slate": "#a3afc8", "grey": "#9d9bb8"},
	}
}

// LightColours are the colours for a terminal with a light background.
func LightColours() Colours {
	return Colours{
		DefaultFgColor:                  "#17152f",
		MutedFgColor:                    "#615e7f",
		FaintFgColor:                    "#9b98b5",
		AccentColor:                     "#5233e6",
		ActiveBorderColor:               "#5233e6",
		InactiveBorderColor:             "#c4c0dc",
		AddedColor:                      "#146b3a",
		RemovedColor:                    "#b21f43",
		WarningColor:                    "#9c5212",
		SelectedLineBgColor:             "#dcd4ff",
		InactiveViewSelectedLineBgColor: "#efebfb",
		VisualSelectionBgColor:          "#c9cbd6",
		PullRequestColors:               []string{"#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6", "#e34948"},
		ComponentColors:                 map[string]string{"blue": "#2c66a8", "teal": "#0b6e6e", "purple": "#7448ad", "amber": "#9c5212", "olive": "#5f6e12", "rose": "#a8325e", "indigo": "#4a4fb0", "slate": "#556170", "grey": "#6f7987"},
	}
}
