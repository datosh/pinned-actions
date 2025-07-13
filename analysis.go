package main

import "fmt"

type Analysis struct {
	Repository    string `json:"repository"`
	ActionsPinned int    `json:"actions_pinned"`
	ActionsTotal  int    `json:"actions_total"`
	HasRenovate   bool   `json:"has_renovate"`
	HasDependabot bool   `json:"has_dependabot"`
}

func NewAnalysis(repository string) Analysis {
	return Analysis{
		Repository:    repository,
		ActionsPinned: 0,
		ActionsTotal:  0,
		HasRenovate:   false,
		HasDependabot: false,
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
	updater := "None"
	if a.HasRenovate {
		updater = "Renovate"
	}
	if a.HasDependabot {
		updater = "Dependabot"
	}
	return fmt.Sprintf("%s: %d/%d (%s)", a.Repository, a.ActionsPinned, a.ActionsTotal, updater)
}
