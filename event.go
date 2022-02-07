package gonebot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	POST_TYPE_META    = "meta_event"
	POST_TYPE_MESSAGE = "message"
	POST_TYPE_NOTICE  = "notice"
	POST_TYPE_REQUEST = "request"
)

type EventName string

const (
	EVENT_NAME_MESSAGE           EventName = "message"
	EVENT_NAME_PRIVATE_MESSAGE   EventName = "message.private"
	EVENT_NAME_GROUP_MESSAGE     EventName = "message.group"
	EVENT_NAME_NOTICE            EventName = "notice"
	EVENT_NAME_GROUP_UPLOAD      EventName = "notice.group_upload"
	EVENT_NAME_GROUP_ADMIN       EventName = "notice.group_admin"
	EVENT_NAME_GROUP_DECREASE    EventName = "notice.group_decrease"
	EVENT_NAME_GROUP_INCREASE    EventName = "notice.group_increase"
	EVENT_NAME_GROUP_BAN         EventName = "notice.group_ban"
	EVENT_NAME_FRIEND_ADD        EventName = "notice.friend_add"
	EVENT_NAME_GROUP_RECALL      EventName = "notice.group_recall"
	EVENT_NAME_FRIEND_RECALL     EventName = "notice.friend_recall"
	EVENT_NAME_NOTIFY            EventName = "notice.notify"
	EVENT_NAME_NOTIFY_POKE       EventName = "notice.notify.poke"
	EVENT_NAME_NOTIFY_LUCKY_KING EventName = "notice.notify.lucky_king"
	EVENT_NAME_NOTIFY_HONOR      EventName = "notice.notify.honor"
	EVENT_NAME_REQUEST           EventName = "request"
	EVENT_NAME_REQUEST_FRIEND    EventName = "request.friend"
	EVENT_NAME_REQUEST_GROUP     EventName = "request.group"
	EVENT_NAME_META              EventName = "meta_event"
	EVENT_NAME_META_LIFECYCLE    EventName = "meta_event.lifecycle"
	EVENT_NAME_META_HEARTBEAT    EventName = "meta_event.heartbeat"
)

type I_Event interface {
	GetPostType() string
	GetEventName() string
	GetEventDescription() string

	IsMessageEvent() bool
	IsToMe() bool
}

type Event struct {
	Time     int64  `json:"time"`      // 事件发生的时间戳
	SelfId   int64  `json:"self_id"`   // 收到事件的机器人的QQ号
	PostType string `json:"post_type"` // 事件的类型，message, notice, request, meta_event

	EventName string `json:"-"` // 事件的名称，形如：notice.group.set
	ToMe      bool   `json:"-"` // 是否与我（bot）有关（即私聊我、或群聊At我、我被踢了、等等）
}

// 获取事件的上报类型，有message, notice, request, meta_event
func (e *Event) GetPostType() string {
	return e.PostType
}

// 获取事件的名称，形如：notice.group.set
func (e *Event) GetEventName() string {
	return e.EventName
}

// 获取事件的描述，一般用于日志输出
func (e *Event) GetEventDescription() string {
	return fmt.Sprintf("[%s]: %+v", e.EventName, *e)
}

func (e *Event) IsMessageEvent() bool {
	return e.PostType == POST_TYPE_MESSAGE
}

func (e *Event) IsToMe() bool {
	return e.ToMe
}

// 从JSON对象中生成Event对象（指针）
func convertJsonObjectToEvent(obj gjson.Result) I_Event {
	// 大多数事件的事件类型有3级，而第三级的subtype通常不影响事件的结构
	// 所以下面只用了第一级和第二级类型来构造事件对象
	postType := obj.Get("post_type").String()
	nextType := obj.Get(postType + "_type").String()
	typeName := fmt.Sprintf("%s.%s", postType, nextType) // 前两段类型

	subType := ""
	fullTypeName := typeName
	if obj.Get("sub_type").Exists() {
		subType = obj.Get("sub_type").String()
		fullTypeName = fmt.Sprintf("%s.%s", typeName, subType)
	}

	var ev I_Event
	switch typeName {
	case "message.private":
		ev = &PrivateMessageEvent{}
	case "message.group":
		ev = &GroupMessageEvent{}
	case "notice.group_upload":
		ev = &GroupUploadNoticeEvent{}
	case "notice.group_admin":
		ev = &GroupAdminNoticeEvent{}
	case "notice.group_decrease":
		ev = &GroupDecreaseNoticeEvent{}
	case "notice.group_increase":
		ev = &GroupIncreaseNoticeEvent{}
	case "notice.group_ban":
		ev = &GroupBanNoticeEvent{}
	case "notice.friend_add":
		ev = &FriendAddNoticeEvent{}
	case "notice.group_recall":
		ev = &GroupRecallNoticeEvent{}
	case "notice.friend_recall":
		ev = &FriendRecallNoticeEvent{}
	case "notice.notify":
		switch subType {
		case "poke":
			ev = &PokeNoticeEvent{}
		case "lucky_king":
			ev = &LuckyKingNoticeEvent{}
		case "honor":
			ev = &HonorNoticeEvent{}
		}
	case "request.friend":
		ev = &FriendRequestEvent{}
	case "request.group":
		ev = &GroupRequestEvent{}
	case "meta_event.lifecycle":
		ev = &LifeCycleMetaEvent{}
	case "meta_event.heartbeat":
		ev = &HeartbeatMetaEvent{}
	default:
		panic(fmt.Sprintf("unknown event type: %s", typeName))
	}

	// 借助json库将JSON对象中的字段赋值给Event对象，懒得自个写反射了
	err := json.Unmarshal([]byte(obj.Raw), ev)
	if err != nil {
		panic(err)
	}

	// 设置事件的名称
	setEventField(ev, "EventName", fullTypeName)
	if isEventRelativeToBot(ev) {
		setEventField(ev, "ToMe", true)
	}

	return ev
}

