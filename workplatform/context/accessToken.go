package context

import (
	"context"
	"fmt"

	"github.com/silenceper/wechat/v2/cache"
)

// GetAuthrAccessTokenContext 获取授权方AccessToken
func (ctx *Context) GetAuthrAccessTokenContext(stdCtx context.Context, authorizerCorpId string, authorizerAgentId string) (string, error) {
	authrTokenKey := "wework_authorizer_access_token_" + authorizerCorpId + "_" + authorizerAgentId
	val := cache.GetContext(stdCtx, ctx.Cache, authrTokenKey)
	if val == nil {
		refreshTokenKey := "wework_authorizer_refresh_token_" + authorizerCorpId + "_" + authorizerAgentId
		val := cache.GetContext(stdCtx, ctx.Cache, refreshTokenKey)
		if val == nil {
			return "", fmt.Errorf("cannot get authorizer %s %s refresh token", authorizerCorpId, authorizerAgentId)
		}
		//TODO::
		// token, err := ctx.RefreshAuthrTokenContext(stdCtx, authorizerCorpId, authorizerAgentId, val.(string))
		//if err != nil {
		//	return "", err
		//}
		// return token.AccessToken, nil
		return "", nil
	}

	return val.(string), nil
}

// GetAuthrAccessToken 获取授权方AccessToken
func (ctx *Context) GetAuthrAccessToken(authorizerCorpId string, authorizerAgentId string) (string, error) {
	return ctx.GetAuthrAccessTokenContext(context.Background(), authorizerCorpId, authorizerAgentId)
}
