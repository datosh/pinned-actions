package main

type Config struct {
	PerPage  int
	MaxPages int
	// MinStars and MaxStars bound the star range of the first search; later
	// searches move MaxStars down to page through the results.
	MinStars    int
	MaxStars    int
	DownloadDir string
	ResultDir   string
	Analyzers   []string
}

func NewConfig() *Config {
	return &Config{
		PerPage:     100,
		MaxPages:    1,
		MinStars:    1000,
		MaxStars:    500000,
		DownloadDir: "/tmp/pinned",
		ResultDir:   "results",
		Analyzers:   []string{"pinned"},
	}
}
