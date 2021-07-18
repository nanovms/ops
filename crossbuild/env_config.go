package crossbuild

// EnvironmentConfig is configurable settings for an environment.
type EnvironmentConfig struct {
	ID     string `json:"id"`
	Port   int    `json:"port"`
	Memory string `json:"memory"`
}
