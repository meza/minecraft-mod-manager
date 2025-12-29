package init

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

func (prompter terminalPrompter) ConfirmOverwrite(configPath string) (bool, error) {
	if _, err := fmt.Fprint(prompter.out, i18n.T("cmd.init.prompt.config-overwrite.question", &i18n.Tvars{
		Data: &i18n.TData{"configPath": configPath},
	})); err != nil {
		return false, err
	}
	answer, err := readLine(prompter.in)
	if err != nil {
		return false, err
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func (prompter terminalPrompter) RequestNewConfigPath(configPath string) (string, error) {
	if _, err := fmt.Fprint(prompter.out, i18n.T("cmd.init.prompt.config-path.question", &i18n.Tvars{
		Data: &i18n.TData{"configPath": configPath},
	})); err != nil {
		return "", err
	}
	answer, err := readLine(prompter.in)
	if err != nil {
		return "", err
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", errors.New(i18n.T("cmd.init.error.config-path.empty", nil))
	}
	return answer, nil
}

func readLine(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}
