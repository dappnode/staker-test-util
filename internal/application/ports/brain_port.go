package ports

type BrainPort interface {
	GetValidatorsPubkeys() ([]string, error)
}
