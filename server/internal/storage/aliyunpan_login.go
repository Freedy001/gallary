package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// 阿里云盘登录相关常量
const (
	AuthHost     = "https://auth.aliyundrive.com"
	PassportHost = "https://passport.aliyundrive.com"
	APIHost      = "https://api.aliyundrive.com"

	// OAuth 授权路径
	OAuthAuthorize = "/v2/oauth/authorize"
	// 二维码生成
	QRCodeGenerate = "/newlogin/qrcode/generate.do"
	// 二维码查询
	QRCodeQuery = "/newlogin/qrcode/query.do"
	// Token 刷新
	AccountToken = "/v2/account/token"

	// 客户端 ID
	ClientID = "25dzX3vbYqktVxyX"
)

// QRCodeStatus 二维码扫描状态
type QRCodeStatus string

const (
	QRCodeStatusNew       QRCodeStatus = "NEW"       // 等待扫描
	QRCodeStatusScanned   QRCodeStatus = "SCANED"    // 已扫描，等待确认
	QRCodeStatusConfirmed QRCodeStatus = "CONFIRMED" // 已确认
	QRCodeStatusExpired   QRCodeStatus = "EXPIRED"   // 已过期
)

// QRCodeLoginResult 二维码登录结果
type QRCodeLoginResult struct {
	QRCodeURL    string       // 二维码内容（用于生成二维码图片）
	Status       QRCodeStatus // 当前状态
	Message      string       // 状态消息
	AccessToken  string       // 访问令牌（登录成功后）
	RefreshToken string       // 刷新令牌（登录成功后）
	ExpiresIn    int64        // 过期时间（秒）
	TokenType    string       // 令牌类型
	UserId       string       // 用户ID（用于构建 StorageId）
	UserName     string       // 用户名
	NickName     string       // 昵称
	Avatar       string       // 头像URL
}

// AliyunPanLogin 阿里云盘登录管理器
type AliyunPanLogin struct {
	client    *http.Client
	sessionID string
	qrData    *qrCodeData
}

// qrCodeData 二维码数据（用于查询状态）
type qrCodeData struct {
	T           int64  `json:"t"`
	Ck          string `json:"ck"`
	CodeContent string `json:"codeContent"`
}

// qrCodeGenerateResponse 二维码生成响应
type qrCodeGenerateResponse struct {
	Content struct {
		Data struct {
			T           int64  `json:"t"`
			CodeContent string `json:"codeContent"`
			Ck          string `json:"ck"`
			ResultCode  int    `json:"resultCode"`
			TitleMsg    string `json:"titleMsg"`
		} `json:"data"`
		Status int `json:"status"`
	} `json:"content"`
}

// qrCodeQueryResponse 二维码查询响应
type qrCodeQueryResponse struct {
	Content struct {
		Data struct {
			QRCodeStatus string `json:"qrCodeStatus"`
			ResultCode   int    `json:"resultCode"`
			BizExt       string `json:"bizExt"` // Base64 编码的登录结果
			St           string `json:"st"`
			LoginResult  string `json:"loginResult"`
		} `json:"data"`
		Status int `json:"status"`
	} `json:"content"`
}

// bizExtData bizExt 解码后的数据结构
type bizExtData struct {
	PDSLoginResult struct {
		Role           string                 `json:"role"`
		UserData       map[string]interface{} `json:"userData"`
		IsFirstLogin   bool                   `json:"isFirstLogin"`
		NeedLink       bool                   `json:"needLink"`
		LoginType      string                 `json:"loginType"`
		NickName       string                 `json:"nickName"`
		NeedRpVerify   bool                   `json:"needRpVerify"`
		Avatar         string                 `json:"avatar"`
		AccessToken    string                 `json:"accessToken"`
		UserName       string                 `json:"userName"`
		UserId         string                 `json:"userId"`
		DefaultDriveId string                 `json:"defaultDriveId"`
		ExistLink      []interface{}          `json:"existLink"`
		ExpiresIn      int64                  `json:"expiresIn"`
		ExpireTime     string                 `json:"expireTime"`
		RequestId      string                 `json:"requestId"`
		DataPinSetup   bool                   `json:"dataPinSetup"`
		State          string                 `json:"state"`
		TokenType      string                 `json:"tokenType"`
		DataPinSaved   bool                   `json:"dataPinSaved"`
		RefreshToken   string                 `json:"refreshToken"`
		Status         string                 `json:"status"`
	} `json:"pds_login_result"`
}

