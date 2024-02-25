package testbox

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Korrnals/testbox/infrastructure/config"
)

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

var mu sync.Mutex

// RunTests - метод запуска тестов из конфигурационного файла.
func RunTests(cfg *config.Config, cIP string) error {
	var wg sync.WaitGroup
	reportFileName := "report.txt"
	os.Remove(reportFileName) // Очистка файла от предыдущих тестов

	for _, test := range cfg.Tests {
		wg.Add(1)
		go func(t *config.Test) string {
			defer wg.Done()
			if err := executeTest(t, cfg, cIP, reportFileName); err != nil {
				fmt.Println(err)
			}

			return ""
		}(&test)
	}

	wg.Wait()
	fmt.Println("Все тесты выполнены.")

	return nil
}

// Метод executeTest - логика выполняемых тестов.
func executeTest(t *config.Test, cfg *config.Config, cIP string, reportFileName string) error {
	client := &http.Client{}
	var req *http.Request
	var err error
	var result string
	var responseBody []byte

	scheme := "http://"
	baseUrl := scheme + cIP + ":80" // Пример базового URL

	// Создание запроса в зависимости от типа
	switch t.QueryType {
	case "GET":
		req, err = http.NewRequest("GET", baseUrl+t.URL, nil)
	case "POST":
		requestBody := bytes.NewBufferString(*t.Query)
		req, err = http.NewRequest("POST", baseUrl+t.URL, requestBody)
	case "PUT":
		requestBody := bytes.NewBufferString(*t.Query)
		req, err = http.NewRequest("PUT", baseUrl+t.URL, requestBody)
	case "DELETE":
		req, err = http.NewRequest("DELETE", baseUrl+t.URL, nil)
	default:
		fmt.Printf("Неизвестный тип запроса: %s\n", t.QueryType)
	}

	if err != nil {
		fmt.Printf("Ошибка при создании запроса: %v\n", err)
	}

	// Выполнение запроса
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Ошибка при выполнении запроса: %v\n", err)
	}
	defer resp.Body.Close()

	// Проверка ожидаемого кода ответа
	// После выполнения запроса и всех проверок:
	if *t.ExpectedCode != resp.StatusCode {
		result = fmt.Sprintf("%s fail (ExpectedCode %d got %d)\n", t.Name, *t.ExpectedCode, resp.StatusCode)
	} else if *t.ResponseContains != "" && !strings.Contains(string(responseBody), *t.ResponseContains) {
		result = fmt.Sprintf("%s fail (Response does not contain expected string)\n", t.Name)
	} else {
		result = fmt.Sprintf("%s ok\n", t.Name)
	}

	// Печать результата в консоль - эту строку можно удалить, если она не требуется.
	fmt.Printf("%s", result)

	// Безопасная запись в файл
	mu.Lock()
	defer mu.Unlock()
	writeResultToFile(result, reportFileName)

	return nil
}
