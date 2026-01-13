package carryfallback

import (
    "context"
    "strings"
    
    "github.com/coredns/coredns/plugin"
    "github.com/coredns/coredns/request"
    "github.com/miekg/dns"
)

type Fallback struct {
    Next plugin.Handler
}

func (f Fallback) Name() string { 
    return "carryfallback" 
}

func (f Fallback) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
    state := request.Request{W: w, Req: r}
    
    // 1. 先尝试正常解析
    rc, err := plugin.NextOrFailure(f.Name(), f.Next, ctx, w, r)
    
    // 2. 如果解析失败，且域名包含"-carry"，尝试后备解析
    if (err != nil || rc == dns.RcodeNameError) {
        qname := state.Name()
        
        // 检查是否是service域名（包含.svc.）
        if strings.Contains(qname, ".svc.") && strings.Contains(qname, "-carry.") {
            // 去掉 -carry
            newQname := strings.Replace(qname, "-carry.", ".", 1)
            
            // 创建新的DNS查询
            m := new(dns.Msg)
            m.SetQuestion(newQname, state.QType())
            m.Id = r.Id
            
            // 创建简单的ResponseWriter包装器
            newW := &responseRecorder{ResponseWriter: w}
            
            // 重新解析
            _, err2 := plugin.NextOrFailure(f.Name(), f.Next, ctx, newW, m)
            
            // 如果后备解析成功，使用后备结果
            if err2 == nil && newW.msg != nil {
                newW.msg.Id = r.Id
                w.WriteMsg(newW.msg)
                return dns.RcodeSuccess, nil
            }
        }
    }
    
    return rc, err
}

// 简化的ResponseWriter包装器
type responseRecorder struct {
    dns.ResponseWriter
    msg *dns.Msg
}

func (rw *responseRecorder) WriteMsg(msg *dns.Msg) error {
    rw.msg = msg
    return nil
}
