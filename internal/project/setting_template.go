package project

import (
	"fmt"
	"strings"
)

// SanitizeSettingTitle 清理设定文档标题（不含 .md 后缀）。
func SanitizeSettingTitle(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".md")
	name = strings.TrimSuffix(name, ".MD")
	replacer := strings.NewReplacer("/", "", "\\", "", ":", "：", "..", "", "*", "", "?", "？", "\"", "", "<", "", ">", "", "|", "")
	name = replacer.Replace(name)
	return strings.TrimSpace(name)
}

// SettingBodyTemplate 生成新设定 Markdown 正文。
func SettingBodyTemplate(subdir, title, templateKind string, meta Meta) string {
	title = SanitizeSettingTitle(title)
	header := fmtSettingHeader(title, meta)
	kind := strings.TrimSpace(templateKind)
	primary := settingPrimaryValue(title, meta)

	switch subdir {
	case SettingsSubCharacter:
		switch kind {
		case "villain", "反派":
			return header + fmt.Sprintf("## 姓名\n%s\n\n## 动机\n\n## 能力\n\n## 与主角关系\n\n## 威胁等级\n", primary)
		case "blank", "空白":
			return header
		default:
			body := fmt.Sprintf("## 姓名\n%s\n\n## 性格\n\n## 背景\n\n## 目标\n\n", primary)
			if title == "主角卡" || strings.Contains(title, "主角") {
				body += "## 成长弧线\n"
			}
			return header + body
		}
	case SettingsSubWorld:
		return header + "## 概述\n\n## 核心规则\n\n## 待补充\n"
	case SettingsSubFaction:
		return header + fmt.Sprintf("## 名称\n%s\n\n## 立场\n\n## 核心人物\n\n## 与主角关系\n", primary)
	case SettingsSubLocation:
		return header + fmt.Sprintf("## 位置\n%s\n\n## 环境特征\n\n## 剧情作用\n", primary)
	case SettingsSubItem:
		return header + fmt.Sprintf("## 名称\n%s\n\n## 外观\n\n## 能力/用途\n\n## 归属\n", primary)
	default:
		return header + "## 待补充\n"
	}
}

// settingPrimaryValue 从文档标题推断首字段默认值（姓名/名称等）。
func settingPrimaryValue(title string, meta Meta) string {
	title = SanitizeSettingTitle(title)
	switch title {
	case "主角卡":
		return strings.TrimSpace(meta.Protagonist)
	case "反派设计", "世界观", "力量体系", "科技体系", "金手指", "势力关系", "爽点规划", "总纲":
		return ""
	}
	if i := strings.LastIndex(title, "-"); i >= 0 {
		if v := strings.TrimSpace(title[i+1:]); v != "" {
			return v
		}
	}
	if i := strings.LastIndex(title, "·"); i >= 0 {
		if v := strings.TrimSpace(title[i+len("·"):]); v != "" {
			return v
		}
	}
	return title
}

func fmtSettingHeader(title string, meta Meta) string {
	return fmt.Sprintf("# %s\n\n> 书名：%s | 题材：%s | 风格：%s\n\n",
		title, meta.Title, meta.Genre, meta.WritingStyle())
}
