package controllers

type DebugOptions struct {
	CollectLogsJobsRunForever bool
	ImagePullPolicyAlways     bool
	DebugJobs                 bool
}

type OperatorOptions struct {
	Debug       DebugOptions
	IsOpenShift bool
}