type I_MessageEvent interface {
	GetPostType() string
	GetEventName() string
	GetEventDescription() string
	IsToMe() bool

	GetMessageType() string
	GetSessionId() string
	GetMessage() *Message
	ExtractPlainText() string
}

type MessageEvent struct {
	Event
	MessageType string          `json:"message_type"` // 消息类型，group, private
	SubType     string          `json:"sub_type"`     // 消息子类型，friend, group, other
	MessageId   int32           `json:"message_id"`   // 消息ID
	UserId      int64           `json:"user_id"`      // 消息发送者的QQ号
	Message     Message `json:"message"`      // 消息内容
	RawMessage  string          `json:"raw_message"`  // 原始消息内容
	Font        int32           `json:"font"`         // 字体

}

func (e *MessageEvent) GetMessageType() string {
	return e.MessageType
}

func (e *MessageEvent) GetSessionId() string {
	return fmt.Sprintf("%d", e.UserId)
}

func (e *MessageEvent) GetMessage() *Message {
	return &e.Message
}

func (e *MessageEvent) ExtractPlainText() string {
	return e.Message.ExtractPlainText()
}

func (e *MessageEvent) IsToMe() bool {
	return e.ToMe
}

type MessageEventSender struct {
	UserId   int64  `json:"user_id"`  // 消息发送者的QQ号
	Nickname string `json:"nickname"` // 消息发送者的昵称
	Sex      string `json:"sex"`      // 性别，male 或 female 或 unknown
	Age      int32  `json:"age"`
}

type GroupMessageEventSender struct {
	MessageEventSender
	Card  string `json:"card"`  // 群名片/备注
	Area  string `json:"area"`  // 地区
	Level string `json:"level"` // 成员等级
	Role  string `json:"role"`  // 角色，owner 或 admin 或 member
	Title string `json:"title"` // 专属头衔
}

type PrivateMessageEvent struct {
	MessageEvent
	Sender *MessageEventSender // 发送人信息
}

func (e *PrivateMessageEvent) GetEventDescription() string {
	msg := e.Message.String()
	msgRune := []rune(msg)
	if len(msgRune) > 100 {
		msg = fmt.Sprintf("%s...(省略%d个字符)...%s", string(msgRune[:50]), len(msgRune)-100, string(msgRune[len(msgRune)-50:]))
	}
	msg = strings.Replace(msg, "\n", "\\n", -1)
	return fmt.Sprintf("[私聊消息](#%d 来自%d): %v", e.MessageId, e.UserId, msg)
}

type Anonymous struct {
	Id   int64  `json:"id"`   // 匿名用户的ID
	Name string `json:"name"` // 匿名用户的名词
	Flag string `json:"flag"` // 匿名用户 flag，在调用禁言 API 时需要传入
}

type GroupMessageEvent struct {
	MessageEvent
	GroupId   int64                    `json:"group_id"` // 群号
	Sender    *GroupMessageEventSender `json:"sender"`   // 发送人信息
	Anonymous *Anonymous               `json:"anonymous"`
}

func (e *GroupMessageEvent) GetEventDescription() string {
	msg := e.Message.String()
	msgRune := []rune(msg)
	if len(msgRune) > 100 {
		msg = fmt.Sprintf("%s...(省略%d个字符)...%s", string(msgRune[:50]), len(msgRune)-100, string(msgRune[len(msgRune)-50:]))
	}
	msg = strings.Replace(msg, "\n", "\\n", -1)
	return fmt.Sprintf("[群聊消息](#%d 来自%d@群%d): %v", e.MessageId, e.UserId, e.GroupId, msg)
}

func (e *GroupMessageEvent) GetSessionId() string {
	return fmt.Sprintf("%d@%d", e.UserId, e.GroupId)
}

type LifeCycleMetaEvent struct {
	Event
	MetaEventType string `json:"meta_event_type"` // 元事件类型，lifecycle
	SubType       string `json:"sub_type"`        // 元事件子类型，enable、disable、connect
}

type HeartbeatMetaEvent struct {
	Event
	MetaEventType string `json:"meta_event_type"` // 元事件类型，heartbeat
	Status        struct {
		Online bool `json:"online"` // 在线状态
		Good   bool `json:"good"`   // 同online
	} `json:"status"`
	Interval int64 `json:"interval"` // 元事件心跳间隔，单位ms
}

