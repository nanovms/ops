package crossbuild

var (
	// Source code configuration file name.
	sourceConfigFileName = "source.json"
)

// Source code configuration.
type source struct {
	Commands     sourceCommands     `json:"commands"`
	Dependencies []sourceDependency `json:"dependencies"`
}

// Source code dependency configuration.
type sourceDependency struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Command string `json:"command"`
	AsAdmin bool   `json:"as_admin"`
}

// Source code command configurations.
type sourceCommands struct {
	Run   string `json:"run"`
	Build string `json:"build"`
}
