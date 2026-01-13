package carryfallback

import (
    "context"
    "fmt"
    "strings"

    "github.com/coredns/coredns/plugin"
    "github.com/coredns/coredns/request"

    "github.com/miekg/dns"
)

// Fallback 实现carry后备解析插件
type Fallback struct {
    Next   plugin.Handler
    Suffix string
    Zone   string
}

// Name 返回插件名称
func (f *Fallback) Name() string { return "carryfallback" }

// ServeDNS 处理DNS请求
func (f *Fallback) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
    state := request.Request{W: w, Req: r}

    // 只处理A和AAAA记录查询
    qtype := state.QType()
    if qtype != dns.TypeA && qtype != dns.TypeAAAA {
        return plugin.NextOrFailure(f.Name(), f.Next, ctx, w, r)
    }

    // 获取查询的完整域名
    qname := state.Name()

    // 检查域名是否在我们处理的zone内
    if f.Zone != "" && !strings.HasSuffix(qname, f.Zone+".") {
        return plugin.NextOrFailure(f.Name(), f.Next, ctx, w, r)
    }

    // 1. 先尝试标准解析
    rc, err := plugin.NextOrFailure(f.Name(), f.Next, ctx, w, r)

    // 2. 如果解析成功，直接返回
    if err == nil && rc != dns.RcodeNameError {
        return rc, err
    }

    // 3. 如果解析失败（NXDOMAIN）且域名包含指定后缀
    if (err != nil || rc == dns.RcodeNameError) && strings.Contains(qname, f.Suffix) {
        // 去掉后缀
        newQname := strings.Replace(qname, f.Suffix+".", ".", 1)

        // 创建新的DNS消息
        m := new(dns.Msg)
        m.SetQuestion(newQname, qtype)
        m.Question[0].Qclass = state.QClass()

        // 克隆原始消息的其他部分
        m.Id = r.Id
        m.RecursionDesired = r.RecursionDesired
        m.CheckingDisabled = r.CheckingDisabled

        // 使用新的查询进行解析
        writer := &ResponseWriter{ResponseWriter: w, original: r}

        // 重新解析
        rc, err = plugin.NextOrFailure(f.Name(), f.Next, ctx, writer, m)

        // 如果后备解析成功，返回后备结果
        if err == nil && rc != dns.RcodeNameError {
            // 修改响应ID以匹配原始请求
            if writer.msg != nil {
                writer.msg.Id = r.Id
                w.WriteMsg(writer.msg)
            }
            return rc, err
        }
    }

    // 4. 返回原始结果
    if err != nil {
        return rc, err
    }

    // 如果到这里，说明后备也失败了，返回原始错误
    m := new(dns.Msg)
    m.SetRcode(r, rc)
    w.WriteMsg(m)
    return rc, nil
}

// ResponseWriter 包装原始ResponseWriter以捕获响应
type ResponseWriter struct {
    dns.ResponseWriter
    original *dns.Msg
    msg      *dns.Msg
}

// WriteMsg 覆盖WriteMsg以捕获响应
func (w *ResponseWriter) WriteMsg(msg *dns.Msg) error {
    w.msg = msg
    return nil
}