type NoticeEvent struct {
	Event
	NoticeType string `json:"notice_type"` // 通知类型，group, private
}

// 群文件上传通知
type GroupUploadNoticeEvent struct {
	NoticeEvent
	GroupId int64 `json:"group_id"` // 群号
	UserId  int64 `json:"user_id"`  // 上传者的QQ号
	File    struct {
		Id    string `json:"id"`     // 文件 ID
		Name  string `json:"name"`   // 文件名
		Size  int64  `json:"size"`   // 文件大小
		BusId string `json:"bus_id"` // 文件公众号 ID
	} `json:"file"`
}

// 群管理员变动通知
type GroupAdminNoticeEvent struct {
	NoticeEvent
	SubType string `json:"sub_type"` // 通知子类型，set unset
	GroupId int64  `json:"group_id"` // 群号
	UserId  int64  `json:"user_id"`  // 管理员 QQ 号
}

// 群成员增加通知
type GroupIncreaseNoticeEvent struct {
	NoticeEvent
	SubType    string `json:"sub_type"`    // 通知子类型，approve, invite
	GroupId    int64  `json:"group_id"`    // 群号
	UserId     int64  `json:"user_id"`     // 新成员 QQ 号
	OperatorId int64  `json:"operator_id"` // 操作者 QQ 号
}

// 群成员减少通知
type GroupDecreaseNoticeEvent struct {
	NoticeEvent
	SubType    string `json:"sub_type"`    // 通知子类型，leave, kick, kick_me
	GroupId    int64  `json:"group_id"`    // 群号
	UserId     int64  `json:"user_id"`     // 离开者 QQ 号
	OperatorId int64  `json:"operator_id"` // 操作者 QQ 号
}

// 群禁言通知
type GroupBanNoticeEvent struct {
	NoticeEvent
	SubType    string `json:"sub_type"`    // 通知子类型，ban, lift_ban
	GroupId    int64  `json:"group_id"`    // 群号
	UserId     int64  `json:"user_id"`     // 被禁言 QQ 号
	OperatorId int64  `json:"operator_id"` // 操作者 QQ 号
	Duration   int64  `json:"duration"`    // 禁言时长，单位秒
}

// 好友添加通知
type FriendAddNoticeEvent struct {
	NoticeEvent
	UserId int64 `json:"user_id"` // 好友 QQ 号
}

// 群消息撤回通知
type GroupRecallNoticeEvent struct {
	NoticeEvent
	GroupId    int64 `json:"group_id"`    // 群号
	UserId     int64 `json:"user_id"`     // 撤回者 QQ 号
	OperatorId int64 `json:"operator_id"` // 操作者 QQ 号
	MessageId  int64 `json:"message_id"`  // 消息 ID
}

// 好友消息撤回通知
type FriendRecallNoticeEvent struct {
	NoticeEvent
	UserId    int64 `json:"user_id"`    // 撤回者 QQ 号
	MessageId int64 `json:"message_id"` // 消息 ID
}

// 戳一戳通知
type PokeNoticeEvent struct {
	NoticeEvent
	SubType  string `json:"sub_type"`  // 通知子类型，poke
	GroupId  int64  `json:"group_id"`  // 群号
	UserId   int64  `json:"user_id"`   // 发送戳一戳的 QQ 号
	TargetId int64  `json:"target_id"` // 被戳一戳的 QQ 号
}

// 运气王通知
type LuckyKingNoticeEvent struct {
	NoticeEvent
	SubType  string `json:"sub_type"`  // 通知子类型，lucky_king
	GroupId  int64  `json:"group_id"`  // 群号
	UserId   int64  `json:"user_id"`   // 发红包者的 QQ 号
	TargetId int64  `json:"target_id"` // 运气王的 QQ 号
}

// 群成员荣誉变更
type HonorNoticeEvent struct {
	NoticeEvent
	SubType   string `json:"sub_type"`   // 通知子类型，honor
	GroupId   int64  `json:"group_id"`   // 群号
	UserId    int64  `json:"user_id"`    // QQ 号
	HonorType string `json:"honor_type"` // 荣誉类型，talkative、performer、emotion，分别表示龙王、群聊之火、快乐源泉
}

// 加好友请求事件
type FriendRequestEvent struct {
	Event
	RequestType string `json:"request_type"` // 请求类型，friend
	UserId      int64  `json:"user_id"`      // 发送请求的QQ号
	Comment     string `json:"comment"`      // 验证消息
	Flag        string `json:"flag"`         // 请求 flag，在调用处理请求的 API 时需要传入
}

// 加群请求事件
type GroupRequestEvent struct {
	Event
	RequestType string `json:"request_type"` // 请求类型，group
	SubType     string `json:"sub_type"`     // 请求子类型，add、invite，分别表示加群请求、邀请登录号入群
	GroupId     int64  `json:"group_id"`     // 群号
	UserId      int64  `json:"user_id"`      // 发送请求的QQ号
	Comment     string `json:"comment"`      // 验证消息
	Flag        string `json:"flag"`         // 请求请求 flag，在调用处理请求的 API 时需要传入标识
}