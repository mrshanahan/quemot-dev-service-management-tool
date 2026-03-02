package command

type ServiceCommandSpec struct {
	Args []string
}

type ServiceCommand struct {
}

type ServiceAction int

const (
	ListServices ServiceAction = iota
	StartService
	StopService
	RestartService
	RemoveService
)

func (s *ServiceCommandSpec) Build() (Command, error) {
	return nil, nil
}

func (c *ServiceCommand) Invoke() error {
	return nil
}
