package libinsights

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var COLLECTORS_DIRECTORY string = "."

type collectorMeta struct {
	ID          string `toml:"id"`
	Name        string `toml:"name"`
	ContentType string `toml:"content_type"`
}

type collectorSystemd struct {
	Service string `toml:"service"`
	Timer   string `toml:"timer"`
}

type Collector struct {
	Meta    collectorMeta    `toml:"meta"`
	Systemd collectorSystemd `toml:"systemd"`
}

func LoadCollector(path string) (*Collector, error) {
	path, _ = filepath.Abs(path)
	log.Printf("Loading '%s'.\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCollector(string(data))
}

// ParseCollector parses the content of the configuration file into Collector.
func ParseCollector(config string) (*Collector, error) {
	var cc Collector
	_, err := toml.Decode(config, &cc)
	if err == nil {
		log.Printf("Parsed '%s'.\n", cc.Meta.ID)
	}
	return &cc, err
}

func ensureCollectorsDirectory() error {
	if _, err := os.Stat(COLLECTORS_DIRECTORY); os.IsNotExist(err) {
		return NewError(
			ErrParsing, &err,
			"Collectors directory '{directory}' does not exist.",
			&map[string]string{"directory": COLLECTORS_DIRECTORY},
		)
	}
	return nil
}

func GetCollector(id string) (*Collector, error) {
	if err := ensureCollectorsDirectory(); err != nil {
		return nil, err
	}
	return LoadCollector(filepath.Join(COLLECTORS_DIRECTORY, id+".toml"))
}

// GetCollectors loads collector definitions from COLLECTORS_DIRECTORY.
func GetCollectors() ([]*Collector, error) {
	if err := ensureCollectorsDirectory(); err != nil {
		return nil, err
	}

	configurations, err := filepath.Glob(filepath.Join(COLLECTORS_DIRECTORY, "*.toml"))
	if err != nil {
		return nil, NewError(
			ErrParsing, &err,
			"Cannot scan {directory}",
			&map[string]string{"directory": COLLECTORS_DIRECTORY},
		)
	}

	var collectors []*Collector
	for _, file := range configurations {
		config, err := LoadCollector(file)
		if err != nil {
			log.Printf("Collector is malformed: %v\n", err)
			continue
		}
		collectors = append(collectors, config)
	}
	return collectors, nil
}
