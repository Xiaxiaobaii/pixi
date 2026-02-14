# Pixiv 关注作者作品下载器（Go）

这是一个 Pixiv 命令行下载工具，支持：

- 读取你的关注列表
- 下载被关注作者的原图
- 将作品元数据写入 SQLite
- 自动清理本地数据库中不存在的“非法/孤儿”文件

## 基于源码的功能说明

项目核心文件：

- `pixiv.go`：CLI 入口、主下载流程、并发控制
- `lib/lib.go`：Pixiv API 调用、图片下载、SQLite 操作、文件清理

当前代码执行流程：

1. 交互输入 `homeid` 与 `cookie`。
2. 初始化数据库与目录。
3. 执行本地清理（`DeleteBadImageFromRootfs`）。
4. 请求关注列表：
   - `GET /ajax/user/{homeid}/following?offset=0&limit=99&rest=show`
5. 请求每个关注作者的作品 ID：
   - `GET /ajax/user/{uid}/profile/all`
6. 遍历作品：
   - 若在黑名单中则跳过；
   - 若数据库已存在则跳过；
   - 读取作品详情（`GET /ajax/illust/{pid}`）；
   - 下载原图；
   - 写入 `pixiv.db` 的 `imgs` 表。

## 运行要求

- Go `1.26rc1`（以 `go.mod` 为准）
- `go.mod` 中使用了本地 `replace`，需要同级目录存在：
  - `../autotool`
  - `../sqlite`
  - `../lrys`
- 有效 Pixiv 登录 Cookie

## 构建

```bash
./build.sh
# 或
go build -ldflags "-s -w"
```

## 运行

```bash
./pixiv
```

启动后按提示输入：

- `homeid`（Pixiv 用户 ID）
- `cookie`（Pixiv Cookie 字符串）

## 命令行参数（源码中已声明）

- `-onlyBad`（bool）：只执行清理后退出
- `-debug`（bool）：调试开关
- `-proxyType`（string）：代理类型，`none` 或 `http`
- `-proxy`（string）：代理地址
- `-thread`（int）：并发线程数（默认 `10`）

注意：当前代码没有调用 `flag.Parse()`，因此这些参数默认不会生效（会保持默认值）。

## 输出与数据

- 图片目录：`img/<author_uid>/<illust_id>.jpg`
- 主数据库：`pixiv.db`
  - 表：`imgs`、`star`
- 黑名单数据库：`Blacklist.db`
  - 表：`bad`

`imgs` 表写入字段：

- `name`
- `id`
- `Author`
- `Authorid`
- `R18`
- `createDate`
- `tags`
- `size`

## 注意事项

- 关注列表目前只请求单页（`limit=99`）。
- 网络错误使用递归重试。
- `cookie` 通过 `fmt.Scanln` 读取，若字符串内包含空格可能被截断。
- 请确保你的使用行为符合 Pixiv 服务条款与当地法律法规。

## License

仓库中未发现许可证文件；若需要分发，请补充 License。