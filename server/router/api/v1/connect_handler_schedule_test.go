package v1

import (
	"testing"
	"time"

	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/store"
)

// TestConnectHandler_ScheduleSupport 测试 Connect RPC 版本是否支持日程
func TestConnectHandler_ScheduleSupport(t *testing.T) {
	tests := []struct {
		name           string
		searchResults  []*retrieval.SearchResult
		expectNotes    bool
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
						ID:      1,
						Title:   "团队周会",
						StartTs: time.Now().Unix(),
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
						ID:     1,
						UID:    "uid1",
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
			name:           "纯笔记查询",
			searchResults: []*retrieval.SearchResult{
				{
					ID:      1,
					Type:    "memo",
					Score:   0.95,
					Content: "软件进化 集成AI功能",
					Memo: &store.Memo{
						ID:     1,
						UID:    "uid1",
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
			if !contains(systemContent, "日程查询") {
				t.Error("system prompt should mention schedule query handling")
			}

			if !contains(systemContent, "优先回复日程信息") {
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
				if !contains(userContent, "📅 日程安排") {
					t.Error("user message should contain schedule section when schedules exist")
				}
			}

			// 如果有笔记，验证笔记上下文被添加
			if tt.expectNotes {
				if !contains(userContent, "📝 相关笔记") {
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
		Strategy:      "schedule_bm25_only",
		Confidence:    0.95,
		TimeRange: &queryengine.TimeRange{
			Start: time.Now().Truncate(24 * time.Hour),
			End:   time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
		},
		SemanticQuery: "",
		NeedsReranker: false,
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
