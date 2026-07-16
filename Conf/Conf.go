package Conf

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)



func LoadEnvMap(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть конфиг %s: %w", path, err)
	}
	defer f.Close()

	result := make(map[string]string)

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: строка не в формате КУРС=ID (%q)", path, lineNum, line)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "" || value == "" {
			return nil, fmt.Errorf("%s:%d: пустой курс или ID (%q)", path, lineNum, line)
		}

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения %s: %w", path, err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("%s пуст — не найдено ни одного курса", path)
	}

	return result, nil
}
