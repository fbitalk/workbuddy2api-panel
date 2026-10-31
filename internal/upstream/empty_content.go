// empty_content.go 空 content 归一化：上游对「message has empty content」极不宽容。
//
// 实测（2026-10-31，CN /v2，两模型）：
//   - user 消息 content:null → deepseek-v4.1-flash 直接挂起（无响应超时），
//     kimi-k3.1-pro 返回 api_error「有收容为空的訊息，模型無法處理」（503）；
//   - 同样位置改成单个空格 " " → 小请求两模型均 200，但长历史里上游仍按
//     空内容拒绝（空白字符不算非空），占位符必须是真实非空白文本。
// 客户端（ZCode/Claude Code 类 CLI）长会话历史里会混入 content:null 的消息
// （被中断的响应、工具调用轮），一条就够让整单挂死——用户侧表现为无限
// 「重新连接中」。这里把所有空 content 归一化为单个空格（语义中性，上游放行）。
//
// 规则：
//   - content:null 且带 tool_calls → ""（OpenAI 惯例，实测放行）；
//   - content:null（无 tool_calls）→ "."；
//   - content 为字符串且去空白后为空 → "."；
//   - content 为数组且所有 text part 均为空 → " "。
package upstream

import (
	"regexp"
	"strings"
)

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
			m["content"] = "（此消息无正文）"
		case string:
			if strings.TrimSpace(c) == "" && !hasToolCalls {
				m["content"] = "（此消息无正文）"
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
				m["content"] = "（此消息无正文）"
			}
		}
	}
}


// ensureVisibleContent 消息内容只有符号、没有实际文字（字母/数字/中日韩）时，
// 上游同样按「message has empty content」拒绝（实测 tool 结果为 "[]" / "=> []"
// 即触发 11151）。替换为带文字的占位。
var visibleTextRe = regexp.MustCompile(`[0-9A-Za-z一-鿿ぁ-ヿ가-힯]`)

func ensureVisibleContent(msgs []any) {
	for _, raw := range msgs {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		c, ok := m["content"].(string)
		if !ok || c == "" || visibleTextRe.MatchString(c) {
			continue
		}
		m["content"] = "（此消息无正文）"
	}
}