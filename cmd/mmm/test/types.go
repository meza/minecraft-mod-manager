package test

import "github.com/meza/minecraft-mod-manager/internal/models"

type testItemStatus int

const (
	testItemStatusChecking testItemStatus = iota
	testItemStatusSupported
	testItemStatusUnsupported
	testItemStatusInconclusive
)

type testItem struct {
	Mod    models.Mod
	Status testItemStatus
	Reason string
}

type testExecutionOutcome struct {
	items []testItem
	err   error
}

type testExecutionInput struct {
	cfg           models.ModsJSON
	targetVersion string
	items         []testItem
	indexByKey    map[string]int
	deps          testDeps
}
