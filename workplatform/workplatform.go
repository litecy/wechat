package workplatform

import (
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/silenceper/wechat/v2/credential"
	"github.com/silenceper/wechat/v2/work/server"
	"github.com/silenceper/wechat/v2/workplatform/basic"
	"github.com/silenceper/wechat/v2/workplatform/config"
	"github.com/silenceper/wechat/v2/workplatform/context"
	"github.com/silenceper/wechat/v2/workplatform/work"
)

// OpenPlatform 微信开放平台相关api
type WorkPlatform struct {
	*context.Context
}

type Options func(*context.Context)

func WithDebug(debug bool) Options {
	return func(ctx *context.Context) {
		ctx.RestyClient = ctx.RestyClient.EnableGenerateCurlOnDebug().SetDebug(debug)
	}
}

// NewOpenPlatform new openplatform
func NewWorkPlatform(cfg *config.Config, options ...Options) *WorkPlatform {
	ctx := &context.Context{
		Config:      cfg,
		RestyClient: resty.New(),
	}
	for _, option := range options {
		option(ctx)
	}
	return &WorkPlatform{ctx}
}

// GetServer get server
func (workPlatform *WorkPlatform) GetServer(req *http.Request, writer http.ResponseWriter) *server.Server {
	off := work.NewWork(workPlatform.Context, "authorizerCorpId", "authorizerAgentId")
	return off.GetServer(req, writer)
}

// GetWork 企业微信
func (workPlatform *WorkPlatform) GetWork(appID string) *work.Work {
	return work.NewWork(workPlatform.Context, "authorizerCorpId", "authorizerAgentId")
}

// GetBasic platform 基础功能
func (workPlatform *WorkPlatform) GetBasic() *basic.Basic {
	return basic.NewBasic(workPlatform.Context)
}

// SetAccessTokenHandle 自定义access_token获取方式
func (workPlatform *WorkPlatform) SetAccessTokenHandle(accessTokenHandle credential.AccessTokenHandle) {
	workPlatform.AccessTokenHandle = accessTokenHandle
}
