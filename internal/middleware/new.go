package middleware

import (
	"autonomous-task-management/config"
	"autonomous-task-management/pkg/encrypter"
	"autonomous-task-management/pkg/log"
	"autonomous-task-management/pkg/scope"
)

type Middleware struct {
	l            log.Logger
	jwtManager   scope.Manager
	cookieConfig config.CookieConfig
	internalKey  string
	config       *config.Config
	encrypter    encrypter.Encrypter
}

func New(l log.Logger, jwtManager scope.Manager, cookieConfig config.CookieConfig, internalKey string, cfg *config.Config, enc encrypter.Encrypter) Middleware {
	return Middleware{
		l:            l,
		jwtManager:   jwtManager,
		cookieConfig: cookieConfig,
		internalKey:  internalKey,
		config:       cfg,
		encrypter:    enc,
	}
}
