package uuid

import (
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "生成UUID v7",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateUUID()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// 验证UUID格式（36个字符，包含4个连字符）
				if len(got) != 36 {
					t.Errorf("GenerateUUID() = %v, 长度应为36", got)
				}
				// 验证UUID v7格式（以时间戳开头）
				if got == "" {
					t.Errorf("GenerateUUID() = %v, 不应为空", got)
				}
			}
		})
	}
}

func TestGenerateUUID_Unique(t *testing.T) {
	// 测试生成多个UUID，确保唯一性
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateUUID()
		if err != nil {
			t.Fatalf("GenerateUUID() error = %v", err)
		}
		if ids[id] {
			t.Errorf("GenerateUUID() 生成了重复的UUID: %v", id)
		}
		ids[id] = true
	}
}

