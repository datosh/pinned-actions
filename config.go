package main

import "fmt"

type Config struct {
	PerPage     int
	MaxPages    int
	Query       string
	DownloadDir string
	ResultFile  string
}

func NewConfig() *Config {
	return &Config{
		PerPage:     100,
		MaxPages:    1,
		Query:       "stars:>1000",
		DownloadDir: "/tmp/pinned",
		ResultFile:  "result.json",
	}
}

func (c Config) String() string {
	s := ""
	s += "PerPage: " + fmt.Sprint(c.PerPage) + "\n"
	s += "MaxPages: " + fmt.Sprint(c.MaxPages) + "\n"
	s += "Query: " + c.Query + "\n"
	s += "DownloadDir: " + c.DownloadDir + "\n"
	s += "ResultFile: " + c.ResultFile + "\n"
	return s
}
