package repo

type Config struct {
	Env      string
	Server   ServerConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Name     string
}
