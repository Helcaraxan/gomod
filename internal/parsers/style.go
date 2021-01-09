package parsers

import (
	"errors"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/printer"
)

func ParseStyleConfiguration(log *logger.Logger, config string) (*printer.StyleOptions, error) {
	styleOptions := &printer.StyleOptions{}
	for _, setting := range strings.Split(config, ",") {
		if setting == "" {
			continue
		}

		configKey := setting
		var configValue string
		if valueIdx := strings.Index(setting, "="); valueIdx >= 0 {
			configKey = setting[:valueIdx]
			configValue = setting[valueIdx+1:]
		}
		configKey = strings.ToLower(strings.TrimSpace(configKey))
		configValue = strings.ToLower(strings.TrimSpace(configValue))

		var err error
		switch configKey {
		case "scale_nodes":
			err = parseStyleScaleNodes(log, styleOptions, configValue)
		case "cluster":
			err = parseStyleCluster(log, styleOptions, configValue)
		default:
			log.Error("Skipping unknown style option.", zap.String("option", configKey))
			err = errors.New("invalid config")
		}
		if err != nil {
			return nil, err
		}
	}
	return styleOptions, nil
}

func parseStyleScaleNodes(log *logger.Logger, styleOptions *printer.StyleOptions, raw string) error {
	switch strings.ToLower(raw) {
	case "", "true", "on", "yes":
		styleOptions.ScaleNodes = true
	case "false", "off", "no":
		styleOptions.ScaleNodes = false
	default:
		log.Error("Could not set 'scale_nodes' style. Accepted values are 'true' and 'false'.", zap.String("value", raw))
		return errors.New("invalid 'scale_nodes' value")
	}
	return nil
}

func parseStyleCluster(log *logger.Logger, styleOptions *printer.StyleOptions, raw string) error {
	switch strings.ToLower(raw) {
	case "off", "false", "no":
		styleOptions.Cluster = printer.Off
	case "", "shared", "on", "true", "yes":
		styleOptions.Cluster = printer.Shared
	case "full":
		styleOptions.Cluster = printer.Full
	default:
		log.Error("Could not set 'cluster' style. Accepted values are 'off', 'shared' and 'full'.", zap.String("value", raw))
		return errors.New("invalid 'cluster' value")
	}
	return nil
}
