package context

import (
	"github.com/go-resty/resty/v2"
	"github.com/silenceper/wechat/v2/credential"
	"github.com/silenceper/wechat/v2/workplatform/config"
)

// Context struct
type Context struct {
	*config.Config
	credential.AccessTokenHandle

	RestyClient *resty.Client
}
