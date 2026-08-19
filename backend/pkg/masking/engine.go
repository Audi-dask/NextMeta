package masking

import (
	"regexp"
	"strings"
)

/*
Engine 负责查询结果数据脱敏。
它根据数据源配置的脱敏规则，对命中的字段值执行不同类型的字符串遮盖。
*/
type Engine struct{}

/*
NewEngine 创建脱敏引擎。
当前脱敏逻辑无内部状态，每次创建成本很低。
*/
func NewEngine() *Engine {
	return &Engine{}
}

/*
Mask 按脱敏类型处理单个字符串值。
不认识的 ruleType 会原样返回，避免错误配置导致数据被意外清空。
*/
func (e *Engine) Mask(value string, ruleType string) string {
	if value == "" {
		return ""
	}

	length := len(value)
	if length <= 1 {
		return "*"
	}

	switch ruleType {
	case "mask_middle":
		// 中间脱敏，例如 13812345678 -> 138****5678；长度过短时全部脱敏。
		if length < 4 {
			return strings.Repeat("*", length)
		}
		start := length / 3
		end := length - (length / 3)
		return value[:start] + strings.Repeat("*", end-start) + value[end:]

	case "mask_all":
		// 全部脱敏，统一返回固定长度星号。
		return "******"

	case "mask_left":
		// 左半部分脱敏，保留右半部分。
		half := length / 2
		return strings.Repeat("*", half) + value[half:]

	case "mask_right":
		// 右半部分脱敏，保留左半部分。
		half := length / 2
		return value[:half] + strings.Repeat("*", length-half)

	default:
		return value
	}
}

/*
ShouldMask 判断列名是否命中脱敏规则 pattern。
当前支持精确匹配和包含 * 的简单通配符匹配，比较过程不区分大小写。
*/
func (e *Engine) ShouldMask(columnName string, pattern string) bool {
	// 统一转小写，保证匹配不受字段大小写影响。
	columnName = strings.ToLower(columnName)
	pattern = strings.ToLower(pattern)

	if strings.Contains(pattern, "*") {
		// 将简单通配符转换为正则，例如 *phone* -> .*phone.*。
		regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
		matched, err := regexp.MatchString(regexPattern, columnName)
		if err == nil && matched {
			return true
		}
		return false
	}

	// 不包含通配符时按完整列名精确匹配。
	return columnName == pattern
}
