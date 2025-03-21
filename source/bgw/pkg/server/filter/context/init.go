package context

import "bgw/pkg/server/filter"

func Init() {
	filter.Register(filter.ContextFilterKey, new)
	filter.Register(filter.ContextFilterKeyGlobal, newGlobalFilter())
}
