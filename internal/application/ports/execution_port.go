package ports

type ExecutionPort interface {
	GetIsSyncing() (bool, error)
}
