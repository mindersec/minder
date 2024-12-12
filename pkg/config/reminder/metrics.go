package reminder

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled" default:"true"`
	Host    string `mapstructure:"host" default:"127.0.0.1"`
	Port    int    `mapstructure:"port" default:"8080"`
}
