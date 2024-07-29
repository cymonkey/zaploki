package zaploki

type SinkConfig struct {
	DynamicLabels  []string
	PrintFieldKey  bool
	LoglineBuilder LogLineBuilder
}
