package console

import (
	"strings"

	"github.com/chzyer/readline"
)

const (
	Yes = "y"
	No  = "n"
)

var yesNoConstraints = []string{"y", "n"}

func YesOrNo(question string) (string, error) {
	return Prompt(question, yesNoConstraints...)
}

func Prompt(question string, constraints ...string) (string, error) {
	if len(constraints) == 0 {
		rl, err := readline.New(question)
		if err != nil {
			return "", err
		}
		return rl.Readline()
	}
	def := strings.ToUpper(constraints[0])
	var prompt strings.Builder
	prompt.WriteString(question)
	prompt.WriteString(" [")
	prompt.WriteString(def)
	for i := 1; i < len(constraints); i++ {
		prompt.WriteString("/")
		prompt.WriteString(constraints[i])
	}
	prompt.WriteString("]:")
	rl, err := readline.New(prompt.String())
	if err != nil {
		return "", err
	}
	response, err := rl.Readline()
	if err != nil {
		return "", err
	}
	// return default on no input
	if response == "" {
		return constraints[0], nil
	}
	normalized := strings.ToLower(response)
	for _, c := range constraints {
		if normalized == c {
			return normalized, nil
		}
	}
	// no constraint matched, return default
	return constraints[0], nil
}
