package controllers

//DebugOptions - Operator Debug Options
type DebugOptions struct {
	CollectLogsJobsRunForever bool
	ImagePullPolicyAlways     bool
	DebugJobs                 bool
}

//OperatorOptions - Options to control the global behavior of the operator
type OperatorOptions struct {
	Debug       DebugOptions
	IsOpenShift bool
}
