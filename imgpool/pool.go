// Package imgpool 图片缓存池
package imgpool

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/FloatTech/zbputils/control"
	"github.com/FloatTech/zbputils/pool"
	"github.com/FloatTech/zbputils/process"
	"github.com/sirupsen/logrus"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const cacheurl = "https://gchat.qpic.cn/gchatpic_new//%s/0"
const imgpoolgrp = 117500479

var pushkey string

type Image struct {
	img *pool.Item
	f   string
}

// GetImage context name
func GetImage(ctx *zero.Ctx, name string) (m Image, err error) {
	m.img, err = pool.GetItem(name)
	if err == nil && m.img.String() != "" {
		_, err = http.Head(m.String())
		if err != nil {
			err = errors.New("img file outdated")
			return
		}
		return
	}
	err = errors.New("no such img")
	return
}

// NewImage context name file
func NewImage(ctx *zero.Ctx, name, f string) (m Image, err error) {
	if strings.HasPrefix(f, "http") {
		m.f = f
	} else {
		m.f = "file:///" + f
	}
	m.img, err = pool.GetItem(name)
	if err == nil && m.img.String() != "" {
		_, err = http.Head(m.String())
		if err == nil {
			return
		}
	}
	id := ctx.SendGroupMessage(imgpoolgrp, message.Message{message.Text(name), message.Image(m.f)})
	if id == 0 {
		err = errors.New("send image error")
		return
	}
	defer process.SleepAbout1sTo2s() // 防止风控
	msg := ctx.GetMessage(message.NewMessageID(strconv.FormatInt(id, 10)))
	for _, e := range msg.Elements {
		if e.Type == "image" {
			u := e.Data["url"]
			u = u[:strings.LastIndex(u, "/")]
			u = u[strings.LastIndex(u, "/")+1:]
			if u != "" {
				m.img, err = pool.NewItem(name, u)
				logrus.Infoln("[imgpool] 缓存:", name, "url:", u)
				if pushkey != "" {
					_ = m.img.Push(pushkey)
				}
			} else {
				err = errors.New("get msg error")
			}
			return
		}
	}
	err = errors.New("get msg error")
	return
}

// RegisterListener key engine
func RegisterListener(key string, en control.Engine) {
	pushkey = key
	en.OnMessage(zero.OnlyGroup, func(ctx *zero.Ctx) bool {
		return ctx.Event.GroupID == imgpoolgrp
	}).SetBlock(true).
		Handle(func(ctx *zero.Ctx) {
			var u, n string
			for _, e := range ctx.Event.Message {
				if e.Type == "image" {
					u = e.Data["url"]
					u = u[:strings.LastIndex(u, "/")]
					u = u[strings.LastIndex(u, "/")+1:]
				} else if e.Type == "text" {
					n = e.Data["text"]
				}
			}
			if u == "" || n == "" {
				return
			}
			img, err := pool.NewItem(n, u)
			if err != nil {
				ctx.SendChain(message.Text("ERROR:", err))
				return
			}
			err = img.Push(key)
			logrus.Infoln("[imgpool] 推送缓存:", n, "url:", u)
			if err != nil {
				ctx.SendChain(message.Text("ERROR:", err))
				return
			}
		})
}

// String url
func (m Image) String() string {
	return fmt.Sprintf(cacheurl, m.img.String())
}
