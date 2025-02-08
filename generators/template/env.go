package template

import (
	"fmt"
	"strings"
)

func GetEnv(args []string) string {
	upperCaseName := strings.ToUpper(args[0])
	return fmt.Sprintf(`SERVICE_NAME=%s
		SERVICE_STATUS=LOCAl
		SERVICE_ID=medfri-%s`, upperCaseName, args[0])
}
