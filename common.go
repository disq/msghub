package msghub

type Logger interface {
	Printf(string, ...interface{})
}
