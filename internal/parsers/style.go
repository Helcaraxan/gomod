package parsers

import (
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/printer"
)

func ParseVisualConfig(logger *logrus.Logger, config string) (*printer.StyleOptions, error) {
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
			err = parseStyleScaleNodes(logger, styleOptions, configValue)
		case "cluster":
			err = parseStyleCluster(logger, styleOptions, configValue)
		default:
			logger.Errorf("Skipping unknown visual option '%s'.", configKey)
			err = errors.New("invalid config")
		}
		if err != nil {
			return nil, err
		}
	}
	return styleOptions, nil
}

func parseStyleScaleNodes(logger *logrus.Logger, styleOptions *printer.StyleOptions, raw string) error {
	switch strings.ToLower(raw) {
	case "", "true", "on", "yes":
		styleOptions.ScaleNodes = true
	case "false", "off", "no":
		styleOptions.ScaleNodes = false
	default:
		logger.Errorf("Could not set 'scale_nodes' style to '%s'. Accepted values are 'true' and 'false'.", raw)
		return errors.New("invalid 'scale_nodes' value")
	}
	return nil
}

func parseStyleCluster(logger *logrus.Logger, styleOptions *printer.StyleOptions, raw string) error {
	switch strings.ToLower(raw) {
	case "off", "false", "no":
		styleOptions.Cluster = printer.Off
	case "", "shared", "on", "true", "yes":
		styleOptions.Cluster = printer.Shared
	case "full":
		styleOptions.Cluster = printer.Full
	default:
		logger.Errorf("Could not set 'cluster' style to '%s'. Accepted values are 'off', 'shared' and 'full'.", raw)
		return errors.New("invalid 'cluster' value")
	}
	return nil
}
