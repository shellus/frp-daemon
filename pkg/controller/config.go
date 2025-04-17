package controller

type ControllerConfig struct {
	MQTT MQTTConfig `yaml:"mqtt"`
}

type MQTTConfig struct {
	Broker string `yaml:"broker"`
}

func LoadControllerConfig(configPath string) (*ControllerConfig, error) {
	return nil, nil
}
