# Code Audit Report - Loki Mode Cycle 1

## Summary
- Files reviewed: 55
- Total issues found: 44

| Category | P0 | P1 | P2 | P3 | Total |
|----------|----|----|----|----|-------|
| Go Backend | 2 | 2 | 4 | 2 | 10 |
| TypeScript | 3 | 2 | 4 | 3 | 12 |
| i18n | 0 | 1 | 0 | 0 | 1 (227+170 keys) |

---

## P0 - Critical (5 issues)

### Go Backend

#### 1. prompts.go:532-564 - Race condition in global `metricsRegistry` map
- **File**: `plugin/ai/agent/prompts.go`
- **Issue**: `metricsRegistry` map accessed concurrently without synchronization
- **Fix**: Add `sync.RWMutex` or use `sync.Map`

#### 2. ai_service.go:70-94 - Improper sync.Once usage
- **File**: `server/router/api/v1/ai_service.go`
- **Issue**: `sync.Once` with nil check inside Do function
- **Fix**: Use mutex with double-check locking

### TypeScript

#### 3. PartnerGreeting.tsx:187-196 - Memory leak: setTimeout not cleaned up
- **File**: `web/src/components/AIChat/PartnerGreeting.tsx`
- **Issue**: setTimeout without cleanup
- **Fix**: Return cleanup function or use useRef

#### 4. AIChatContext.tsx:676-717 - Fire-and-forget async in setState
- **File**: `web/src/contexts/AIChatContext.tsx`
- **Issue**: Nested async setState without proper dependency tracking
- **Fix**: Use useEffect with dependencies or abort controller

#### 5. AIChatContext.tsx:722-774 - Same fire-and-forget issue
- **File**: `web/src/contexts/AIChatContext.tsx`
- **Issue**: Same as #4

---

## P1 - High (5 issues)

### Go Backend

#### 6. prompts.go:128-367 - Race condition in `PromptRegistry`
- **File**: `plugin/ai/agent/prompts.go`
- **Issue**: Global `PromptRegistry` accessed concurrently without mutex
- **Fix**: Add `sync.RWMutex` to `PromptRegistry`

#### 7. scheduler_v2.go:259-290 - Closure captures loop variable
- **File**: `plugin/ai/agent/scheduler_v2.go`
- **Issue**: `ctx` captured in closure, may be cancelled before callback fires
- **Fix**: Create detached context with timeout

### TypeScript

#### 8. MemoPreview.tsx:15-19 - State never reset on error
- **File**: `web/src/components/ScheduleAI/MemoPreview.tsx`
- **Issue**: `isCreating` stays true if onConfirm fails
- **Fix**: Wrap in try-catch

#### 9. AIChat.tsx:240-349 - Missing dependencies in useCallback
- **File**: `web/src/pages/AIChat.tsx`
- **Issue**: `uiTools` used but not in dependency array
- **Fix**: Add `uiTools` to dependency array

### i18n

#### 10. Missing translations in zh-Hant.json
- **Issue**: 227 keys missing, 170 orphaned keys
- **Fix**: Sync translations, remove orphaned keys

---

## P2 - Medium (8 issues)

### Go Backend

#### 11. router/service.go:92-98 - Goroutine leak potential
#### 12. memo_parrot.go:205-216 - Context not respected during streaming
#### 13. amazing_parrot.go:275-390 - Potential double-lock
#### 14. schedule_agent_service.go:153 - ContextStore not concurrent-safe

### TypeScript

#### 15. GenerativeUIContainer.tsx:29-49 - useEffect dependency issue
#### 16. ScheduleQuickInput.tsx:185 - onUIEvent dependency issue
#### 17. AIChatContext.tsx:881-883 - Empty dependency array with stale closure
#### 18. useScheduleAgent.ts:147-150 - Type assertion without validation

---

## P3 - Low (5 issues)

### Go Backend

#### 19. prompts.go:494-496 - Global variable without synchronization
#### 20. history_matcher.go:113-156 - No context deadline check in loop

### TypeScript

#### 21. MetricsDashboard.tsx:16-27 - Race condition on rapid changes
#### 22. useAITools.ts:25-89 - Console.log in production
#### 23. PartnerGreeting.tsx:165-173 - Unnecessary dependency on `t`

---

## Next Steps
1. Fix P0 issues first (5)
2. Fix P1 issues (5)
3. Fix P2 issues (8)
4. Fix P3 issues (5)
5. Fix i18n issues (227 missing + 170 orphaned)
6. Re-run audit
