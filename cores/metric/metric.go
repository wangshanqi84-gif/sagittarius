package metric

type IMetric interface {
	Start()
	Reports() chan string
}
