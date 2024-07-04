package api

const (
	ServedByReportsServerAnnotation = "kyverno.reports-server.io/served-by"
	ServedByReportsServerValue      = "reports-server"
)

func labelReports(obj map[string]string) map[string]string {
	if obj == nil {
		obj = make(map[string]string)
	}
	obj[ServedByReportsServerAnnotation] = ServedByReportsServerValue
	return obj
}
