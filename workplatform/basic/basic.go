package basic

import (
	stdContext "context"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
	"github.com/silenceper/wechat/v2/workplatform/context"
)

const (
	getPermantCodeURL = "https://qyapi.weixin.qq.com/cgi-bin/service/v2/get_permanent_code?suite_access_token=%s"
)

// Basic 基础信息设置
type Basic struct {
	*context.Context
}

// NewBasic new
func NewBasic(ctx *context.Context) *Basic {
	return &Basic{ctx}
}

/*
参数说明：

参数	说明
permanent_code	企业微信永久授权码,最长为512字节
auth_corp_info	授权方企业信息
auth_corp_info.corpid	授权方企业微信id
auth_corp_info.corp_name	授权方企业名称，即企业简称
auth_user_info	授权管理员的信息，可能不返回
auth_user_info.userid	授权管理员的userid，可能为空
auth_user_info.open_userid	授权管理员的open_userid，可能为空
auth_user_info.name	授权管理员的name，可能为空
auth_user_info.avatar	授权管理员的头像url，可能为空
register_code_info	推广二维码安装相关信息，扫推广二维码安装时返回。成员授权时暂不支持。（注：无论企业是否新注册，只要通过扫推广二维码安装，都会返回该字段）
register_code_info.register_code	注册码
register_code_info.template_id	推广包ID
register_code_info.state	仅当获取注册码指定该字段时才返回
state	安装应用时，扫码或者授权链接中带的state值。详见state说明
*/
type AuthInfo struct {
	util.CommonError

	PermanentCode    string            `json:"permanent_code"` //
	AuthCorpInfo     *AuthCorpInfo     `json:"auth_corp_info"`
	AuthUserInfo     *AuthUserInfo     `json:"auth_user_info"`
	RegisterCodeInfo *RegisterCodeInfo `json:"register_code_info"`
	State            string            `json:"state"`
}

type AuthCorpInfo struct {
	CorpId   string `json:"corpid"`
	CorpName string `json:"corp_name"`
}

type AuthUserInfo struct {
	UserId     string `json:"userid"`
	OpenUserId string `json:"open_userid"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
}

type RegisterCodeInfo struct {
	RegisterCode string `json:"register_code"`
	TemplateId   string `json:"template_id"`
	State        string `json:"state"`
}

// GetBasicInfo 获取基础信息
func (b *Basic) GetBasicInfoContext(stdCtx stdContext.Context, platformAccessToken, authCode string) (*AuthInfo, error) {
	req := map[string]string{
		"auth_code": authCode,
	}
	uri := fmt.Sprintf(getPermantCodeURL, platformAccessToken)

	var ret AuthInfo

	_, err := b.RestyClient.R().SetContext(stdCtx).SetBody(req).SetResult(&ret).Post(uri)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
