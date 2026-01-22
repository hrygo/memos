package v1

import (
	"encoding/json"
	"log/slog"
	"strings"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

// ParseScheduleIntentFromAIResponse parses schedule creation intent from AI's response text.
// Marker format: <<<SCHEDULE_INTENT:{"detected":true,"schedule_description":"..."}>>>
//
// This is a shared utility function used by both gRPC and Connect handlers.
func ParseScheduleIntentFromAIResponse(aiResponse string) *v1pb.ScheduleCreationIntent {
	// 查找意图标记：使用独特的 <<<SCHEDULE_INTENT: 格式避免误判
	const intentMarker = "<<<SCHEDULE_INTENT:"

	startIdx := strings.Index(aiResponse, intentMarker)
	if startIdx == -1 {
		// 没有意图标记，用户没有创建日程的意图
		return nil
	}

	// 提取 JSON 部分
	startIdx += len(intentMarker)

	// 查找结束标记 >>>（使用 LastIndex 避免描述中的 >>> 截断）
	endIdx := strings.LastIndex(aiResponse[startIdx:], ">>>")
	if endIdx == -1 {
		slog.Debug("ScheduleIntent marker found but missing closing '>>>'")
		return nil
	}

	jsonStr := strings.TrimSpace(aiResponse[startIdx : startIdx+endIdx])

	// 清理 JSON 字符串：移除换行符和制表符，但保留空格（description 中可能包含空格）
	cleanJSON := strings.ReplaceAll(jsonStr, "\n", "")
	cleanJSON = strings.ReplaceAll(cleanJSON, "\t", "")
	cleanJSON = strings.TrimSpace(cleanJSON)

	// 解析 JSON
	type IntentJSON struct {
		Detected            bool   `json:"detected"`
		ScheduleDescription string `json:"schedule_description"` // 正确的字段名
		Description         string `json:"description"`          // 兼容旧字段名
	}

	var intentJSON IntentJSON
	if err := json.Unmarshal([]byte(cleanJSON), &intentJSON); err != nil {
		slog.Debug("Failed to parse schedule intent JSON", "error", err, "original", jsonStr, "cleaned", cleanJSON)
		return nil
	}

	// 检查是否检测到意图
	if !intentJSON.Detected {
		return nil
	}

	// 获取描述（优先使用正确的字段名，兼容旧字段名）
	description := intentJSON.ScheduleDescription
	if description == "" {
		description = intentJSON.Description // 兼容旧格式
	}

	// 验证描述不为空
	if strings.TrimSpace(description) == "" {
		slog.Debug("ScheduleIntent detected but description is empty")
		return nil
	}

	// 构建返回对象
	intent := &v1pb.ScheduleCreationIntent{
		Detected:            true,
		ScheduleDescription: description,
	}

	// 记录成功解析
	slog.Debug("ScheduleIntent successfully parsed", "description", description)

	return intent
}
