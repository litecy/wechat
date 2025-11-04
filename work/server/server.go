package server

import (
	stdcontext "context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/silenceper/wechat/v2/util"
	"github.com/silenceper/wechat/v2/work/context"
	"github.com/silenceper/wechat/v2/work/message"
)

type RawMessageHandler func(stdcontext.Context, *http.Request, []byte) ([]byte, error)

// Server struct
type Server struct {
	*context.Context
	Writer  http.ResponseWriter
	Request *http.Request

	skipValidate bool

	openID string

	// 当存在 rawMessageHandler 时， 优先按照 rawMessageHandler 处理请求，否则按照 messageHandler 处理请求
	rawMessageHandler RawMessageHandler

	messageHandler func(stdcontext.Context, *message.MixMessage) *message.Reply

	RequestRawXMLMsg  []byte
	RequestMsg        *message.MixMessage
	ResponseRawXMLMsg []byte
	ResponseMsg       interface{}
	isEcho            bool // 是否是echo请求

	isSafeMode    bool
	isJSONContent bool
	random        []byte
	nonce         string
	timestamp     int64
}

// NewServer init
func NewServer(context *context.Context) *Server {
	srv := new(Server)
	srv.Context = context
	srv.skipValidate = true // 企业微信不校验签名
	return srv
}

// SkipValidate set skip validate
func (srv *Server) SkipValidate(skip bool) {
	srv.skipValidate = skip
}

// Serve 处理微信的请求消息
func (srv *Server) Serve() error {
	if !srv.Validate() {
		log.Error("Validate Signature Failed.")
		return fmt.Errorf("请求校验失败")
	}

	// echostr, exists := srv.GetQuery("echostr")
	// if exists {
	// 	srv.String(echostr)
	// 	return nil
	// }

	if srv.rawMessageHandler != nil {
		response, err := srv.handleRawRequest()
		if err != nil {
			return err
		}
		// 非安全模式下，请求处理方法返回为 nil 则直接回复 success 给微信服务器
		if response == nil && !srv.isSafeMode {
			srv.String("success")
			return nil
		}

		// debug print request msg
		log.Debugf("request msg =%s", string(srv.RequestRawXMLMsg))

		return srv.buildRawResponse(response)
	} else {
		response, err := srv.handleRequest()
		if err != nil {
			return err
		}
		// 非安全模式下，请求处理方法返回为 nil 则直接回复 success 给微信服务器
		if response == nil && !srv.isSafeMode {
			srv.String("success")
			return nil
		}

		// debug print request msg
		log.Debugf("request msg =%s", string(srv.RequestRawXMLMsg))

		return srv.buildResponse(response)
	}
	return nil
}

// Validate 校验请求是否合法
func (srv *Server) Validate() bool {
	if srv.skipValidate {
		return true
	}
	timestamp := srv.Query("timestamp")
	nonce := srv.Query("nonce")
	signature := srv.Query("signature")
	log.Debugf("validate signature, timestamp=%s, nonce=%s", timestamp, nonce)
	return signature == util.Signature(srv.Token, timestamp, nonce)
}

func (srv *Server) handleRawRequest() (reply []byte, err error) {
	// set isSafeMode
	srv.isSafeMode = false
	// encryptType := srv.Query("encrypt_type")
	echoStr := srv.Query("echostr")
	srv.isEcho = echoStr != ""
	// if encryptType == "aes" || srv.isEcho {
	srv.isSafeMode = true
	// }

	// set request contentType
	contentType := srv.Request.Header.Get("Content-Type")
	srv.isJSONContent = strings.Contains(contentType, "application/json")

	// set openID
	srv.openID = srv.Query("openid")

	_, err = srv.getRawMessage()
	if err != nil {
		return
	}

	if srv.isEcho {
		reply = []byte(srv.RequestRawXMLMsg)
		return reply, nil
	}

	reply, err = srv.rawMessageHandler(srv.Request.Context(), srv.Request, srv.RequestRawXMLMsg)
	return
}

// HandleRequest 处理微信的请求
func (srv *Server) handleRequest() (reply *message.Reply, err error) {
	// set isSafeMode
	srv.isSafeMode = false
	encryptType := srv.Query("encrypt_type")
	if encryptType == "aes" {
		srv.isSafeMode = true
	}

	// set request contentType
	contentType := srv.Request.Header.Get("Content-Type")
	srv.isJSONContent = strings.Contains(contentType, "application/json")

	// set openID
	srv.openID = srv.Query("openid")

	var msg interface{}
	msg, err = srv.getMessage()
	if err != nil {
		return
	}
	mixMessage, success := msg.(*message.MixMessage)
	if !success {
		err = errors.New("消息类型转换失败")
	}
	srv.RequestMsg = mixMessage
	reply = srv.messageHandler(srv.Request.Context(), mixMessage)
	return
}

