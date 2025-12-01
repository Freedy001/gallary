package model

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Setting 系统设置模型
type Setting struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Category  string    `gorm:"type:varchar(50);not null;index" json:"category"`   // auth, storage, cleanup
	Key       string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"key"` // 设置键名
	Value     string    `gorm:"type:text" json:"value"`                            // 设置值
	ValueType string    `gorm:"type:varchar(20);default:string" json:"value_type"` // string, int, bool, json
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}

// 设置分类常量
const (
	SettingCategoryAuth    = "auth"
	SettingCategoryStorage = "storage"
	SettingCategoryCleanup = "cleanup"
)

// 设置键名常量
const (
	// 认证相关
	SettingKeyAdminPassword   = "admin_password"
	SettingKeyPasswordVersion = "password_version" // 密码版本号，用于使旧token失效
	// 清理相关
	SettingKeyTrashAutoDeleteDays = "trash_auto_delete_days"
)

// 值类型常量
const (
	SettingValueTypeString = "string"
	SettingValueTypeInt    = "int"
	SettingValueTypeBool   = "bool"
	SettingValueTypeJSON   = "json"
)

// AuthDTO 存储配置 DTO
type AuthDTO struct {
	Password        string `json:"password"`
	PasswordVersion int    `json:"passwordVersion"`
}

// CleanupDTO 存储配置 DTO
type CleanupDTO struct {
	TrashAutoDeleteDays int `json:"trash_auto_delete_days"`
}

type StorageId string

const StorageTypeLocal StorageId = "local"

func (t StorageId) info() (string, string) {
	split := strings.Split(string(t), ",")
	if len(split) > 1 {
		return split[0], split[1]
	}
	return split[0], ""
}

func (t StorageId) DriverName() string {
	e, _ := t.info()
	switch e {
	case "local":
		return "local"
	case "":
		return ""
	default:
		return ""
	}
}

func AliyunpanStorageId(accountId string) StorageId {
	return StorageId("aliyunpan:" + accountId)
}

// StorageConfigDTO 存储配置 DTO
type StorageConfigDTO struct {
	DefaultId StorageId `json:"storageId"`

	LocalConfig     *LocalStorageConfig       `json:"localConfig,omitempty"`
	AliyunpanConfig []*AliyunPanStorageConfig `json:"aliyunpanConfig,omitempty"`
}

type LocalStorageConfig struct {
	Id        StorageId `json:"id"`
	BasePath  string    `json:"base_path"`
	URLPrefix string    `json:"url_prefix"`
}

type AliyunPanStorageConfig struct {
	Id                  StorageId `json:"id"`
	RefreshToken        string    `json:"refresh_token,omitempty"`        // 刷新Token
	BasePath            string    `json:"base_path,omitempty"`            // 云盘存储基础路径
	DriveType           string    `json:"drive_type,omitempty"`           // 网盘类型: file/album/resource
	DownloadChunkSize   int       `json:"download_chunk_size,omitempty"`  // 下载分片大小 (KB), 默认 512
	DownloadConcurrency int       `json:"download_concurrency,omitempty"` // 下载并发数, 默认 8
}

func ToSettings(category string, t any) []*Setting {
	r := reflect.TypeOf(t)
	var settings []*Setting

	for i := range r.NumField() {
		field := r.Field(i)
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonTag == "" || !field.IsExported() {
			continue
		}

		var valueType string
		var value string
		switch field.Type.Kind() {
		case reflect.String:
			valueType = SettingValueTypeString
			value = reflect.ValueOf(t).Field(i).String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			valueType = SettingValueTypeInt
			value = strconv.FormatInt(reflect.ValueOf(t).Field(i).Int(), 10)
		case reflect.Bool:
			valueType = SettingValueTypeBool
			value = strconv.FormatBool(reflect.ValueOf(t).Field(i).Bool())
		case reflect.Struct, reflect.Slice, reflect.Map:
			valueType = SettingValueTypeJSON
			marshal, _ := json.Marshal(reflect.ValueOf(t).Field(i).Interface())
			value = string(marshal)
		default:
			continue
		}

		settings = append(settings, &Setting{
			Category:  category,
			Key:       jsonTag,
			Value:     value,
			ValueType: valueType,
		})
	}

	return settings
}

func ToSettingDTO[T any](category string, settings []*Setting) T {
	var target T
	// 获取 target 指针指向的元素值 (这样才能进行 Set 操作)
	v := reflect.ValueOf(&target).Elem()
	t := v.Type()

	// 1. 将 settings 转换为 Map 以便快速查找 (Key -> Value)
	// Key 为 json tag 名, Value 为数据库存的字符串值
	settingMap := make(map[string]string)
	for _, s := range settings {
		// 只有匹配分类的才处理 (虽然通常入参已经筛选过，多做一次校验更安全)
		if s.Category == category {
			settingMap[s.Key] = s.Value
		}
	}

	// 2. 遍历结构体 T 的所有字段
	for i := 0; i < v.NumField(); i++ {
		structField := t.Field(i)

		// 跳过未导出的字段
		if !structField.IsExported() {
			continue
		}

		// 获取 json tag
		tag := structField.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		// 处理 "key,omitempty" 这种情况，只取逗号前的 "key"
		tagName := strings.Split(tag, ",")[0]

		// 3. 在 settings Map 中查找对应的值
		valStr, ok := settingMap[tagName]
		if !ok {
			continue // 没找到设置，保持字段的零值
		}

		fieldVal := v.Field(i)

		// 4. 根据字段的 Kind 进行类型转换和赋值
		if !fieldVal.CanSet() {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.String:
			// 包含 string 以及 type StorageId string 这种别名类型
			fieldVal.SetString(valStr)
		case reflect.Bool:
			if b, err := strconv.ParseBool(valStr); err == nil {
				fieldVal.SetBool(b)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if n, err := strconv.ParseInt(valStr, 10, 64); err == nil {
				fieldVal.SetInt(n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if n, err := strconv.ParseUint(valStr, 10, 64); err == nil {
				fieldVal.SetUint(n)
			}
		case reflect.Float32, reflect.Float64:
			if f, err := strconv.ParseFloat(valStr, 64); err == nil {
				fieldVal.SetFloat(f)
			}
		case reflect.Struct, reflect.Slice, reflect.Map:
			// 复杂类型（结构体、切片、Map）数据库中存的是 JSON 字符串
			// 直接反序列化到字段的地址上
			if valStr != "" {
				// fieldVal.Addr().Interface() 获取字段的指针指针 interface{}
				_ = json.Unmarshal([]byte(valStr), fieldVal.Addr().Interface())
			}
		default:
			panic("unhandled default case")
		}
	}

	return target
}
