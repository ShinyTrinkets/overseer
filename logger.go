package overseer

type Logger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
}

type Attrs map[string]interface{}

func NewLogger(name string) Logger {
	return Logger
}
