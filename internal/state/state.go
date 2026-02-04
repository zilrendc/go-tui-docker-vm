package state

type State string

const (
	StateInit           State = "INIT"
	StateLoadConfig     State = "LOAD_CONFIG"
	StateVMSelector     State = "VM_SELECTOR"
	StateConnectingVM   State = "CONNECTING_VM"
	StateContainerList  State = "CONTAINER_LIST"
	StateVMShell        State = "VM_SHELL"
	StateContainerShell State = "CONTAINER_SHELL"
	StateError          State = "ERROR"
)

type StateMachine struct {
	Current State
	Err     error
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		Current: StateInit,
	}
}

func (sm *StateMachine) Transition(next State) {
	sm.Current = next
}

func (sm *StateMachine) SetError(err error) {
	sm.Err = err
	sm.Current = StateError
}
