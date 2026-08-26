基于 Go 实现的自来水厂加氯加药与滤池控制系统项目，一款市政供水控制服务，完成水厂加氯加药投加、滤池反冲洗轮换、出水浊度监测与水质告警管理。

系统由浊度/余氯监测（internal/turb）、滤池与反冲洗（internal/filter）、加氯加药投加（internal/dose）、投加泵控制（internal/pump）、水质告警（internal/alarm）、化验记录（internal/record）、事件总线（internal/event）与中控台（internal/console）组成，HTTP 入口位于 cmd/server，前端页面为 web/console.html。

运行方式：

go run ./cmd/server

服务默认监听 127.0.0.1:18080，数据目录为 runtime。主要接口包括 /api/console/summary、/api/quality、/api/dose、/api/filter/backwash、/api/pressure、/api/flow、/api/console/command 与 /ws/console 事件流。

构建采用 vendor 离线模式，唯一第三方依赖为 github.com/cespare/xxhash/v2（化验记录指纹）。
