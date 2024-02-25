package testbox

import (
	"context"
	"errors"
	"fmt"

	"io"
	"log"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/Korrnals/testbox/infrastructure/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

// DockerClient - структура для работы с docker API.
type DockerClient struct {
	*client.Client
}

// Функция NewDockerClient - конструктор для структуры DockerClient.
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerClient{Client: cli}, nil
}

// Метод BuildTestImage - для сборки образа из архива.
func (dc *DockerClient) BuildTestImage(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	tar, err := archive.TarWithOptions(".", &archive.TarOptions{}) // создание архива из директории переданной через флаг
	if err != nil {
		log.Printf("Error: %v", err)
	}

	opts := types.ImageBuildOptions{ // опции сборки образа
		Dockerfile: cfg.Dockerfile,
		Tags:       []string{cfg.ContainerName},
		Remove:     true,
	}
	res, err := dc.ImageBuild(ctx, tar, opts) // сборка образа из архива
	if err != nil {
		log.Printf("Error: %v", err)
	}

	defer res.Body.Close() // закрытие потока после завершения сборки

	err = print(res.Body)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	return nil
}

// Метод RunContainer - запускает контейнер из образа.
func (dc *DockerClient) RunContainer(ctx context.Context, img string, cfg *config.Config, containerName string) error {
	// создаем контейнер из образа
	c, err := dc.ContainerCreate(ctx, &container.Config{
		Image: img,
		Env:   *cfg.Environment, // добавляем переменные окружения в контейнер
	}, nil, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := dc.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("run container: %w", err)
	}

	out, err := dc.ContainerLogs(ctx, c.ID, container.LogsOptions{ShowStdout: true}) // получаем логи контейнера
	if err != nil {
		return fmt.Errorf("container logs: %w", err)
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out) // выводим логи контейнера в консоль

	return nil
}

// Метод DestroyContainer - выполняет остановки и удаления контейнеров.
func (dc *DockerClient) DestroyContainer(ctx context.Context, containerID string) error {
	// останавливаем контейнер
	if err := dc.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return err
	}

	// ожидаем завершения контейнера
	statusCh, errCh := dc.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}

	// удаляем контейнер
	if err := dc.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		return err
	}
	fmt.Printf("container '%s' - removed successfully", containerID)

	return nil
}

// Функция RunDependImages скачивает образы для зависимостей и запускает их в контейнере.
func (dc *DockerClient) RunDependImages(ctx context.Context, cfg *config.Config) error {
	var errGroup errgroup.Group

	dependencies := *cfg.Dependencies // получаем зависимости из структуры []config.Dependency

	// скачиваем образы для зависимостей и запускаем в горутине
	for i := range dependencies {
		i := i // создаем локальную копию переменной для безопасного захвата в замыкании
		errGroup.Go(func() error {
			reader, err := dc.ImagePull(ctx, dependencies[i].Image, types.ImagePullOptions{}) // скачиваем образы для зависимостей
			if err != nil {
				return err
			}
			defer reader.Close()       // закрываем поток после завершения скачивания
			io.Copy(os.Stdout, reader) // выводим прогресс скачивания образа

			return dc.RunContainer(ctx, dependencies[i].Image, cfg, dependencies[i].Name) // запускаем контейнер
		})
	}
	if err := errGroup.Wait(); err != nil {
		return err
	}

	return nil
}

// Метод RemoveDepends - удаляет контейнеры и образы зависимостей.
func (dc *DockerClient) RemoveDepends(ctx context.Context, cfg *config.Config) error {
	depends := *cfg.Dependencies // получаем зависимости из структуры []config.Dependency

	if cfg.Dependencies == nil {
		return errors.New("зависимости отсутствуют в конфигурационном файле")
	}

	// удаляем артефакты зависимостей
	for _, dep := range depends {
		// удаляем контейнеры
		if err := dc.DestroyContainer(ctx, dep.Name); err != nil {
			return err
		}

		// удаляем образы
		resp, err := dc.ImageRemove(ctx, dep.Image, types.ImageRemoveOptions{})
		if err != nil {
			return err
		}
		fmt.Println(resp)
	}

	return nil
}

// Метод удаления собранного образа.
func (dc *DockerClient) BuildImageRemove(ctx context.Context, cfg *config.Config) error {
	image := cfg.ContainerName

	resp, err := dc.ImageRemove(ctx, image, types.ImageRemoveOptions{})
	if err != nil {
		return err
	}
	fmt.Println(resp)

	return nil
}

// Метод GetContainerIP - возвращает IP-адрес контейнера.
func (dc *DockerClient) GetContainerIP(ctx context.Context, containerID string) (string, error) {
	inspect, err := dc.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	return inspect.NetworkSettings.IPAddress, nil
}

// Метод ClearAll - выполняет очистку контейнеров и образов (удаление артефактов тестов).
func (dc *DockerClient) ClearAll(ctx context.Context, cfg *config.Config) error {
	// удаляем контейнеры
	dc.DestroyContainer(ctx, cfg.ContainerName)
	fmt.Println("container('s) removed successfully")
	// удаляем образы
	dc.RemoveDepends(ctx, cfg)
	fmt.Println("dependencies('s) removed successfully")
	// удаляем собранный тестовый образ
	dc.BuildImageRemove(ctx, cfg)
	fmt.Println("image('s) removed successfully")

	return nil
}
