# P1-C003: 相关笔记推荐

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 6 人天  
> **负责团队**: 团队 C  
> **Sprint**: Sprint 2

---

## 1. 目标与背景

### 1.1 核心目标

基于语义相似度和标签共现，为当前笔记推荐相关笔记，帮助用户发现知识关联。

### 1.2 用户价值

- 笔记关联发现率从 5% 提升至 20%
- 减少重复记录

### 1.3 技术价值

- 复用向量检索能力
- 为知识图谱奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- [ ] P1-A005: 缓存层（用于缓存推荐结果）

### 2.2 并行依赖

- P1-C001/C002: 搜索高亮（可复用部分逻辑）

### 2.3 后续依赖

- P2-C002: 重复检测系统
- P3-C001: 知识图谱可视化

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    相关推荐架构                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  触发场景:                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ 编辑器输入   │  │ 笔记详情页   │  │ 保存笔记后   │          │
│  │ (防抖500ms) │  │ (侧边栏)     │  │ (后台推送)   │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
│         └────────────────┬┴─────────────────┘                   │
│                          │                                       │
│                          ▼                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  RelatedService                          │   │
│  │                                                          │   │
│  │  ┌───────────────────────────────────────────────────┐  │   │
│  │  │  1. 向量相似度检索 (权重 0.6)                       │  │   │
│  │  │     使用 pgvector cosine similarity               │  │   │
│  │  └───────────────────────────────────────────────────┘  │   │
│  │                          +                               │   │
│  │  ┌───────────────────────────────────────────────────┐  │   │
│  │  │  2. 标签共现计算 (权重 0.3)                         │  │   │
│  │  │     共享标签数 / 总标签数                          │  │   │
│  │  └───────────────────────────────────────────────────┘  │   │
│  │                          +                               │   │
│  │  ┌───────────────────────────────────────────────────┐  │   │
│  │  │  3. 时间邻近度 (权重 0.1)                           │  │   │
│  │  │     7天内得分高                                    │  │   │
│  │  └───────────────────────────────────────────────────┘  │   │
│  │                          │                               │   │
│  │                          ▼                               │   │
│  │  ┌───────────────────────────────────────────────────┐  │   │
│  │  │  加权融合 + 去重 + 排序 → Top-5                     │  │   │
│  │  └───────────────────────────────────────────────────┘  │   │
│  │                                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **向量检索**: 查找语义相似的笔记
2. **标签计算**: 计算标签共现分数
3. **时间计算**: 计算时间邻近分数
4. **融合排序**: 加权融合后取 Top-5

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| 相似度来源 | 仅向量 | 多维融合 | B | 更准确的推荐 |
| 触发时机 | 实时 | 防抖 | B | 减少计算压力 |

---

## 4. 技术实现

### 4.1 接口定义

```protobuf
// proto/api/v1/memo_service.proto

rpc GetRelatedMemos(GetRelatedMemosRequest) returns (GetRelatedMemosResponse);

message GetRelatedMemosRequest {
  string memo_name = 1;
  int32 limit = 2;           // default: 5
}

message GetRelatedMemosResponse {
  repeated RelatedMemo memos = 1;
}

message RelatedMemo {
  string name = 1;
  string title = 2;
  float similarity = 3;
  repeated string shared_tags = 4;
  int64 created_ts = 5;
}
```

### 4.2 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `server/service/memo/related.go` | 相关服务 |
| `server/router/api/v1/memo_service.go` | API 处理器 |
| `web/src/components/MemoRelated/RelatedList.tsx` | 前端组件 |

### 4.3 后端实现

