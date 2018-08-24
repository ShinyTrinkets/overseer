package overseer

type Logger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
}

type Attrs map[string]interface{}

var log Logger

// func SetupLogger(l Logger) {
// 	log = Logging(l)
// }
