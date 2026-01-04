package uuid

import (
	"fmt"

	"github.com/google/uuid"
)

// GenerateUUID 生成UUID v7格式的字符串
// 返回36字符的标准UUID格式字符串，例如: "01234567-89ab-7def-0123-456789abcdef"
func GenerateUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("生成UUID失败: %w", err)
	}
	return id.String(), nil
}
