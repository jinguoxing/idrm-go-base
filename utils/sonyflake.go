package utils

import (
	"errors"
	"net"
	"os"

	"github.com/sony/sonyflake"
)

var (
	sf *sonyflake.Sonyflake
)

// https://github.com/tinrab/makaroni/tree/master/utilities/unique-id
// NewMachineID 根据ip获取唯一id（增强版：支持本地开发环境）
func NewMachineID() func() (uint16, error) {
	return func() (uint16, error) {
		ipStr := os.Getenv("POD_IP")

		// 本地开发环境：如果没有设置POD_IP，使用本地IP或默认值
		if ipStr == "" {
			// 尝试获取本机IP
			addrs, err := net.InterfaceAddrs()
			if err == nil {
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
						if ipnet.IP.To4() != nil {
							ipStr = ipnet.IP.String()
							break
						}
					}
				}
			}

			// 如果还是没有IP，使用默认值
			if ipStr == "" {
				ipStr = "127.0.0.1"
			}
		}

		ip := net.ParseIP(ipStr)
		ip = ip.To16()
		if ip == nil || len(ip) < 4 {
			return 0, errors.New("invalid IP")
		}
		return uint16(ip[14])<<8 + uint16(ip[15]), nil
	}
}

// GetUniqueID 使用sonyflake获取唯一、自增id
func GetUniqueID() (uint64, error) {
	return sf.NextID()
}

// init 初始化sonyflake
// ⚠️ 注意：为保持与原项目兼容，未设置 StartTime
// 将使用 sonyflake 默认值：2014-09-01 00:00:00 UTC
// 如需修改 StartTime，请确保不会影响已存在的 ID
func init() {
	var st sonyflake.Settings
	st.MachineID = NewMachineID()
	// 关键：不设置 st.StartTime，保持与原项目的兼容性

	sf = sonyflake.NewSonyflake(st)
	if sf == nil {
		panic("sonyflake not created")
	}
}
