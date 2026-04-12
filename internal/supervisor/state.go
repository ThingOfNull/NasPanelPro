package supervisor

// Mode 监督器当前模式。
type Mode int

const (
	ModeGUI Mode = iota
	ModeConsole
)