// GetOpenID return openID
func (srv *Server) GetOpenID() string {
	return srv.openID
}

// getMessage 解析微信返回的消息
func (srv *Server) getMessage() (interface{}, error) {
	var rawXMLMsgBytes []byte
	var err error
	if srv.isSafeMode {
		encryptedXMLMsg, dataErr := srv.getEncryptBody()
		if dataErr != nil {
			return nil, dataErr
		}

		// 验证消息签名
		timestamp := srv.Query("timestamp")
		srv.timestamp, err = strconv.ParseInt(timestamp, 10, 32)
		if err != nil {
			return nil, err
		}
		nonce := srv.Query("nonce")
		srv.nonce = nonce
		msgSignature := srv.Query("msg_signature")
		msgSignatureGen := util.Signature(srv.Token, timestamp, nonce, encryptedXMLMsg.ToUserName)
		if msgSignature != msgSignatureGen {
			return nil, fmt.Errorf("消息不合法，验证签名失败")
		}

		// 解密
		srv.random, rawXMLMsgBytes, err = util.DecryptMsg(srv.CorpID, encryptedXMLMsg.EncryptedMsg, srv.EncodingAESKey)
		if err != nil {
			return nil, fmt.Errorf("消息解密失败, err=%v", err)
		}
	} else {
		rawXMLMsgBytes, err = io.ReadAll(srv.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("从body中解析xml失败, err=%v", err)
		}
	}

	srv.RequestRawXMLMsg = rawXMLMsgBytes

	return srv.parseRequestMessage(rawXMLMsgBytes)
}

// getMessage 解析微信返回的消息
func (srv *Server) getRawMessage() (interface{}, error) {
	var rawXMLMsgBytes []byte
	var err error
	if srv.isSafeMode {
		var encryptedXMLMsg *message.EncryptedXMLMsg
		echoStr := srv.Query("echostr")
		if echoStr == "" {

			var dataErr error
			encryptedXMLMsg, dataErr = srv.getEncryptBody()
			if dataErr != nil {
				return nil, dataErr
			}
		} else {
			encryptedXMLMsg = &message.EncryptedXMLMsg{
				EncryptedMsg: echoStr,
			}
		}

		// 验证消息签名
		timestamp := srv.Query("timestamp")
		srv.timestamp, err = strconv.ParseInt(timestamp, 10, 32)
		if err != nil {
			return nil, err
		}
		nonce := srv.Query("nonce")
		srv.nonce = nonce
		msgSignature := srv.Query("msg_signature")
		msgSignatureGen := util.Signature(srv.Token, timestamp, nonce, encryptedXMLMsg.EncryptedMsg)
		if msgSignature != msgSignatureGen {
			return nil, fmt.Errorf("消息不合法，验证签名失败")
		}

		if srv.isEcho {
			// 解密
			srv.random, rawXMLMsgBytes, err = util.DecryptMsgLoose(srv.CorpID, encryptedXMLMsg.EncryptedMsg, srv.EncodingAESKey)
			if err != nil {
				return nil, fmt.Errorf("消息解密失败, err=%v", err)
			}
		} else {
			// 解密
			srv.random, rawXMLMsgBytes, err = util.DecryptMsg(srv.AgentID, encryptedXMLMsg.EncryptedMsg, srv.EncodingAESKey)
			if err != nil {
				return nil, fmt.Errorf("消息解密失败, err=%v", err)
			}
		}
	} else {
		rawXMLMsgBytes, err = io.ReadAll(srv.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("从body中解析xml失败, err=%v", err)
		}
	}

	srv.RequestRawXMLMsg = rawXMLMsgBytes
	return nil, nil
}

func (srv *Server) getEncryptBody() (*message.EncryptedXMLMsg, error) {
	var encryptedXMLMsg = &message.EncryptedXMLMsg{}
	if srv.isJSONContent {
		if err := json.NewDecoder(srv.Request.Body).Decode(encryptedXMLMsg); err != nil {
			return nil, fmt.Errorf("从body中解析json失败,err=%v", err)
		}
	} else {
		if err := xml.NewDecoder(srv.Request.Body).Decode(encryptedXMLMsg); err != nil {
			return nil, fmt.Errorf("从body中解析xml失败,err=%v", err)
		}
	}
	return encryptedXMLMsg, nil
}

