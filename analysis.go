package main

import "fmt"

type Analysis struct {
	Repository    string `json:"repository"`
	ActionsPinned int    `json:"actions_pinned"`
	ActionsTotal  int    `json:"actions_total"`
}

func NewAnalysis(repository string) Analysis {
	return Analysis{
		Repository:    repository,
		ActionsPinned: 0,
		ActionsTotal:  0,
	}
}

func (a *Analysis) CountPinned() {
	a.ActionsPinned++
	a.ActionsTotal++
}

func (a *Analysis) CountUnpinned() {
	a.ActionsTotal++
}

func (a Analysis) String() string {
	return fmt.Sprintf("%s: %d/%d", a.Repository, a.ActionsPinned, a.ActionsTotal)
}
