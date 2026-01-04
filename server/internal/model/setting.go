package model

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

// Setting 系统设置模型
type Setting struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Category  string    `gorm:"type:varchar(50);not null;index" json:"Category"`   // auth, storage, cleanup
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
	SettingCategoryAI      = "ai"
)

// 值类型常量
const (
	SettingValueTypeString = "string"
	SettingValueTypeInt    = "int"
	SettingValueTypeBool   = "bool"
	SettingValueTypeJSON   = "json"
)

type SettingPO interface {
	Category() string
	ToSettings() []*Setting
}

// AuthPO 存储配置 DTO
type AuthPO struct {
	Password        string `json:"password"`
	PasswordVersion int64  `json:"passwordVersion"`
}

func (a AuthPO) Category() string {
	return SettingCategoryAuth
}

func (a AuthPO) ToSettings() []*Setting {
	return toSetting(a)
}

// CleanupPO 存储配置 DTO
type CleanupPO struct {
	TrashAutoDeleteDays int `json:"trash_auto_delete_days"`
}

func (a CleanupPO) Category() string {
	return SettingCategoryCleanup
}

func (a CleanupPO) ToSettings() []*Setting {
	return toSetting(a)
}

type StorageId string

const StorageTypeLocal StorageId = "local"

func (t StorageId) Info() (string, string) {
	split := strings.Split(string(t), ",")
	if len(split) > 1 {
		return split[0], split[1]
	}
	return split[0], ""
}

func (t StorageId) DriverName() string {
	e, _ := t.Info()
	switch e {
	case "local":
		return "本地存储"
	case "aliyunpan":
		return "阿里云盘"
	default:
		return ""
	}
}

func AliyunpanStorageId(accountId string) StorageId {
	return StorageId("aliyunpan:" + accountId)
}

type StorageItem interface {
	StorageId() StorageId
	Path() string
	ToSettings() []*Setting
}

// StorageConfigPO 存储配置 DTO
type StorageConfigPO struct {
	DefaultId StorageId `json:"storageId"`

	LocalConfig     *LocalStorageConfig       `json:"localConfig,omitempty"`
	AliyunpanConfig []*AliyunPanStorageConfig `json:"aliyunpanConfig,omitempty"`
	AliyunpanGlobal *AliyunPanGlobalConfig    `json:"aliyunpanGlobal,omitempty"` // 阿里云盘全局配置
}

func (a StorageConfigPO) Category() string {
	return SettingCategoryStorage
}

func (a StorageConfigPO) ToSettings() []*Setting {
	return toSetting(a)
}

func (a StorageConfigPO) GetStorageConfigById(id StorageId) StorageItem {
	if id == StorageTypeLocal {
		return a.LocalConfig
	}

	for _, config := range a.AliyunpanConfig {
		if config.StorageId() == id {
			return config
		}
	}

	return nil
}

type LocalStorageConfig struct {
	Id       StorageId `json:"id"`
	BasePath string    `json:"base_path"`
}

func (l *LocalStorageConfig) Path() string {
	return l.BasePath
}

func (l *LocalStorageConfig) StorageId() StorageId {
	return l.Id
}

func (l *LocalStorageConfig) ToSettings() []*Setting {
	return StorageConfigPO{LocalConfig: l}.ToSettings()
}

type AliyunPanStorageConfig struct {
	Id           StorageId `json:"id"`
	RefreshToken string    `json:"refresh_token,omitempty"` // 刷新Token
	BasePath     string    `json:"base_path,omitempty"`     // 云盘存储基础路径
	DriveType    string    `json:"drive_type,omitempty"`    // 网盘类型: file/album/resource
}

// AliyunPanGlobalConfig 阿里云盘全局配置（所有账号共享）
type AliyunPanGlobalConfig struct {
	DownloadChunkSize   int64 `json:"download_chunk_size"`  // 下载分片大小 (KB), 默认 512
	DownloadConcurrency int   `json:"download_concurrency"` // 下载并发数, 默认 8
}

func (l *AliyunPanStorageConfig) Path() string {
	return l.BasePath
}

func (l *AliyunPanStorageConfig) StorageId() StorageId {
	return l.Id
}

func (l *AliyunPanStorageConfig) ToSettings() []*Setting {
	return StorageConfigPO{AliyunpanConfig: []*AliyunPanStorageConfig{l}}.ToSettings()
}

