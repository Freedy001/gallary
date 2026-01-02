package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

// Vector 自定义向量类型，用于 PostgreSQL pgvector 扩展
type Vector []float32

// Value 实现 driver.Valuer 接口，将 Vector 转换为数据库可存储的值
func (v Vector) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nil, nil
	}
	return v.String(), nil
}

// String 将 Vector 转换为 PostgreSQL vector 格式的字符串
func (v Vector) String() string {
	if len(v) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i, f := range v {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	sb.WriteString("]")
	return sb.String()
}

// Scan 实现 sql.Scanner 接口，从数据库读取 Vector
func (v *Vector) Scan(src interface{}) error {
	if src == nil {
		*v = nil
		return nil
	}

	var str string
	switch val := src.(type) {
	case string:
		str = val
	case []byte:
		str = string(val)
	default:
		return fmt.Errorf("无法将 %T 转换为 Vector", src)
	}

	// 解析 PostgreSQL vector 格式: [1.0,2.0,3.0]
	str = strings.TrimSpace(str)
	if str == "" || str == "[]" {
		*v = nil
		return nil
	}

	// 移除方括号
	str = strings.TrimPrefix(str, "[")
	str = strings.TrimSuffix(str, "]")

	if str == "" {
		*v = nil
		return nil
	}

	parts := strings.Split(str, ",")
	result := make([]float32, len(parts))
	for i, part := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return fmt.Errorf("解析向量元素失败: %v", err)
		}
		result[i] = float32(f)
	}

	*v = result
	return nil
}

// FloatsToVectorString 将 float32 数组转换为 PostgreSQL 向量字符串
func FloatsToVectorString(floats []float32) string {
	if len(floats) == 0 {
		return "[]"
	}

	result := "["
	for i, f := range floats {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", f)
	}
	result += "]"
	return result
}
