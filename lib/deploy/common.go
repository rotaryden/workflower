package deploy

import "fmt"

const SERVICE_PREFIX = "aiwf_"

func getServiceName(appName string) string {
	return fmt.Sprintf("%s%s.service", SERVICE_PREFIX, appName)
}