func CreateStorageItemById(id StorageId) StorageItem {
	if id == StorageTypeLocal {
		return &LocalStorageConfig{Id: id}
	}

	if strings.HasPrefix(string(id), "aliyunpan") {
		return &AliyunPanStorageConfig{Id: id}
	}

	return nil
}

func toSetting(t SettingPO) []*Setting {
	if t.Category() == "" {
		panic("请实现 category 方法返回设置分类")
	}

	r := reflect.TypeOf(t)
	val := reflect.ValueOf(t)

	if r.Kind() == reflect.Ptr {
		r = r.Elem()
		val = val.Elem()
	}

	var settings []*Setting

	for i := range r.NumField() {
		field := r.Field(i)
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonTag == "" || !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		kind := field.Type.Kind()

		if (kind == reflect.Pointer ||
			kind == reflect.Slice ||
			kind == reflect.Map ||
			kind == reflect.Interface ||
			kind == reflect.Func ||
			kind == reflect.Chan) && fieldVal.IsNil() {
			continue
		}

		if kind == reflect.Ptr {
			fieldVal = fieldVal.Elem()
			kind = fieldVal.Kind()
		}

		var valueType string
		var value string
		switch kind {
		case reflect.String:
			valueType = SettingValueTypeString
			value = fieldVal.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			valueType = SettingValueTypeInt
			value = strconv.FormatInt(fieldVal.Int(), 10)
		case reflect.Bool:
			valueType = SettingValueTypeBool
			value = strconv.FormatBool(fieldVal.Bool())
		case reflect.Struct, reflect.Slice, reflect.Map:
			valueType = SettingValueTypeJSON
			marshal, _ := json.Marshal(fieldVal.Interface())
			value = string(marshal)
		default:
			continue
		}

		settings = append(settings, &Setting{
			Category:  t.Category(),
			Key:       jsonTag,
			Value:     value,
			ValueType: valueType,
		})
	}

	return settings
}

func ToSettingPO[T SettingPO](settings []*Setting) T {
	var target T
	// 获取 target 指针指向的元素值 (这样才能进行 Set 操作)
	v := reflect.ValueOf(&target).Elem()
	t := v.Type()

	// 1. 将 settings 转换为 Map 以便快速查找 (Key -> Value)
	// Key 为 json tag 名, Value 为数据库存的字符串值
	settingMap := make(map[string]string)
	for _, s := range settings {
		settingMap[s.Key] = s.Value
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

		setFieldValue(fieldVal, valStr)
	}

	return target
}

func setFieldValue(fieldVal reflect.Value, valStr string) {
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			elemType := fieldVal.Type().Elem()
			newVal := reflect.New(elemType)
			setFieldValue(newVal.Elem(), valStr)
			fieldVal.Set(newVal)
		} else {
			setFieldValue(fieldVal.Elem(), valStr)
		}
		return
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

// ==================== AI 配置 ====================
type Provider string

const (
	OpenAI                    Provider = "openAI"
	SelfHosted                Provider = "selfHosted"
	AliyunMultimodalEmbedding Provider = "alyunMultimodalEmbedding"
)

// ModelId 模型组合标识符（提供商ID,api_model_name）
type ModelId string

