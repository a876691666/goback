package operationlog

import "github.com/goback/pkg/dal"

// ListRequest 操作日志列表请求（使用 PocketBase 风格参数）
// 示例请求:
//   GET /operation-logs?filter=username~"admin"&&created_at>="2022-01-01"&sort=-created_at&page=1&perPage=20&fields=id,username,module,action
type ListRequest = dal.ListParams
