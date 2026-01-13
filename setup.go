package carryfallback

import (
    "github.com/coredns/coredns/plugin"
    "github.com/coredns/coredns/plugin/pkg/fall"
    "github.com/miekg/dns"
    "github.com/coredns/coredns/core/dnsserver"
    "github.com/coredns/coredns/plugin/file"
    "github.com/coredns/coredns/plugin/rewrite"
    "strings"

    "github.com/caddyserver/caddy"
)

func init() {
    plugin.Register("carryfallback", setup)
}

func setup(c *caddy.Controller) error {
    c.Next() // 跳过插件名

    // 创建插件实例
    f := &Fallback{}

    // 获取配置参数
    for c.NextBlock() {
        switch c.Val() {
        case "suffix":
            if !c.NextArg() {
                return c.ArgErr()
            }
            f.Suffix = c.Val()
        case "zone":
            if !c.NextArg() {
                return c.ArgErr()
            }
            f.Zone = c.Val()
        }
    }

    // 如果没有指定后缀，默认使用"-carry"
    if f.Suffix == "" {
        f.Suffix = "-carry"
    }

    // 添加到DNS服务器配置
    dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
        f.Next = next
        return f
    })

    return nil
}
