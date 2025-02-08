package template

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/util"
)

func GetEntity(args []string) string {

	return fmt.Sprintf(`package entity

func (%s) TableName() string {
	return "%s"
}

type %s struct {}`,
		util.CapitalizeFirst(args[0]), args[0], util.CapitalizeFirst(args[0]))
}
