package work

import (
	"github.com/silenceper/wechat/v2/credential"
	"github.com/silenceper/wechat/v2/work"
	workConfig "github.com/silenceper/wechat/v2/work/config"
	wpContext "github.com/silenceper/wechat/v2/workplatform/context"
)

// OfficialAccount 代公众号实现业务
type Work struct {
	// 授权的公众号的appID
	authorizerCorpId  string
	authorizerAgentId string
	*work.Work
}

// NewOfficialAccount 实例化
// appID :为授权方公众号 APPID，非开放平台第三方平台 APPID
func NewWork(wpCtx *wpContext.Context, authorizerCorpId string, authorizerAgentId string) *Work {
	work := work.NewWork(&workConfig.Config{
		CorpID:         wpCtx.CorpID,
		CorpSecret:     wpCtx.CorpSecret,
		Cache:          wpCtx.Cache,
		Token:          wpCtx.Token,
		EncodingAESKey: wpCtx.EncodingAESKey,
	})
	// 设置获取access_token的函数
	work.SetAccessTokenHandle(NewDefaultAuthrAccessToken(wpCtx, authorizerCorpId, authorizerAgentId))
	return &Work{authorizerCorpId: authorizerCorpId, authorizerAgentId: authorizerAgentId, Work: work}
}

// DefaultAuthrAccessToken 默认获取授权ak的方法
type DefaultAuthrAccessToken struct {
	wpCtx             *wpContext.Context
	authorizerCorpId  string
	authorizerAgentId string
}

// NewDefaultAuthrAccessToken New
func NewDefaultAuthrAccessToken(wpCtx *wpContext.Context, authorizerCorpId string, authorizerAgentId string) credential.AccessTokenHandle {
	return &DefaultAuthrAccessToken{
		wpCtx:             wpCtx,
		authorizerCorpId:  authorizerCorpId,
		authorizerAgentId: authorizerAgentId,
	}
}

// GetAccessToken 获取ak
func (ak *DefaultAuthrAccessToken) GetAccessToken() (string, error) {
	return ak.wpCtx.GetAuthrAccessToken(ak.authorizerCorpId, ak.authorizerAgentId)
}
