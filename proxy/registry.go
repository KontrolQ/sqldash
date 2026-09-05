package proxy

import (
	"sync"
	"time"

	"sqldash/database"
	"sqldash/models"
)

var (
	knownNames = make(map[string]bool)
	knownGuard sync.RWMutex
	knownUntil time.Time
)

func known(database string) bool {
	knownGuard.RLock()
	fresh := time.Now().Before(knownUntil)
	held := knownNames[database]
	knownGuard.RUnlock()

	if fresh {
		return held
	}

	refresh()

	knownGuard.RLock()
	defer knownGuard.RUnlock()

	return knownNames[database]
}

func Forget() {
	knownGuard.Lock()
	defer knownGuard.Unlock()

	knownUntil = time.Time{}
}

func refresh() {
	records := make([]models.Database, 0)

	if findError := database.DB.Find(&records).Error; findError != nil {
		return
	}

	names := make(map[string]bool, len(records))
	for _, record := range records {
		names[record.Name] = true
	}

	knownGuard.Lock()
	defer knownGuard.Unlock()

	knownNames = names
	knownUntil = time.Now().Add(RegistryFreshness)
}
