package ruletypes

type RuleTypeService interface {
	Create() error
	Update() error
	Delete() error
}
