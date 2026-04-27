package main

import "fmt"

type Config struct {
	PerPage     int
	MaxPages    int
	Query       string
	DownloadDir string
	ResultDir   string
	Analyzers   []string
}

func NewConfig() *Config {
	return &Config{
		PerPage:     100,
		MaxPages:    1,
		Query:       "stars:>1000",
		DownloadDir: "/tmp/pinned",
		ResultDir:   "results",
		Analyzers:   []string{"pinned"},
	}
}

func (c Config) String() string {
	s := ""
	s += "PerPage: " + fmt.Sprint(c.PerPage) + "\n"
	s += "MaxPages: " + fmt.Sprint(c.MaxPages) + "\n"
	s += "Query: " + c.Query + "\n"
	s += "DownloadDir: " + c.DownloadDir + "\n"
	s += "ResultDir: " + c.ResultDir + "\n"
	s += "Analyzers: " + fmt.Sprint(c.Analyzers) + "\n"
	return s
}
