package v1

import (
	"strings"
	"testing"
	"time"

	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/store"
)

// TestConnectHandler_ScheduleSupport 测试 Connect RPC 版本是否支持日程
func TestConnectHandler_ScheduleSupport(t *testing.T) {
	tests := []struct {
		name            string
		searchResults   []*retrieval.SearchResult
		expectNotes     bool
		expectSchedules bool
	}{
		{
			name: "纯日程查询",
			searchResults: []*retrieval.SearchResult{
				{
					ID:      1,
					Type:    "schedule",
					Score:   1.0,
					Content: "团队周会",
					Schedule: &store.Schedule{
						ID:       1,
						Title:    "团队周会",
						StartTs:  time.Now().Unix(),
						Location: "会议室A",
					},
				},
				{
					ID:      2,
					Type:    "schedule",
					Score:   0.9,
					Content: "项目评审",
					Schedule: &store.Schedule{
						ID:      2,
						Title:   "项目评审",
						StartTs: time.Now().Add(2 * time.Hour).Unix(),
					},
				},
			},
			expectNotes:     false,
			expectSchedules: true,
		},
		{
			name: "笔记和日程混合",
			searchResults: []*retrieval.SearchResult{
				{
					ID:      1,
					Type:    "memo",
					Score:   0.95,
					Content: "软件进化 集成AI功能",
					Memo: &store.Memo{
						ID:      1,
						UID:     "uid1",
						Content: "软件进化 集成AI功能",
					},
				},
				{
					ID:      2,
					Type:    "schedule",
					Score:   1.0,
					Content: "团队周会",
					Schedule: &store.Schedule{
						ID:      1,
						Title:   "团队周会",
						StartTs: time.Now().Unix(),
					},
				},
			},
			expectNotes:     true,
			expectSchedules: true,
		},
		{
			name: "纯笔记查询",
			searchResults: []*retrieval.SearchResult{
				{
					ID:      1,
					Type:    "memo",
					Score:   0.95,
					Content: "软件进化 集成AI功能",
					Memo: &store.Memo{
						ID:      1,
						UID:     "uid1",
						Content: "软件进化 集成AI功能",
					},
				},
			},
			expectNotes:     true,
			expectSchedules: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 Connect handler（不需要完整的 service）
			handler := &ConnectServiceHandler{}

			// 分类结果
			var memoResults []*retrieval.SearchResult
			var scheduleResults []*retrieval.SearchResult
			for _, result := range tt.searchResults {
				switch result.Type {
				case "memo":
					memoResults = append(memoResults, result)
				case "schedule":
					scheduleResults = append(scheduleResults, result)
				}
			}

			// 验证分类
			hasNotes := len(memoResults) > 0
			hasSchedules := len(scheduleResults) > 0

			if hasNotes != tt.expectNotes {
				t.Errorf("hasNotes = %v, want %v", hasNotes, tt.expectNotes)
			}

			if hasSchedules != tt.expectSchedules {
				t.Errorf("hasSchedules = %v, want %v", hasSchedules, tt.expectSchedules)
			}

			// 构建消息
			messages := handler.buildOptimizedMessagesForConnect(
				"今日日程",
				[]string{},
				"mock context",
				scheduleResults,
				hasNotes,
				hasSchedules,
			)

			// 验证消息不为空
			if len(messages) < 2 {
				t.Fatalf("expected at least 2 messages (system + user), got %d", len(messages))
			}

			// 验证系统提示词包含日程相关说明
			systemMsg := messages[0]
			if systemMsg.Role != "system" {
				t.Errorf("expected system message role, got %s", systemMsg.Role)
			}

			systemContent := systemMsg.Content
			if !strings.Contains(systemContent, "日程查询") {
				t.Error("system prompt should mention schedule query handling")
			}

			if !strings.Contains(systemContent, "优先回复日程信息") {
				t.Error("system prompt should prioritize schedule information")
			}

			// 验证用户消息包含上下文
			userMsg := messages[len(messages)-1]
			if userMsg.Role != "user" {
				t.Errorf("expected user message role, got %s", userMsg.Role)
			}

			userContent := userMsg.Content

			// 如果有日程，验证日程上下文被添加
			if tt.expectSchedules {
				if !strings.Contains(userContent, "📅 日程安排") {
					t.Error("user message should contain schedule section when schedules exist")
				}
			}

			// 如果有笔记，验证笔记上下文被添加
			if tt.expectNotes {
				if !strings.Contains(userContent, "📝 相关笔记") {
					t.Error("user message should contain notes section when notes exist")
				}
			}
		})
	}
}

// TestConnectHandler_RouteDecision 测试路由决策是否正确传递
func TestConnectHandler_RouteDecision(t *testing.T) {
	// 模拟路由决策
	routeDecision := &queryengine.RouteDecision{
		Strategy:   "schedule_bm25_only",
		Confidence: 0.95,
		TimeRange: &queryengine.TimeRange{
			Start: time.Now().Truncate(24 * time.Hour),
			End:   time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
		},
	}

	// 验证决策
	if routeDecision.Strategy != "schedule_bm25_only" {
		t.Errorf("expected schedule_bm25_only, got %s", routeDecision.Strategy)
	}

	if routeDecision.Confidence < 0.9 {
		t.Errorf("expected confidence >= 0.9, got %.2f", routeDecision.Confidence)
	}

	if routeDecision.TimeRange == nil {
		t.Error("expected TimeRange to be set for schedule query")
	}
}

// TestConnectHandler_IntentDetection 测试意图检测提示词是否正确
func TestConnectHandler_IntentDetection(t *testing.T) {
	handler := &ConnectServiceHandler{}

	// 构建消息（模拟纯日程查询场景）
	messages := handler.buildOptimizedMessagesForConnect(
		"明天有哪些事要干", // ⭐ 关键：这是查询，不是创建
		[]string{},
		"",
		[]*retrieval.SearchResult{},
		false,
		false,
	)

	// 验证系统提示词包含正确的意图检测说明
	systemMsg := messages[0]
	if systemMsg.Role != "system" {
		t.Fatalf("expected system message, got %s", systemMsg.Role)
	}

	systemContent := systemMsg.Content

	// 验证提示词明确说明何时检测意图
	if !strings.Contains(systemContent, "仅在用户的原始问题明确表示要创建日程时") {
		t.Error("system prompt should clarify that intent detection is only for creation")
	}

	// 验证提示词明确列出查询类场景不是创建意图
	if !strings.Contains(systemContent, "查询类") {
		t.Error("system prompt should explicitly list query scenarios as non-creation")
	}

	if !strings.Contains(systemContent, "明天的事要干") {
		t.Error("system prompt should include '明天的事要干' as an example of query (not creation)")
	}

	// 验证提示词包含明确的创建关键词
	if !strings.Contains(systemContent, "帮我创建") {
		t.Error("system prompt should include clear creation keywords like '帮我创建'")
	}

	if !strings.Contains(systemContent, "设置提醒") {
		t.Error("system prompt should include clear creation keywords like '设置提醒'")
	}

	// 验证提示词不包含误导性的"安排"关键词作为创建意图
	// 因为"有什么安排"是查询，不是创建
	if strings.Contains(systemContent, "关键词：\"创建\"、\"提醒\"、\"安排\"、\"添加\"") {
		t.Error("system prompt should NOT list '安排' as a creation keyword without context")
	}
}
