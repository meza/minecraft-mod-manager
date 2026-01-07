package install

import "github.com/meza/minecraft-mod-manager/internal/httpclient"

type installItemProgressMsg struct {
	key      string
	progress httpclient.DownloadProgressMsg
}

type installItemProgressErrMsg struct {
	key string
	err error
}

type installItemSuccessMsg struct {
	key         string
	displayName string
}

type installItemFailureMsg struct {
	key    string
	reason string
}

type installItemAbortedMsg struct {
	key string
}

type installExecutionFinishedMsg struct {
	outcome installExecutionOutcome
}
