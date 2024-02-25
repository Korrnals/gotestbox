package config

import (
	"flag"
	"log"

	"github.com/spf13/viper"
)

// const configFile = "config4test"

// Config - структура конфигурации приложения.
type Config struct {
	ContainerName string    `yaml:"containerName"`
	Dockerfile    string    `yaml:"dockerfile"`
	Environment   *[]string `yaml:"environment,omitempty"`
	Dependencies  *[]Dependency
	Tests         []Test
}

// Dependency - структура зависимости приложения.
type Dependency struct {
	Name        string   `yaml:"name"`
	Environment []string `yaml:"environment,omitempty"`
	Image       string   `yaml:"image"`
}

// Test определяет структуру теста.
type Test struct {
	Name             string  `yaml:"name"`
	URL              string  `yaml:"url"`
	QueryType        string  `yaml:"queryType"`
	Query            *string `yaml:"query,omitempty"`
	ExpectedCode     *int    `yaml:"expectedCode"`
	ResponseContains *string `yaml:"responseContains,omitempty"`
}

// NewConfig - конструктор для структуры Config.
func NewConfig() (*Config, error) {
	var configFile string
	// Создаем флаги командной строки для работы с конфигурационным файлом
	flag.StringVar(&configFile, "f", "./config.yaml", "путь до файла конфигурации")
	flag.Parse()

	// загружаем конфигурационный файл
	viper.SetConfigName(configFile) // определение имени конфигурационного файла
	viper.SetConfigType("yaml")     // определение типа конфигурационного файла
	viper.AddConfigPath("configs")  // поиск конфигурационного файла в директории configs
	viper.AddConfigPath(".")        // поиск конфигурационного файла в текущей директории

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Неверный YAML нельзя считать YAML: %v", err)
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("ошибка при заполнении конфигурации: %v", err)
	}

	return &config, nil
}
