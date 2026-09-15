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
	Include []string `toml:"include"`
	Exclude []string `toml:"exclude"`
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
	} else {
		if err := validateSkills(config.Skills, "skill"); err != nil {
			return err
		}
	}
	if err := validateSkills(config.Include, "include skill"); err != nil {
		return err
	}
	return validateSkills(config.Exclude, "exclude skill")
}

func validateSkills(skills []string, kind string) error {
	seen := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		if strings.TrimSpace(skill) == "" {
			return fmt.Errorf("%s ID must not be empty", kind)
		}
		if _, ok := seen[skill]; ok {
			return fmt.Errorf("%s %q is listed more than once", kind, skill)
		}
		seen[skill] = struct{}{}
	}
	return nil
}