// tokenRefreshResponse Token 刷新响应
type tokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
	UserName     string `json:"user_name"`
	NickName     string `json:"nick_name"`
	Avatar       string `json:"avatar"`
}

// NewAliyunPanLogin 创建登录管理器
func NewAliyunPanLogin() (*AliyunPanLogin, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("创建 cookie jar 失败: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	return &AliyunPanLogin{
		client: client,
	}, nil
}

// GenerateQRCode 生成二维码
func (l *AliyunPanLogin) GenerateQRCode() (*QRCodeLoginResult, error) {
	// Step 1: OAuth 授权初始化，获取 SESSIONID
	authURL := fmt.Sprintf("%s%s?login_type=custom&response_type=code&redirect_uri=%s&client_id=%s&state=%s",
		AuthHost,
		OAuthAuthorize,
		url.QueryEscape("https://www.aliyundrive.com/sign/callback"),
		ClientID,
		url.QueryEscape(`{"origin":"file://"}`),
	)

	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建授权请求失败: %w", err)
	}
	l.setCommonHeaders(req)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("授权请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 提取 SESSIONID
	for _, cookie := range l.client.Jar.Cookies(resp.Request.URL) {
		if cookie.Name == "SESSIONID" {
			l.sessionID = cookie.Value
			break
		}
	}

	// Step 2: 生成二维码
	qrURL := fmt.Sprintf("%s%s?appName=aliyun_drive", PassportHost, QRCodeGenerate)
	req, err = http.NewRequest("GET", qrURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建二维码请求失败: %w", err)
	}
	l.setCommonHeaders(req)

	resp, err = l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("二维码请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取二维码响应失败: %w", err)
	}

	var qrResp qrCodeGenerateResponse
	if err := json.Unmarshal(body, &qrResp); err != nil {
		return nil, fmt.Errorf("解析二维码响应失败: %w", err)
	}

	if qrResp.Content.Status != 0 {
		return nil, fmt.Errorf("生成二维码失败: %s", qrResp.Content.Data.TitleMsg)
	}

	// 保存二维码数据用于后续查询
	l.qrData = &qrCodeData{
		T:           qrResp.Content.Data.T,
		Ck:          qrResp.Content.Data.Ck,
		CodeContent: qrResp.Content.Data.CodeContent,
	}

	return &QRCodeLoginResult{
		QRCodeURL: qrResp.Content.Data.CodeContent,
		Status:    QRCodeStatusNew,
		Message:   "等待扫描二维码",
	}, nil
}

// CheckQRCodeStatus 检查二维码扫描状态
func (l *AliyunPanLogin) CheckQRCodeStatus() (*QRCodeLoginResult, error) {
	if l.qrData == nil {
		return nil, fmt.Errorf("请先生成二维码")
	}

	// 构建查询请求
	queryURL := fmt.Sprintf("%s%s?appName=aliyun_drive", PassportHost, QRCodeQuery)

	// 构建表单数据
	formData := url.Values{}
	formData.Set("t", fmt.Sprintf("%d", l.qrData.T))
	formData.Set("ck", l.qrData.Ck)
	formData.Set("appName", "aliyun_drive")

	req, err := http.NewRequest("POST", queryURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建查询请求失败: %w", err)
	}
	l.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取查询响应失败: %w", err)
	}

	var queryResp qrCodeQueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, fmt.Errorf("解析查询响应失败: %w", err)
	}

	result := &QRCodeLoginResult{
		QRCodeURL: l.qrData.CodeContent,
		Status:    QRCodeStatus(queryResp.Content.Data.QRCodeStatus),
	}

	switch result.Status {
	case QRCodeStatusNew:
		result.Message = "等待扫描二维码"
	case QRCodeStatusScanned:
		result.Message = "已扫描，请在手机上确认登录"
	case QRCodeStatusConfirmed:
		result.Message = "登录成功"
		// 解析登录结果
		if err := l.parseLoginResult(queryResp.Content.Data.BizExt, result); err != nil {
			return nil, err
		}
	default:
		result.Status = QRCodeStatusExpired
		result.Message = "二维码已过期，请重新生成"
	}

	return result, nil
}

