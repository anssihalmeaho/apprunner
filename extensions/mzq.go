package extensions

import (
	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/mzq/bro"
	"github.com/anssihalmeaho/mzq/msg"
	"github.com/anssihalmeaho/mzq/queue"
)

func init() {
	funl.AddExtensionInitializer(msg.InitMsg)
	funl.AddExtensionInitializer(queue.InitQueue)
	funl.AddExtensionInitializer(bro.InitBro)
}
