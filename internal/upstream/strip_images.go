// strip_images.go 剥离消息里的 base64 图片（data: URL）。
//
// 实测（2026-10-31，CN /v2，kimi-k3.1-pro）：user 消息携带 144KB base64 截图
// 时，上游返回 400 code=11133 "Invalid request parameters"，且对历史长度
// 敏感（前缀 89 条正常、90 条即失败）。同位置换成文本占位后立即恢复。
// deepseek 对真实图片的行为未验证（日志里的 URL 已被客户端脱敏，无法重放）。
//
// 策略：只剥离 data: URL 的图片（本通道的图片输入形态），保留同消息的
// 文本，并替换为占位文本让模型知道该处曾有媒体。远程 http(s) 图片 URL
// 不在剥离范围（未验证，保守保留）。
package upstream

import (
	"log"
)

func stripDataURLImages(obj map[string]any) {
	msgs, ok := obj["messages"].([]any)
	if !ok {
		return
	}
	changed := false
	for _, raw := range msgs {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		parts, ok := m["content"].([]any)
		if !ok {
			continue
		}
		hasImage := false
		for _, p := range parts {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if pm["type"] == "image_url" {
				hasImage = true
			}
		}
		if !hasImage {
			continue
		}
		kept := make([]any, 0, len(parts))
		for _, p := range parts {
			pm, ok := p.(map[string]any)
			if !ok {
				kept = append(kept, p)
				continue
			}
			if pm["type"] == "image_url" {
				kept = append(kept, map[string]any{"type": "text", "text": "[图片已省略：当前模型通道不支持图片输入]"})
				changed = true
				continue
			}
			kept = append(kept, p)
		}
		m["content"] = kept
	}
	if changed {
		log.Println("[payload] 已剥离消息中的 base64 图片（上游通道不接受，替换为占位文本）")
	}
}
