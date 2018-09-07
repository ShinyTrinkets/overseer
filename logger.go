package overseer

// Logger types must support Info and Error functions
type Logger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
}

// Attrs are used for providing additional info in the log messages
type Attrs map[string]interface{}

// LogBuilderType function builds new log instances
type LogBuilderType func(name string) Logger

// NewLogger builder function
var NewLogger LogBuilderType

// SetupLogBuilder helper is used for setting up the log builder function
func SetupLogBuilder(b LogBuilderType) {
	NewLogger = b
}
