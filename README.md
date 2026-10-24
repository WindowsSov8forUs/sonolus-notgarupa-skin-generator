# sonolus-notgarupa-skin-generator

生成 NotGarupa 的全部皮肤条目：原作的音符 / 方向键 / lane 场 / 判定文字皮肤，自绘 HUD 所需的全部部件（分数、评级槽、生命槽、生命遮罩、连击数字、AddScore、FAST/SLOW、Auto 标签、SS 特效）、lane 光柱、舞台灯光与音箱，以及预渲染的字形。精灵名即原作精灵名，约定见 `sonolus-notgarupa-engine/docs/RESOURCE_CONTRACT.md`。

## 生成哪些条目

默认运行即生成完整资源，没有选择单个条目的参数。条目集合由原作 JP 10.1.4 的皮肤主数据决定（`internal/master/skins.json`，规则见 `internal/master/master.go`）：

| 来源 | 组合 | 条目数 |
|---|---|---|
| 普通设置 | 音符皮肤 7 × 方向键皮肤 5 × lane 15，判定皮肤固定 skin00 | 525 |
| Collaboration / Limited 特殊组合 | 固定该行替换的组件（音符、lane、方向键、判定），其余轴照常排列 | 251 |
| HABAHIRO（谱面强制） | 音符、lane、判定固定为 habahiro，方向键保留玩家设置（普通 5 种 + persona） | 6 |

共 782 个条目。条目名为 `garupa-<音符>-<方向键>-<lane>`（音符 → 方向键 → lane，条目也按这个顺序排列），判定皮肤不是 skin00 时再加 `-<判定>`，标题为对应的原作名称，以 ` / ` 连接（例：`garupa-skin00-skin00-skin00`「TYPE1 / TYPE1 / レーンスキン00」、`garupa-skin00-skin00-skin_bike`「TYPE1 / TYPE1 / bikeコラボ」、`garupa-skin_persona`「ファントム復刻スキン」）；HABAHIRO 条目为 `garupa-habahiro-<方向键>`「HABAHIRO / TYPE1」。名称来自原作各服主数据（`internal/master/skins.json` 的 `names`）：每种语言用对应服的名称——日文 JP、英文 EN、繁体中文 TW、简体中文 CN，该服没有时取 JP、EN、TW、CN 中第一个有的名称（例如 Roselia 的 lane：「Roselia Lane」「Roselia的軌跡」「Roselia的按键条皮肤」）；普通设置用各行的皮肤名（`TYPE1`、`レーンスキン00`、`エフェクトスキン00`、`SEスキン00`、`デフォルト背景` 等），Collaboration 行用季节描述（`MasterSeasonSpecial.description`，如 `ペルソナコラボ`），Limited 行用限定皮肤名（如 `エイプリルフール2019`）；同一组件出现在多行时取 Limited 行，其次取季节最早的行。特殊行替换的各轴是一个固定组合，条目名只写一次（该组合的第一个 bundle），标题只写一次（该行的名称）。条目的作者为 `GirlsBandParty`，副标题为游戏名。包中（`db.json`）的条目按原作主数据的顺序排列（即 `Items()` 的顺序）：先是普通设置，按各轴 master 行的顺序组合（默认组合在最前）；其后是特殊组合，按 Collaboration、Limited 的行号；HABAHIRO 在最后。sonolus-pack 按目录名列出条目，打包后再按这个顺序重排。没有任何途径取得原始资源的主数据条目整体不生成，见 `sonolus-notgarupa-engine/docs/EXCLUDED_RESOURCES.md`。

每个条目包含：

- 该组合的音符、方向键、lane（不含技能叠加层 `game_play_line_skill_adjust_effect`）与判定精灵。
- 所有条目共用的 HUD、lane 光柱、舞台、着色变体与字形。
- `default/judge_auto`：默认判定图集的 judge_auto。原作在 Auto Live 中从不使用特殊判定皮肤，引擎在 Auto 中画这一个。
- 音符皮肤主数据 `noteSyncEdgeMargin` 为 1.1 的条目（TYPE7）带有存在标记 `notgarupa/noteSyncEdgeMargin_1.1`（1×1 透明），引擎据此把同时押线的两端向内收 1.1 × 音符缩放。
- HABAHIRO 条目：普通命名的精灵用 skin00 音符与 lane（lane 变更之前显示 skin00 lane），另加 `habahiro/<原作名>` 的宽音符图集、长条贴图、lane 变更后的场地与闪光，见 [HABAHIRO.md](HABAHIRO.md)。它们不是与普通条目并列的皮肤：服务器不在列表中提供它们，只按玩家的方向键配置项为 `#HABAHIRO` 关卡选用（`Result.Habahiro` 为 true）。

命名规则：着色变体以 `名称@RRGGBB[AA]` 命名，九宫格 / 三段切片以 `#L/#M/#R`、`#TL…#BR` 后缀命名，字形以 `sgm_<字号>_<颜色>_<字符>` 命名（2 倍分辨率）。加色混合的美术（舞台灯光、音箱发光等）已换算为普通 alpha 混合；只有舞台美术会裁掉透明边。纹理：普通条目 2048×4096（全部内容放不进 2048²），HABAHIRO 条目 4096×8192（全部按原分辨率）。

