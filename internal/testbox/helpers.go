package testbox

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Функция print - печать в консоль и обработка потенциальной ошибки.
func print(r io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(r) // Создание сканера по входному потоку.
	for scanner.Scan() {           // Читаем построчно.
		lastLine = scanner.Text() // Запоминаем последнюю строку.
		fmt.Println(lastLine)     // Печатаем последнюю строку.
	}

	if err := scanner.Err(); err != nil {
		return err // Возврат ошибки сканера, если таковая имеется.
	}

	if lastLine == "" {
		return errors.New("последняя строка пуста")
	}

	errLine := &ErrorLine{} // Создание экземпляра структуры ErrorLine.
	if err := json.Unmarshal([]byte(lastLine), errLine); err != nil {
		return fmt.Errorf("json unmarshal error: %v", err) // Ошибка десериализации JSON.
	}

	if errLine.Error != "" {
		return errors.New(errLine.Error) // Возврат ошибки в случае ошибки в docker API.
	}

	return nil
}

// writeResultToFile записывает результат теста в файл.
func writeResultToFile(result string, reportFileName string) {
	// Открытие файла с флагами O_APPEND, O_CREATE и O_WRONLY.
	// O_APPEND - для добавления данных в конец файла, если файл существует.
	// O_CREATE - для создания файла, если он не существует.
	// O_WRONLY - файл открывается только для записи.
	file, err := os.OpenFile(reportFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Ошибка при открытии файла: %v\n", err)
		return
	}
	defer file.Close()

	// Запись строки результата в файл.
	if _, err := file.WriteString(result + "\n"); err != nil {
		fmt.Printf("Ошибка при записи в файл: %v\n", err)
		return
	}
}
