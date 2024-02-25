package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Korrnals/testbox/infrastructure/config"
	"github.com/Korrnals/testbox/internal/testbox"
)

func main() {
	var cleanupNeeded bool // Флаг, указывающий на необходимость очистки

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Отсутствует файл конфигурации; %v", err)
	}

	dc, err := testbox.NewDockerClient()
	if err != nil {
		log.Fatalf("Не получается связаться с докером; %v", err)
	}

	defer func() {
		if cleanupNeeded {
			// Выполнение очистки в случае ошибки
			if err := dc.ClearAll(ctx, cfg); err != nil {
				log.Fatalf("Не удалось удалить артефакты; %v", err)
			}
			fmt.Println("Утилита завершена с ошибкой. Артефакты удалены.")
		}
	}()

	defer dc.ClearAll(ctx, cfg)

	if err := dc.RunDependImages(ctx, cfg); err != nil {
		cleanupNeeded = true
		log.Fatalf("Не удалось скачать и запустить образы зависимостей; %v", err)
	}

	if err := dc.BuildTestImage(cfg); err != nil {
		cleanupNeeded = true
		log.Fatalf("Не удалось собрать образ; %v", err)
	}

	if err := dc.RunContainer(ctx, cfg.ContainerName, cfg, cfg.ContainerName); err != nil {
		cleanupNeeded = true
		log.Fatalf("Не удалось запустить тестируемый контейнер; %v", err)
	}

	time.Sleep(time.Second * 5) // Ожидание запуска контейнера

	cIP, err := dc.GetContainerIP(ctx, cfg.ContainerName)
	if err != nil {
		cleanupNeeded = true
		log.Fatalf("Не удалось получить IP-адрес; %v", err)

	}

	// Запуск тестов
	err = testbox.RunTests(cfg, cIP)
	if err != nil {
		cleanupNeeded = true
		log.Fatalf("Не удалось выполнить тесты; %v", err)
	}

	// Если программа дошла до этой точки без ошибок, сбрасываем флаг очистки
	cleanupNeeded = false
	fmt.Println("Все операции успешно завершены. Утилита завершена без ошибок.")
}
