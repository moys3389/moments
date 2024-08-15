package main

import (
	"github.com/kingwrcy/moments/app"
	"github.com/kingwrcy/moments/logger"
	"github.com/samber/do/v2"
)

var gitCommitID string

// @title		Moments API
// @version	0.2.1
func main() {
	if gitCommitID != "" {
		do.MustInvoke[*logger.Logger](nil).Info().Msgf("git commit id = %s", gitCommitID)
	}

	do.MustInvoke[*app.App](nil).Start()
}
