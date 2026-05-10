package agent

type Config struct {
	ServerAddr       string
	Secret           string
	AgentName        string
	AgentProcessName string
	RealmProcessName string
	ServiceName      string
	InstanceName     string
	InstallDir       string
	ConfigDir        string
	LogDir           string
	DataDir          string
	PidFile          string
	Mode             string
	RealmBinaryPath  string
	Foreground       bool
	PrintVersion     bool
}
