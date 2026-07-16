package Course

import (
	"sync"

	"AutoTracking/Conf"
)

var (
	mu      sync.RWMutex
	courses map[string]string 
)


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



func Choise(course string) (spreadsheetID string, ok bool) {
	mu.RLock()
	defer mu.RUnlock()

	id, ok := courses[course]
	return id, ok
}
