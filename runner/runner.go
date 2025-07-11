package runner

import (
	"github.com/YASSERRMD/Yolama/runner/llamarunner"
	"github.com/YASSERRMD/Yolama/runner/ollamarunner"
)

func Execute(args []string) error {
	if args[0] == "runner" {
		args = args[1:]
	}

	var newRunner bool
	if args[0] == "--ollama-engine" {
		args = args[1:]
		newRunner = true
	}

	if newRunner {
		return ollamarunner.Execute(args)
	} else {
		return llamarunner.Execute(args)
	}
}
