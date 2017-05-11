package executor

type Option func(*executor)

func WithDirSizeFunc(fn DirSizeFunc) Option {
	return func(e *executor) {
		e.dirSize = fn
	}
}

func WithCommandFunc(fn CmdFunc) Option {
	return func(e *executor) {
		e.execCommand = fn
	}
}
