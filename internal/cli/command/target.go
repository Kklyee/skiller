package command

import (
	"github.com/Kklyee/skiller/internal/environment"
	"github.com/Kklyee/skiller/internal/paths"
)

func recordEnvironmentTarget(pathSet paths.Set, target environment.Target) error {
	return environment.New(pathSet.StatePath()).Save(target)
}

func clearEnvironmentTarget(pathSet paths.Set, kind environment.Kind, name, path string) error {
	store := environment.New(pathSet.StatePath())
	current, ok, err := store.Load()
	if err != nil || !ok {
		return err
	}
	if current.Kind != kind || current.Name != name || (kind == environment.KindProject && current.Path != path) {
		return nil
	}
	return store.Clear()
}
