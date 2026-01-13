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
    qname := state.Name()
    
    // 1. 先尝试正常解析
    recorder := &responseRecorder{ResponseWriter: w}
    rc, err := plugin.NextOrFailure(f.Name(), f.Next, ctx, recorder, r)
    
    // 2. 如果解析失败，且域名包含"-carry"，尝试后备解析
    if (err != nil || rc == dns.RcodeNameError) && 
       strings.Contains(qname, ".svc.") && 
       strings.Contains(qname, "-carry.") {
        
        // 去掉 -carry
        newName := strings.Replace(qname, "-carry.", ".", 1)
        
        // 创建新查询
        newR := r.Copy()
        newR.Question[0] = dns.Question{
            Name:   newName,
            Qtype:  state.QType(),
            Qclass: state.QClass(),
        }
        
        // 尝试后备解析
        newRecorder := &responseRecorder{ResponseWriter: w}
        rc2, _ := plugin.NextOrFailure(f.Name(), f.Next, ctx, newRecorder, newR)
        
        // 如果后备解析成功，使用后备结果
        if rc2 != dns.RcodeNameError && newRecorder.msg != nil {
            newRecorder.msg.Id = r.Id
            w.WriteMsg(newRecorder.msg)
            return rc2, nil
        }
    }
    
    // 3. 返回原始结果
    if recorder.msg != nil {
        w.WriteMsg(recorder.msg)
    }
    return rc, err
}

// 简单的ResponseWriter包装器
type responseRecorder struct {
    dns.ResponseWriter
    msg *dns.Msg
}

func (r *responseRecorder) WriteMsg(m *dns.Msg) error {
    r.msg = m
    return nil
}
