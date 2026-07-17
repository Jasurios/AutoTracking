package Course

import (
	"sync"

	"AutoTracking/Conf"
)

var (
	mu      sync.RWMutex
	courses map[string]string // курс -> spreadsheetID, грузится из config.env
)

// Init читает config.env и кладёт курсы в память. Дергать один раз при старте.
func Init(path string) error {
	loaded, err := Conf.LoadEnvMap(path)
	if err != nil {
		return err
	}

	mu.Lock()
	courses = loaded
	mu.Unlock()

	return nil
}

// Choise возвращает spreadsheetID по имени курса.
// mu.RLock тут не просто так — Handler дергает Choise на каждый HTTP-запрос,
// а Init теоретически может переинициализировать courses в рантайме
func Choise(course string) (spreadsheetID string, ok bool) {
	mu.RLock()
	defer mu.RUnlock()

	id, ok := courses[course]
	return id, ok
}
