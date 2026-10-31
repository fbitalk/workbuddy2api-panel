// empty_content.go 空 content 归一化：上游对「message has empty content」极不宽容。
//
// 实测（2026-10-31，CN /v2，两模型）：
//   - user 消息 content:null → deepseek-v4.1-flash 直接挂起（无响应超时），
//     kimi-k3.1-pro 返回 api_error「有收容为空的訊息，模型無法處理」（503）；
//   - 同样位置改成单个空格 " " → 两模型均立即 200。
// 客户端（ZCode/Claude Code 类 CLI）长会话历史里会混入 content:null 的消息
// （被中断的响应、工具调用轮），一条就够让整单挂死——用户侧表现为无限
// 「重新连接中」。这里把所有空 content 归一化为单个空格（语义中性，上游放行）。
//
// 规则：
//   - content:null 且带 tool_calls → ""（OpenAI 惯例，实测放行）；
//   - content:null（无 tool_calls）→ " "；
//   - content 为字符串且去空白后为空 → " "；
//   - content 为数组且所有 text part 均为空 → " "。
package upstream

import "strings"

func normalizeEmptyContent(msgs []any) {
	for _, raw := range msgs {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		content, has := m["content"]
		if !has {
			continue
		}
		_, hasToolCalls := m["tool_calls"]
		switch c := content.(type) {
		case nil:
			if hasToolCalls {
				m["content"] = ""
			} else {
				m["content"] = " "
			}
		case string:
			if strings.TrimSpace(c) == "" && !hasToolCalls {
				m["content"] = " "
			}
		case []any:
			anyMeaningful := false
			for _, part := range c {
				pm, ok := part.(map[string]any)
				if !ok {
					anyMeaningful = true
					break
				}
				if t, _ := pm["type"].(string); t != "text" {
					anyMeaningful = true
					break
				}
				if s, _ := pm["text"].(string); strings.TrimSpace(s) != "" {
					anyMeaningful = true
					break
				}
			}
			if len(c) > 0 && !anyMeaningful {
				m["content"] = " "
			}
		}
	}
}
