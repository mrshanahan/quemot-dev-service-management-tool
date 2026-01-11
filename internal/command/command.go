package command

type CommandSpec interface {
	Build() (Command, error)
}

type Command interface {
	Invoke() error
}

type EmptyWriter struct{}

func (w *EmptyWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