来源：

- Bestdori 资源（音符、方向键、场地、判定皮肤），按服务器优先级取第一个拥有该 bundle 的服务器，运行时下载到 `.cache/`。
- 原作 APK 内置资源（不在 AssetBundle 中、无法联网获取）：HUD 图集（`rhythm-game-ui`、`rhythm-game-additive`、连击数字、`uicommon`、`ui-additive-effect`）、lane tap 光柱、Sudden 线、分数槽高分段贴图、舞台灯光 / 音箱 / 荧光棒、生命遮罩 `sub_glow`（演奏场景的 GameOverCover 贴图）与 `sgm` 字体，随项目放在 `assets/apk/`（`go:embed` 编入程序）。`assets/apk/render-catalog.json` 是这些图集的 atlas 行（矩形、pivot、border）。

## 命令行

从 GitHub Release 下载对应平台的压缩包（linux / windows / darwin，amd64 / arm64；由 `.github/workflows/build.yml` 矩阵编译，推送 `v*` 标签时发布），解压后直接运行；也可以用 Go 安装或在本仓库中运行（仓库为私有，需要 `GOPRIVATE=github.com/WindowsSov8forUs/*` 和可访问仓库的 git 凭据）：

```sh
go install github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/cmd/sonolus-notgarupa-skin-generator@latest
sonolus-notgarupa-skin-generator          # 生成全部条目
```

输出到 `out/`：`out/source`（sonolus-pack 源）、`out/pack`（可直接被服务器加载）与 `out/contracts/<条目>.json`（每个精灵的尺寸、pivot、PPU、裁切、着色、切片、字形度量）。首次运行下载约 18 MB 素材；全部 782 个条目约 1.3 GB，8 核约 10 分钟。

参数只用于特殊场景：

| 参数 | 默认值 | 作用 |
|---|---|---|
| `-out` | `out` | 输出目录 |
| `-cache` | `.cache` | 下载缓存目录 |
| `-offline` | false | 只使用缓存中的下载 |
| `-regions` | `jp,en,tw,cn,kr` | Bestdori 服务器优先级。各服有独占素材，每个 bundle 取自第一个拥有它的服务器（该 bundle 的全部文件都来自同一服）；某服没有时记录在缓存中，`-offline` 重跑时选择相同的服务器 |
| `-j` | CPU 数 | 同时生成的条目数（内存不足时调小） |
| `-v` | false | 输出下载日志 |
| `-version` | false | 输出版本号后退出 |

## 作为 Go 库

模块根包 `skin` 提供与命令行相同的生成功能，不写文件、不退出进程、不使用 sonolus-pack-go，便于在服务器启动时一次性生成：

```go
import skin "github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator"

names, err := skin.Items() // 全部条目名，不需要网络
err = skin.Generate(ctx, skin.Options{Source: skin.Source{CacheDir: "cache/notgarupa"}},
	func(r *skin.Result) error { /* 每个条目生成完成时调用一次 */ return nil })
```

- `Options`：`Source` 控制下载（`CacheDir`、`Regions`、`Offline`、`HTTPClient`、`Log`），`Workers` 为并行数（0 = CPU 数）；`ctx` 可取消。多个生成器可以共用同一个 `CacheDir`（缓存写入是原子的）。
- `Generate` 逐个交出条目（emit 在同一个 goroutine 中调用，顺序不定），第一个错误会中止并返回；共用的素材只下载、解码一次。
- `Result`：`Item`（sonolus-core-go 的 `database.DatabaseSkinItem`，资源字段留空）、`Habahiro`、`Data`（`resource.SkinData`）、`Texture`、`Thumbnail`（PNG）、`Contract` 与 `Axes`（音符、方向键、lane 三个设置轴）。`Result.Files()` 按 item.json 的字段名返回资源字节："thumbnail"、"data"（已 gzip 压缩）、"texture"。命令行在打包后另写 `out/pack/garupa-axes.json`：按包中顺序列出每个可见条目的设置轴（`axis`、`bundle`、原作名称 `name`），服务器据此提供组合搜索；库中同样的信息在 `Result.Axes`（HABAHIRO 条目为空）。
- `skin.CreateSourceTree(dir)` 创建 sonolus-pack 源目录，`(*SourceTree).Add(r)` 写入一个条目（命令行就是 `Generate` + `SourceTree` + sonolus-pack-go）。

在 sonolus-server-go 中注册（作为依赖同样需要 `GOPRIVATE`）：

```go
err = skin.Generate(ctx, opts, func(r *skin.Result) error {
	files, err := r.Files()
	if err != nil {
		return err
	}
	it := r.Item // database.DatabaseSkinItem，资源字段待填
	it.Thumbnail, it.Data, it.Texture = s.AddBytes(files["thumbnail"], ""), s.AddBytes(files["data"], ""), s.AddBytes(files["texture"], "")
	s.Skin.Items = append(s.Skin.Items, sonolusserver.SkinItemModel{DatabaseSkinItem: it})
	return nil
})
```

公共代码（主数据、Bestdori 下载与缓存、图像处理、打包输出等）放在本项目 `internal/` 下，项目可单独构建与运行。
