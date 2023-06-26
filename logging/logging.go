package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	cli "github.com/peterebden/go-cli-init/v5/logging"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("puku")

func InitLogging(verbosity cli.Verbosity) {
	be := logging.AddModuleLevel(logging.NewBackendFormatter(logging.NewLogBackend(os.Stderr, "", 0), formatter{}))
	be.SetLevel(logging.Level(verbosity), "puku")
	log.SetBackend(be)
}

func GetLogger() *logging.Logger {
	return log
}

type formatter struct {
}

func (formatter) Format(_ int, r *logging.Record, w io.Writer) error {
	_, err := fmt.Fprintf(w, "%v %v %v", time.Now().Format("15:04:05"), r.Level.String(), r.Message())
	return err
}
