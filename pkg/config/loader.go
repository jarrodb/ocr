package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("port", 8080)
	viper.SetDefault("api_token", "mocktoken")
	viper.SetDefault("auth_required", true)
	viper.SetDefault("mode", "dev")
	viper.SetDefault("delay", "0s")
	viper.SetDefault("failure_rate", 0.0)
	viper.SetDefault("default_region", "us-ca")
	viper.SetDefault("generated_plate_prefix", "MCK")

	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/config")
	viper.AddConfigPath(".")

	_ = viper.ReadInConfig()
}

type Option func(*Config)

func WithFixtures(fx []Fixture) Option {
	return func(c *Config) { c.Fixtures = fx }
}

func WithVehicles(v []VehicleSeed) Option {
	return func(c *Config) { c.Vehicles = v }
}

func WithAPIToken(token string) Option {
	return func(c *Config) { c.APIToken = token }
}

func Load(opts ...Option) (*Config, error) {
	cfg := &Config{
		Port:                 viper.GetInt("port"),
		APIToken:             viper.GetString("api_token"),
		AuthRequired:         viper.GetBool("auth_required"),
		Mode:                 viper.GetString("mode"),
		Delay:                viper.GetString("delay"),
		FailureRate:          viper.GetFloat64("failure_rate"),
		CORSOrigins:          viper.GetStringSlice("cors_origins"),
		DefaultRegion:        viper.GetString("default_region"),
		GeneratedPlatePrefix: viper.GetString("generated_plate_prefix"),
	}

	if len(cfg.CORSOrigins) == 0 {
		cfg.CORSOrigins = []string{"*"}
	}

	if err := viper.UnmarshalKey("fixtures", &cfg.Fixtures); err != nil {
		return nil, fmt.Errorf("unmarshal fixtures: %w", err)
	}
	if err := viper.UnmarshalKey("vehicles", &cfg.Vehicles); err != nil {
		return nil, fmt.Errorf("unmarshal vehicles: %w", err)
	}

	if len(cfg.Vehicles) == 0 {
		cfg.Vehicles = defaultVehicleSeeds()
	}

	for _, opt := range opts {
		opt(cfg)
	}
	return cfg, nil
}

func defaultVehicleSeeds() []VehicleSeed {
	return []VehicleSeed{
		{Make: "Toyota", Model: "Camry", Color: "silver", Type: "Sedan", YearMin: 2018, YearMax: 2022},
		{Make: "Honda", Model: "Civic", Color: "blue", Type: "Sedan", YearMin: 2017, YearMax: 2021},
		{Make: "Ford", Model: "F-150", Color: "white", Type: "Pickup", YearMin: 2019, YearMax: 2023},
		{Make: "Chevrolet", Model: "Silverado", Color: "black", Type: "Pickup", YearMin: 2018, YearMax: 2022},
		{Make: "Tesla", Model: "Model 3", Color: "red", Type: "Sedan", YearMin: 2020, YearMax: 2024},
		{Make: "Subaru", Model: "Outback", Color: "green", Type: "Wagon", YearMin: 2016, YearMax: 2020},
		{Make: "Jeep", Model: "Wrangler", Color: "yellow", Type: "SUV", YearMin: 2015, YearMax: 2022},
		{Make: "Nissan", Model: "Altima", Color: "gray", Type: "Sedan", YearMin: 2017, YearMax: 2021},
	}
}
