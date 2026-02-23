package impl

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
	Pid() int
	Cleanup()
	Kill()
	Open() (string, error)
}
