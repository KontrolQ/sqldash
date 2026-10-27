package redishost

import (
	"sqldash/config"
	"sqldash/models"

	"gorm.io/gorm"
)

func restoreMemoryAllowed(used, excluded, limit int64) bool {
	return limit == 0 || used-excluded <= limit-max(int64(1024*1024), limit/10)
}

func memoryExceedsBudget(limit int64) bool {
	return config.Redis.MemoryBudget > 0 && (limit == 0 || limit > config.Redis.MemoryBudget)
}

func checkCapacity(transaction *gorm.DB) error {
	if config.Redis.MaximumInstances > 0 {
		var instances int64
		if err := transaction.Model(&models.RedisInstance{}).Count(&instances).Error; err != nil {
			return Unavailable
		}
		if instances > int64(config.Redis.MaximumInstances) {
			return CapacityExceeded
		}
	}
	return checkMemoryCapacity(transaction)
}

func initialMemoryLimit(transaction *gorm.DB) (int64, error) {
	if config.Redis.MemoryLimit > 0 || config.Redis.MemoryBudget == 0 {
		return config.Redis.MemoryLimit, nil
	}
	reserved, err := reservedMemory(transaction)
	if err != nil {
		return 0, err
	}
	if reserved > config.Redis.MemoryBudget || config.Redis.MemoryBudget-reserved < instanceOverhead+1024*1024 {
		return 0, CapacityExceeded
	}
	return config.Redis.MemoryBudget - reserved - instanceOverhead, nil
}

func checkMemoryCapacity(transaction *gorm.DB) error {
	if config.Redis.MemoryBudget == 0 {
		return nil
	}
	reserved, err := reservedMemory(transaction)
	if err != nil {
		return err
	}
	if reserved > config.Redis.MemoryBudget {
		return CapacityExceeded
	}
	return nil
}

func reservedMemory(transaction *gorm.DB) (int64, error) {
	var unlimited int64
	if err := transaction.Model(&models.RedisInstance{}).Where("memory_limit = 0").Count(&unlimited).Error; err != nil {
		return 0, Unavailable
	}
	if unlimited > 0 {
		return 0, CapacityExceeded
	}
	var reserved int64
	if err := transaction.Model(&models.RedisInstance{}).Select("coalesce(sum(memory_limit + ?), 0)", instanceOverhead).Scan(&reserved).Error; err != nil {
		return 0, Unavailable
	}
	if reserved < 0 {
		return 0, CapacityExceeded
	}
	return reserved, nil
}
