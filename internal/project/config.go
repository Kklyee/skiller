package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const ConfigFileName = ".skiller.toml"

type Config struct {
	Profile string   `toml:"profile"`
	Skills  []string `toml:"skills"`
}

func Find(start string) (string, error) {
	if start == "" {
		start = "."
	}

	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve project path %q: %w", start, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect project path %q: %w", start, err)
	}
	if !info.IsDir() {
		absolute = filepath.Dir(absolute)
	}

	for {
		path := filepath.Join(absolute, ConfigFileName)
		info, err := os.Stat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("project config %q is not a regular file", path)
			}
			return path, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect project config %q: %w", path, err)
		}

		parent := filepath.Dir(absolute)
		if parent == absolute {
			break
		}
		absolute = parent
	}

	return "", fmt.Errorf("project config %q not found from %q", ConfigFileName, start)
}

func Load(dir string) (Config, string, error) {
	path, err := Find(dir)
	if err != nil {
		return Config{}, "", err
	}
	config, err := LoadFile(path)
	if err != nil {
		return Config{}, "", err
	}
	return config, path, nil
}

func LoadFile(path string) (Config, error) {
	var config Config
	metadata, err := toml.DecodeFile(path, &config)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("project config %q does not exist", path)
		}
		return Config{}, fmt.Errorf("read project config %q: %w", path, err)
	}

	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		return Config{}, fmt.Errorf("project config %q has unsupported field %q", path, undecoded[0])
	}
	if err := validate(config, metadata); err != nil {
		return Config{}, fmt.Errorf("validate project config %q: %w", path, err)
	}

	return config, nil
}

func validate(config Config, metadata toml.MetaData) error {
	hasProfile := metadata.IsDefined("profile")
	hasSkills := metadata.IsDefined("skills")
	if hasProfile == hasSkills {
		return errors.New("define exactly one of profile or skills")
	}

	if hasProfile {
		if strings.TrimSpace(config.Profile) == "" {
			return errors.New("profile must not be empty")
		}
		return nil
	}

	seen := make(map[string]struct{}, len(config.Skills))
	for _, skill := range config.Skills {
		if strings.TrimSpace(skill) == "" {
			return errors.New("skill ID must not be empty")
		}
		if _, ok := seen[skill]; ok {
			return fmt.Errorf("skill %q is listed more than once", skill)
		}
		seen[skill] = struct{}{}
	}
	return nil
}