// Parse 解析组合ID，返回提供商ID和api模型名称
func (id ModelId) Parse() (providerId string, modelName string) {
	parts := strings.SplitN(string(id), ",", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return string(id), ""
}

// CreateModelId 创建组合ID
func CreateModelId(providerId, modelName string) ModelId {
	return ModelId(providerId + "," + modelName)
}

// ModelItem 单个模型配置项
type ModelItem struct {
	ApiModelName string `json:"api_model_name"` // API 调用时使用的模型名称
	ModelName    string `json:"model_name"`     // 内部标识/负载均衡分组
}

type ModelConfig struct {
	ID          string       `json:"id"`
	Provider    Provider     `json:"provider"`
	Models      []*ModelItem `json:"models"` // 模型列表（新）
	Endpoint    string       `json:"endpoint"`
	APIKey      string       `json:"api_key"`
	Enabled     bool         `json:"enabled"`
	ExtraConfig string       `json:"extra_config"`
}

// GetFirstModelItem 获取第一个模型项
func (m *ModelConfig) GetFirstModelItem() *ModelItem {
	if len(m.Models) > 0 {
		return m.Models[0]
	}
	return nil
}

func (m *ModelConfig) Hash() string {
	// 创建一个字符串切片，包含所有需要参与哈希计算的字段
	fields := []string{
		m.ID,
		string(m.Provider),
		strconv.FormatBool(m.Enabled),
		m.Endpoint,
		m.APIKey,
		m.ExtraConfig,
	}
	// 添加所有模型项
	for _, item := range m.Models {
		fields = append(fields, item.ApiModelName, item.ModelName)
	}
	// 将所有字段连接成一个字符串
	combined := strings.Join(fields, "|")

	h := sha1.Sum([]byte(combined))
	// 将哈希值转换为十六进制字符串并返回
	return hex.EncodeToString(h[:])
}

// AIGlobalConfig AI 全局配置
type AIGlobalConfig struct {
	DefaultSearchModelId         string `json:"default_search_model_id"`          // 默认搜索模型 ID
	DefaultTagModelId            string `json:"default_tag_model_id"`             // 默认打标签模型 ID
	DefaultPromptOptimizeModelId string `json:"default_prompt_optimize_model_id"` // 默认打标签模型 ID
	PromptOptimizeSystemPrompt   string `json:"prompt_optimize_system_prompt"`
}

// AIPo AI 配置 PO
type AIPo struct {
	Models       []*ModelConfig  `json:"models"`        // 通用模型配置
	GlobalConfig *AIGlobalConfig `json:"global_config"` // 全局配置
}

func (a AIPo) Category() string {
	return SettingCategoryAI
}

func (a AIPo) ToSettings() []*Setting {
	return toSetting(a)
}

// GetEnabled 获取所有启用的模型配置（包括自托管模型）
func (a AIPo) GetEnabled() []*ModelConfig {
	return lo.Filter(a.Models, func(item *ModelConfig, index int) bool { return item.Enabled })
}

// FindById 根据组合ID查找模型配置和模型项
// compositeId 格式: "providerId,apiModelName" 或旧格式 "providerId"
func (a AIPo) FindById(compositeId string) (*ModelConfig, *ModelItem) {
	providerId, modelName := ModelId(compositeId).Parse()

	provider, find := lo.Find(a.GetEnabled(), func(item *ModelConfig) bool { return item.ID == providerId })
	if !find {
		return nil, nil
	}

	// 如果指定了 modelName，查找对应的模型项
	if modelName != "" {
		modelItem, find := lo.Find(provider.Models, func(item *ModelItem) bool { return item.ModelName == modelName })
		if find {
			return provider, modelItem
		}
		return nil, nil
	}

	// 未指定 modelName，返回第一个模型项
	return provider, provider.GetFirstModelItem()
}

// ProviderWithModelItem 提供商配置和模型项的组合
type ProviderWithModelItem struct {
	Provider  *ModelConfig
	ModelItem *ModelItem
}

// FindModelConfigByModelName 根据 ModelName 查找所有启用的提供商配置（用于负载均衡）
// 返回所有包含指定 ModelName 的提供商配置及对应的模型项
func (a AIPo) FindModelConfigByModelName(modelName string) []*ProviderWithModelItem {
	return lo.FlatMap(a.GetEnabled(), func(provider *ModelConfig, index int) []*ProviderWithModelItem {
		return lo.FilterMap(provider.Models, func(item *ModelItem, index int) (*ProviderWithModelItem, bool) {
			if item.ModelName == modelName {
				return &ProviderWithModelItem{
					Provider:  provider,
					ModelItem: item,
				}, true
			}
			return nil, false
		})
	})
}

// GetDefaultTagModelName 获取默认打标签模型的 ModelName（用于负载均衡）
func (a AIPo) GetDefaultTagModelName() string {
	if a.GlobalConfig != nil && a.GlobalConfig.DefaultTagModelId != "" {
		provider, modelItem := a.FindById(a.GlobalConfig.DefaultTagModelId)
		if provider != nil && provider.Enabled && modelItem != nil {
			return modelItem.ModelName
		}
	}

	// 回退到第一个启用的模型的第一个模型项
	models := a.GetEnabled()
	if len(models) > 0 && len(models[0].Models) > 0 {
		return models[0].Models[0].ModelName
	}

	return ""
}
