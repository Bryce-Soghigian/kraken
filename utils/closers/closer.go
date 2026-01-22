package closers

import (
	"io"

	"github.com/uber/kraken/utils/log"
	"go.uber.org/zap"
)

// Close closes the closer. A message will be logged.
// The main reason for the helper existence is to have a utility for defer io.Closer() statements.
func Close(closer io.Closer) {
	if closer != nil {
		err := closer.Close()
		if err != nil {
			// Check if this is a "file already closed" error and handle it gracefully
			errMsg := err.Error()
			if errMsg == "file already closed" || errMsg == "close: file already closed" {
				log.Desugar().Debug(
					"attempted to close already closed file (this is usually harmless)",
					zap.Error(err),
				)
			} else {
				log.Desugar().Error(
					"failed to close a closer",
					zap.Error(err),
					zap.Stack("stack"),
				)
			}
		}
	}
}
