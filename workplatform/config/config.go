package config

import (
	"github.com/silenceper/wechat/v2/cache"
)

// Config .config for 微信开放平台
type Config struct {
	CorpID             string `json:"corp_id"`     // corp_id,企业微信第三方代开发应用模板ID
	CorpSecret         string `json:"corp_secret"` // corp_secret,企业微信第三方代开发应用模板Secret
	Cache              cache.Cache
	Token              string `json:"token"`            // 微信客服回调配置，用于生成签名校验回调请求的合法性
	EncodingAESKey     string `json:"encoding_aes_key"` // 微信客服回调p配置，用于解密回调消息内容对应的密文
	PlatformTicketFunc func(corpId string) (string, error)
}