func (srv *Server) parseRequestMessage(rawXMLMsgBytes []byte) (msg *message.MixMessage, err error) {
	msg = &message.MixMessage{}
	if !srv.isJSONContent {
		err = xml.Unmarshal(rawXMLMsgBytes, msg)
		return
	}
	// parse json
	err = json.Unmarshal(rawXMLMsgBytes, msg)
	if err != nil {
		return
	}
	return
}

// SetMessageHandler 设置用户自定义的回调方法
func (srv *Server) SetMessageHandler(handler func(stdcontext.Context, *message.MixMessage) *message.Reply) {
	srv.messageHandler = handler
}

func (srv *Server) SetRawMessageHandler(handler RawMessageHandler) {
	srv.rawMessageHandler = handler
}

func (srv *Server) buildRawResponse(reply []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic error: %v\n%s", e, debug.Stack())
		}
	}()
	if reply == nil {
		// do nothing
		return nil
	}

	if string(reply) == "success" {
		srv.String("success")
		return nil
	}

	if srv.isEcho {
		srv.String(string(reply))
		return nil
	}

	srv.ResponseRawXMLMsg = reply
	return
}

func (srv *Server) buildResponse(reply *message.Reply) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic error: %v\n%s", e, debug.Stack())
		}
	}()
	if reply == nil {
		// do nothing
		return nil
	}
	// msgType := reply.MsgType
	// switch msgType {
	// case message.MsgTypeNoop:
	// case message.MsgTypeText:
	// case message.MsgTypeImage:
	// case message.MsgTypeVoice:
	// case message.MsgTypeVideo:
	// case message.MsgTypeMusic:
	// case message.MsgTypeNews:
	// case message.MsgTypeTransfer:
	// default:
	// 	err = message.ErrUnsupportReply
	// 	return
	// }

	// if msgType == message.MsgTypeNoop {
	// 	srv.String("success")
	// 	return nil
	// }

	// msgData := reply.MsgData
	// value := reflect.ValueOf(msgData)
	// // msgData must be a ptr
	// kind := value.Kind().String()
	// if kind != "ptr" {
	// 	return message.ErrUnsupportReply
	// }

	// params := make([]reflect.Value, 1)
	// params[0] = reflect.ValueOf(srv.RequestMsg.FromUserName)
	// value.MethodByName("SetToUserName").Call(params)

	// params[0] = reflect.ValueOf(srv.RequestMsg.ToUserName)
	// value.MethodByName("SetFromUserName").Call(params)

	// params[0] = reflect.ValueOf(msgType)
	// value.MethodByName("SetMsgType").Call(params)

	// params[0] = reflect.ValueOf(util.GetCurrTS())
	// value.MethodByName("SetCreateTime").Call(params)

	// srv.ResponseMsg = msgData
	// srv.ResponseRawXMLMsg, err = xml.Marshal(msgData)
	return
}

// Send 将自定义的消息发送
func (srv *Server) Send() (err error) {
	if len(srv.ResponseRawXMLMsg) == 0 {
		return nil
	}
	srv.String("success")
	return nil
	// if len(srv.ResponseRawXMLMsg) == 0 {
	// 	return nil
	// }
	// replyMsg := srv.ResponseMsg
	// log.Debugf("response msg =%+v", replyMsg)
	// if srv.isSafeMode {
	// 	// 安全模式下对消息进行加密
	// 	var encryptedMsg []byte
	// 	encryptedMsg, err = util.EncryptMsg(srv.random, srv.ResponseRawXMLMsg, srv.AppID, srv.EncodingAESKey)
	// 	if err != nil {
	// 		return
	// 	}
	// 	// TODO 如果获取不到 timestamp nonce 则自己生成
	// 	timestamp := srv.timestamp
	// 	timestampStr := strconv.FormatInt(timestamp, 10)
	// 	msgSignature := util.Signature(srv.Token, timestampStr, srv.nonce, string(encryptedMsg))
	// 	replyMsg = message.ResponseEncryptedXMLMsg{
	// 		EncryptedMsg: string(encryptedMsg),
	// 		MsgSignature: msgSignature,
	// 		Timestamp:    timestamp,
	// 		Nonce:        srv.nonce,
	// 	}
	// }
	// if replyMsg != nil {
	// 	srv.XML(replyMsg)
	// }
	return
}
