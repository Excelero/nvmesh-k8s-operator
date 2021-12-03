package controllers

//OperatorOptions - Options to control the global behavior of the operator
type OperatorOptions struct {
	IsOpenShift         bool
	DefaultCoreImageTag string
	Development         bool
}
