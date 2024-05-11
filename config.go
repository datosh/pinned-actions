package main

type Config struct {
	PerPage     int
	MaxPages    int
	Query       string
	DownloadDir string
	ResultFile  string
}

func NewConfig() *Config {
	return &Config{
		PerPage:     5,
		MaxPages:    1,
		Query:       "stars:>1000",
		DownloadDir: "/tmp/pinned",
		ResultFile:  "result.json",
	}
}