// parseLoginResult 解析登录结果
func (l *AliyunPanLogin) parseLoginResult(bizExtBase64 string, result *QRCodeLoginResult) error {
	if bizExtBase64 == "" {
		return fmt.Errorf("登录结果为空")
	}

	// Base64 解码
	decoded, err := base64.StdEncoding.DecodeString(bizExtBase64)
	if err != nil {
		return fmt.Errorf("Base64 解码失败: %w", err)
	}

	// 尝试使用 GB18030 解码（阿里云盘使用 GB18030 编码）
	// Go 标准库不直接支持 GB18030，这里假设返回的是 UTF-8 或兼容的编码
	// 如果遇到编码问题，可能需要引入 golang.org/x/text/encoding/simplifiedchinese

	var bizData bizExtData
	if err := json.Unmarshal(decoded, &bizData); err != nil {
		return fmt.Errorf("解析登录数据失败: %w, data: %s", err, string(decoded))
	}

	pds := bizData.PDSLoginResult

	// 使用返回的 refreshToken 去获取完整的 token
	fullToken, err := l.refreshToken(pds.RefreshToken)
	if err != nil {
		// 如果刷新失败，直接使用二维码登录返回的 token
		result.AccessToken = pds.AccessToken
		result.RefreshToken = pds.RefreshToken
		result.ExpiresIn = pds.ExpiresIn
		result.TokenType = pds.TokenType
	} else {
		result.AccessToken = fullToken.AccessToken
		result.RefreshToken = fullToken.RefreshToken
		result.ExpiresIn = fullToken.ExpiresIn
		result.TokenType = fullToken.TokenType
	}

	result.UserId = pds.UserId
	result.UserName = pds.UserName
	result.NickName = pds.NickName
	result.Avatar = pds.Avatar

	return nil
}

// refreshToken 使用 refreshToken 获取完整的 token
func (l *AliyunPanLogin) refreshToken(refreshToken string) (*tokenRefreshResponse, error) {
	tokenURL := fmt.Sprintf("%s%s", APIHost, AccountToken)

	reqBody := map[string]string{
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("创建刷新请求失败: %w", err)
	}
	l.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("刷新请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("刷新 Token 失败: %d, %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取刷新响应失败: %w", err)
	}

	var tokenResp tokenRefreshResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}

	return &tokenResp, nil
}

// setCommonHeaders 设置通用请求头
func (l *AliyunPanLogin) setCommonHeaders(req *http.Request) {
	req.Header.Set("Referer", "https://www.aliyundrive.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
}

// WaitForLogin 等待用户扫码登录（阻塞方式）
func (l *AliyunPanLogin) WaitForLogin(timeout time.Duration, onStatusChange func(status QRCodeStatus, message string)) (*QRCodeLoginResult, error) {
	startTime := time.Now()
	pollInterval := 2 * time.Second

	for {
		// 检查超时
		if time.Since(startTime) > timeout {
			return nil, fmt.Errorf("登录超时")
		}

		result, err := l.CheckQRCodeStatus()
		if err != nil {
			return nil, err
		}

		// 状态变化回调
		if onStatusChange != nil {
			onStatusChange(result.Status, result.Message)
		}

		switch result.Status {
		case QRCodeStatusConfirmed:
			return result, nil
		case QRCodeStatusExpired:
			return nil, fmt.Errorf("二维码已过期")
		}

		time.Sleep(pollInterval)
	}
}

// RefreshAccessToken 使用 refresh_token 刷新 access_token（静态方法）
func RefreshAccessToken(refreshToken string) (*tokenRefreshResponse, error) {
	login, err := NewAliyunPanLogin()
	if err != nil {
		return nil, err
	}
	return login.refreshToken(refreshToken)
}
