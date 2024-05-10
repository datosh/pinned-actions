package main

type Config struct {
	PerPage     int
	MaxPages    int
	Query       string
	DownloadDir string
}

func NewConfig() *Config {
	return &Config{
		PerPage:     5,
		MaxPages:    5,
		Query:       "stars:>1000",
		DownloadDir: "/tmp/pinned",
	}
}
