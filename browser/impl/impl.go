package impl

import (
	"github.com/go-rod/rod/lib/launcher"
)

type BrowserCursor interface {
	Prepare() error

	Next() bool

	Current() IBrowser

	Err() error

	Close() error
}

type IBrowser interface {
	Id() string
	Name() string
	OnLaunch(*launcher.Launcher) (string, error)
}
