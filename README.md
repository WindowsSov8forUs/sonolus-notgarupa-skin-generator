# sonolus-notgarupa-skin-generator

生成 NotGarupa 皮肤包：原作的音符 / 方向键 / lane 场 / 判定文字皮肤，自绘 HUD 所需的全部部件（分数、评级槽、生命槽、连击数字、AddScore、FAST/SLOW、Auto 标签、SS 特效）、lane 光柱、舞台灯光与音箱、Full Combo / All Perfect 文字，以及预渲染的字形。精灵名即原作精灵名，约定见 `sonolus-notgarupa-engine/docs/RESOURCE_CONTRACT.md`；每次运行在本目录写出 `contract.json`（尺寸、pivot、PPU、裁切、着色、切片、字形度量）。

来源：

- Bestdori 资源（音符、方向键、场地、判定皮肤），运行时下载到 `.cache/`，不纳入版本库。
- 原作 APK 内置资源（不在 AssetBundle 中、无法联网获取）：HUD 图集（`rhythm-game-ui`、`rhythm-game-additive`、连击数字、`uicommon`、`ui-additive-effect`）、lane tap 光柱、Sudden 线、分数槽高分段贴图、舞台灯光 / 音箱 / 荧光棒、开场信息、FC / AP 字样与 `sgm` 字体，随项目放在 `assets/apk/`（约 6 MB，`go:embed` 编入程序）。`assets/apk/render-catalog.json` 是这些图集的 atlas 行（矩形、pivot、border）。

## 命令行

从 GitHub Release 下载对应平台的压缩包（linux / windows / darwin，amd64 / arm64；由 `.github/workflows/build.yml` 矩阵编译，推送 `v*` 标签时发布），解压后直接运行；也可以用 Go 安装或在本仓库中运行（仓库为私有，需要 `GOPRIVATE=github.com/WindowsSov8forUs/*` 和可访问仓库的 git 凭据）：

```sh
go install github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/cmd/sonolus-notgarupa-skin-generator@latest
go run ./cmd/sonolus-notgarupa-skin-generator [参数]   # 在本仓库中
```

```sh
sonolus-notgarupa-skin-generator [-note skin00] [-directional directionalflickskin00] [-field skin00] [-judge skin00] \
    [-variant normal|habahiro] [-startup] [-skill-adjust] [-max-texture 4096] [-name ...] [公共参数]
```

程序内置所需的 APK 资源，其余素材联网下载到 `-cache`（默认当前目录的 `.cache/`）。`-version` 输出版本号。

- 输出：`out/source`（sonolus-pack 源）与 `out/pack`（可直接被服务器加载）。默认条目名 `notgarupa-<note>-<field>`。
- `-variant habahiro`（旧写法 `-habahiro`）：生成独立的 HABAHIRO 条目 `notgarupa-<note>-<field>-habahiro`，供 `#HABAHIRO` 关卡使用。它包含普通皮肤的全部精灵，另加原作 HABAHIRO 宽音符图集、长条贴图、lane 变更后的场地和 lane 变更闪光，名称为 `habahiro/<原作名>`。默认输出到 `out/source-habahiro`、`out/pack-habahiro` 与 `contract-habahiro.json`，纹理 4096×8192，全部按原分辨率。精灵名规则与几何见 [HABAHIRO.md](HABAHIRO.md)。普通条目的输出不受影响。
- 着色变体以 `名称@RRGGBB[AA]` 命名，九宫格 / 三段切片以 `#L/#M/#R`、`#TL…#BR` 后缀命名，字形以 `sgm_<字号>_<颜色>_<字符>` 命名（2 倍分辨率）。
- 加色混合的美术（舞台灯光、音箱发光等）已换算为普通 alpha 混合。只有舞台与通关美术会裁掉透明边。
- 纹理：普通条目 2048×4096（全部内容放不进 2048²），HABAHIRO 条目 4096×8192。

## 作为 Go 库

模块根包 `skin` 提供与命令行相同的生成功能，不写文件、不退出进程、不使用 sonolus-pack-go，便于嵌入服务器：

```go
import skin "github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator"

r, err := skin.Generate(ctx, skin.Options{
	Variant: skin.Habahiro, // 零值 Options 即默认的普通皮肤
	Source:  skin.Source{CacheDir: "cache/notgarupa", Region: "jp"},
})
```

- `Options` 的零值即命令行的默认值：`Note`、`Directional`、`Field`、`Judge`、`Variant`（`skin.Normal` / `skin.Habahiro`）、`Startup`、`SkillAdjust`、`MaxTexture`、`Name`。
- `Source` 控制下载：`CacheDir`、`Region`、`Offline`、`HTTPClient`、`Log`；`Generate` 的 `ctx` 可取消下载。多个生成器可以共用同一个 `CacheDir`（缓存写入是原子的）。
- 返回 `*skin.Result`：`Item`（sonolus-core-go 的 `database.DatabaseSkinItem`，资源字段留空）、Data（`resource.SkinData`）、Texture、Thumbnail（PNG）与 Contract（即 `contract.json` 的内容）。
- `Result.Files()` 按 item.json 的字段名返回资源字节："thumbnail"、"data"（已 gzip 压缩）、"texture"。
- `skin.WriteSource(dir, results...)` 把结果写成 sonolus-pack 源目录（命令行就是 `Generate` + `WriteSource` + sonolus-pack-go）。

在 sonolus-server-go 中注册（作为依赖同样需要 `GOPRIVATE`）（资源字段填入仓库地址）：

```go
files, err := r.Files()
if err != nil { /* … */ }
it := r.Item // database.DatabaseSkinItem，资源字段待填
it.Thumbnail, it.Data, it.Texture = s.AddBytes(files["thumbnail"], ""), s.AddBytes(files["data"], ""), s.AddBytes(files["texture"], "")
s.Skin.Items = append(s.Skin.Items, sonolusserver.SkinItemModel{DatabaseSkinItem: it})
```

## 公共参数（命令行）

| 参数 | 默认值 | 作用 |
|---|---|---|
| `-out` | `out/source` | 重新生成的 sonolus-pack 源目录 |
| `-pack` | `out/pack` | 用 sonolus-pack-go 打包到该目录（空字符串则跳过） |
| `-cache` | `.cache` | 下载缓存目录 |
| `-region` | `jp` | Bestdori 服务器区域（jp、en、tw、cn、kr） |
| `-offline` | false | 只使用缓存中的下载 |
| `-v` | false | 输出下载日志 |
| `-version` | false | 输出版本号后退出 |

公共代码（Bestdori 下载与缓存、图像处理、打包输出等）放在本项目 `internal/` 下，项目可单独构建与运行。资源名称约定见 `sonolus-notgarupa-engine/docs/RESOURCE_CONTRACT.md`。
