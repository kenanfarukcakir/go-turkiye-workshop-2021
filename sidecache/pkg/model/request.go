package model

import "strings"

type PurgeRequest struct {
	Url string `json:"url"`
}

func (pr *PurgeRequest) EnsureHasSlashPrefix() {
	if !strings.HasPrefix(pr.Url, "/") {
		pr.Url = "/" + pr.Url
	}
}
