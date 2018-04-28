package msghub

// Logger is a basic logger interface.
type Logger interface {
	Printf(string, ...interface{})
}