```go
// server/service/memo/related.go

type RelatedService struct {
    embeddingStore *store.EmbeddingStore
    memoStore      *store.MemoStore
    cache          cache.CacheService
}

type RelatedMemo struct {
    Name       string   `json:"name"`
    Title      string   `json:"title"`
    Similarity float32  `json:"similarity"`
    SharedTags []string `json:"shared_tags"`
    CreatedTs  int64    `json:"created_ts"`
}

func (s *RelatedService) GetRelatedMemos(
    ctx context.Context,
    memoUID string,
    limit int,
) ([]RelatedMemo, error) {
    // 检查缓存
    cacheKey := fmt.Sprintf("related:%s", memoUID)
    if cached, ok := s.cache.Get(ctx, cacheKey); ok {
        var result []RelatedMemo
        json.Unmarshal(cached, &result)
        return result, nil
    }
    
    // 1. 获取当前笔记的向量
    currentEmb, err := s.embeddingStore.GetByMemoUID(ctx, memoUID)
    if err != nil {
        return nil, err
    }
    
    // 2. 向量相似度检索
    vectorResults, err := s.embeddingStore.FindSimilar(ctx, currentEmb.Vector, limit*2)
    if err != nil {
        return nil, err
    }
    
    // 3. 获取当前笔记标签
    currentMemo, _ := s.memoStore.GetByUID(ctx, memoUID)
    currentTags := extractTags(currentMemo.Payload)
    
    // 4. 计算综合得分
    var results []RelatedMemo
    for _, v := range vectorResults {
        if v.MemoUID == memoUID {
            continue // 排除自身
        }
        
        memo, _ := s.memoStore.GetByUID(ctx, v.MemoUID)
        memoTags := extractTags(memo.Payload)
        
        // 标签共现得分
        sharedTags := intersect(currentTags, memoTags)
        tagScore := float32(len(sharedTags)) / float32(max(len(currentTags), 1))
        
        // 时间邻近得分 (7天内得分高)
        timeDiff := abs(currentMemo.CreatedTs - memo.CreatedTs)
        timeScore := max(0, 1.0 - float32(timeDiff)/(7*24*3600))
        
        // 加权融合
        finalScore := 0.6*v.Similarity + 0.3*tagScore + 0.1*timeScore
        
        results = append(results, RelatedMemo{
            Name:       memo.UID,
            Title:      extractTitle(memo.Content),
            Similarity: finalScore,
            SharedTags: sharedTags,
            CreatedTs:  memo.CreatedTs,
        })
    }
    
    // 5. 排序取 Top-N
    sort.Slice(results, func(i, j int) bool {
        return results[i].Similarity > results[j].Similarity
    })
    
    if len(results) > limit {
        results = results[:limit]
    }
    
    // 写入缓存
    data, _ := json.Marshal(results)
    s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
    
    return results, nil
}
```

### 4.4 前端实现

```tsx
// web/src/components/MemoRelated/RelatedList.tsx

interface RelatedListProps {
  memoName: string;
}

export function RelatedList({ memoName }: RelatedListProps) {
  const { data, isLoading } = useQuery({
    queryKey: ['related', memoName],
    queryFn: () => memoService.getRelatedMemos({ memoName, limit: 5 }),
    enabled: !!memoName,
  });

  if (isLoading) {
    return <Skeleton className="h-32" />;
  }

  if (!data?.memos?.length) {
    return (
      <div className="text-sm text-gray-500">
        {t("memo.no-related")}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {t("memo.related-notes")}
      </h4>
      {data.memos.map((memo) => (
        <Link
          key={memo.name}
          to={`/m/${memo.name}`}
          className="block p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800"
        >
          <div className="text-sm font-medium truncate">{memo.title}</div>
          <div className="flex items-center gap-2 mt-1 text-xs text-gray-500">
            <span>{(memo.similarity * 100).toFixed(0)}% {t("memo.match")}</span>
            {memo.sharedTags.length > 0 && (
              <span className="text-blue-500">
                #{memo.sharedTags[0]}
              </span>
            )}
          </div>
        </Link>
      ))}
    </div>
  );
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `server/service/memo/related.go` - 相关服务
- [ ] `server/router/api/v1/memo_service.go` - API 扩展
- [ ] `web/src/components/MemoRelated/RelatedList.tsx` - 前端组件
- [ ] `web/src/components/MemoRelated/index.ts` - 导出

### 5.2 Proto 变更

- [ ] `proto/api/v1/memo_service.proto` - 新增 RPC

### 5.3 国际化

- [ ] `web/src/locales/en.json` - 新增 key
- [ ] `web/src/locales/zh-Hans.json` - 新增 key

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 有相关笔记 | 有标签和内容相似的笔记 | 返回 Top-5 |
| 无相关笔记 | 无相似内容 | 返回空列表 |
| 自身排除 | 当前笔记 UID | 不包含自身 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 首次响应 | < 500ms | 集成测试 |
| 缓存命中 | < 50ms | 集成测试 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 6 人天 |
| 预期收益 | 笔记关联发现率 +300% |
| 风险评估 | 中 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1-2 | 2人天 | 后端服务实现 |
| Day 3-4 | 2人天 | 前端组件实现 |
| Day 5-6 | 2人天 | 测试 + 优化 |

### 8.2 检查点

- [ ] Day 2: API 可用
- [ ] Day 4: 前端渲染正确
- [ ] Day 6: 性能达标

---

## 附录

### A. 参考资料

- [笔记增强路线图](../../research/memo-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
