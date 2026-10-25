# HABAHIRO 皮肤变体

原作对 `#HABAHIRO`（多 lane 宽音符）谱面强制使用一组专用资源，不受玩家的皮肤设置影响：

- note 皮肤 `ingameskin/noteskin/habahiro`
- 变更后的场地 `ingameskin/fieldskin/habahiro`
- 打击特效 `ingameskin/tapeffect/habahiro`，其中也包含 lane 变更闪光 `Root_effect`
- 背景 `ingameskin/bgskin/habahiro`

方向键皮肤仍按玩家设置。判定皮肤 `judgeskin/habahiro` 与 skin00 逐字节相同，所以继续使用 `judge_*`。

HABAHIRO 是 NotGarupa 的固定组成部分，生成器每次都会输出它，没有开关。一个关卡只能选一个 skin 条目，而原作对 `#HABAHIRO` 关卡强制使用 HABAHIRO 的音符、lane 与判定皮肤（方向键皮肤保留玩家设置），所以每种方向键皮肤各有一个 HABAHIRO 条目，由服务器为这类关卡强制选用。条目中普通命名的精灵用 skin00 音符与 lane（lane 变更之前显示的就是 skin00 lane），另外加上下文列出的 `habahiro/<原作名>` 精灵。

| 项 | 普通 | HABAHIRO |
|---|---|---|
| 条目名 | `garupa-skin00-skin00-skin00` 等 | `garupa-habahiro-<方向键>`（如 `garupa-habahiro-skin00`，标题「HABAHIRO / TYPE1」） |
| sonolus-pack 源 / 可加载的包 | `out/source` / `out/pack`（两个条目在同一个包里） | 同左 |
| 契约 | `out/contracts/<条目>.json` | 同左（另有顶层 `habahiro` 段，内容即本文的机器可读版） |
| 纹理 | 2048×4096 | 4096×8192 |

HABAHIRO 条目的纹理是 4096×8192，全部按原分辨率存放，没有缩小。只有宽音符图集一项就约有 15 Mpx²，所以 4096² 放不下。这超出了 Sonolus 建议的 2048²，但没有超过 8192 的上限。

背景由 `sonolus-notgarupa-background-generator` 生成，条目为 `garupa-habahiro`（只有普通背景；原作只在有 Fever 的多人模式加载 fever 背景）。粒子由粒子生成器负责（`garupa-habahiro-<方向键特效>`），不在本条目内。

## 宽音符精灵名

lane 编号为 0..6。宽度 w 的取值是 1..7，起始 lane `s` 必须满足 `s + w ≤ 7`。把这段 lane 依次用 `_` 连接，就是精灵名的后缀：

```
habahiro/note_<族>_<s>_<s+1>_…_<s+w−1>      例：lane 2..3 → habahiro/note_normal_2_3
```

- 每个族都有 28 个 run。按宽度递增、起始 lane 递增排列时，下标为 `idx(s,w) = (w−1)(16−w)/2 + s`，范围是 0..27。
- 族有 6 个：`note_normal`、`note_normal_16`（后拍单键）、`note_skill`、`note_flick`、`note_long`、`note_long_flash`。
- w = 1 时同样使用 `habahiro/` 版本。例如 `habahiro/note_normal_3` 的像素与 skin00 不同。HABAHIRO 关卡的所有非方向键音符都改用 `habahiro/` 精灵。

几何参数（全部来自原作 `.sprites`）：

- pivot (0.5, 0.5)，PPU 100，高 120 px。
- 宽度随 w 变化：

| w | 1 | 2 | 3 | 4 | 5 | 6 | 7 |
|---|---|---|---|---|---|---|---|
| 宽 px | 308 | 524 | 736 | 952 | 1164 | 1380 | 1596 |

- note 的缩放规则与普通音符相同。宽音符按 pivot 居中绘制。偶数宽度的水平位置取 span 中心 `s + (w−1)/2`，即两 lane 之间的半按钮；原作截图中偶数宽度的音符正好覆盖两条 lane、居中于两者之间。

| 元素 | 精灵 |
|---|---|
| 单键 / 技能 / 后拍单键 | `note_normal_<run>` / `note_skill_<run>` / `note_normal_16_<run>` |
| Flick | `note_flick_<run>` |
| Long / Slide 头尾 | `note_long_<run>`；flick 尾为 `note_flick_<run>` |
| Slide 中间节点 | `note_slide_among`（w = 1）、`note_slide_among_<w>`（w = 2..7），宽 = 上表 |
| 上滑图标 | `note_flick_top`（w = 1，171×138）、`_2`（w = 2，288×138）、`_3`（w ≥ 3，400×138）；局部偏移 (0, 0.7) |
| Long 按住闪光 | `note_long_flash_<run>`，使用 Long 的真实 lane |
| Slide 按住闪光、按住中的 Slide 头 | 用居中的 run，而不是 Slide 的真实 lane：`size = w`，`start = floor((7−size)/2) + (size 为偶数且中心 lane > 3 ? 1 : 0)`，精灵为 `note_long_flash_<run>`、`note_long_<run>` |
| 长条身体 | `habahiro/longNoteLine`、`habahiro/longNoteLine2`（各有 `@E6E6E6`），146×205；网格半宽 × `widthRate(w)` |
| 同时押线 | `habahiro/simultaneous_line`，10×27，PPU 65（skin00 为 PPU 66） |
| 方向键 | 不变，仍用方向键皮肤。宽方向键的身体为空精灵，只画箭头 |

`habahiro/note_long_flash_*` 已和普通版一样由加法混合换算为 alpha 混合。

## lane 变更时的替换

设 `T` 为关卡的 `laneChangeTime`。在 `T` 之前，HABAHIRO 关卡显示 skin00 场地，也就是普通精灵；没有 `#LANE_CHANGE` 的关卡全程如此。

到 `T + 0.4166666567325592` s（`Root_effect` 动画的 ChangeLane 事件，60 Hz 下第 25 帧）时，只替换精灵，变换不变：

| 之前 | 之后 | 尺寸 |
|---|---|---|
| `bg_line_rhythm` | `habahiro/bg_line_rhythm` | 1160×610，pivot (0.5, 0)，UITexture（UI 单位 = 像素） |
| `game_play_line` | `habahiro/game_play_line` | 1800×**43**（skin00 为 38），pivot (0.5, 0.5)，PPU 69 |
| `game_play_line_skill_adjust_effect` | `habahiro/game_play_line_skill_adjust_effect` | 1800×43，PPU 69（需 `-skill-adjust`） |

Sudden 区域也重新绑定到 `habahiro/bg_line_rhythm` 这张贴图，原作没有单独的图像。

Sonolus preview 的平面轨道同样分两张：`notgarupa/preview/lane`（skin00 lane）与 `habahiro/notgarupa/preview/lane`（由 `habahiro/bg_line_rhythm` 反投影，580×2），引擎在 lane 变更时刻 + 0.4166666567325592 s 处换图。

### lane 变更闪光

对 `e = now − T`，在 e ∈ [0, 1] 期间绘制以下 4 个四边形。

- 混合方式为普通 alpha 混合（Sprites-Default），RGB 保持 1，各四边形画满整张贴图。
- 层级：高于判定线，低于主要粒子与音符。
- 精灵 pivot (0.5, 0.5)、PPU 100（`.sprites` 中的值）。
- 位置用下表的矩形，单位为 RhythmGameLines 局部 UI 单位（y 向上，`x0,y0,x1,y1`），坐标映射与 lane 图像相同（skin00 lane 图像底边在 −240）。

| 精灵 | 贴图 | 矩形 | 排序 | alpha 关键帧 (t, a) |
|---|---|---|---|---|
| `habahiro/bg_line_rhythm_flash` | 1160×610 | −580, −242, 580, 368 | 40 | (0, 0.001) (0.0833333, 1) (0.75, 1) (1, 0.001) |
| `habahiro/bg_line_rhythm_flash2` | 1160×610 | −580, −242, 580, 368 | 40 | (0, 0.001) (0.3833333, 1) (0.4166667, 1) (0.45, 1) (1, 0.001) |
| `habahiro/game_play_line_flash` | 1880×120 | −940, −302, 940, −182 | 41 | 同 `bg_line_rhythm_flash` |
| `habahiro/game_play_line_flash2` | 1880×120 | −940, −302, 940, −182 | 41 | 同 `bg_line_rhythm_flash2` |

关键帧之间按 Hermite 插值，切线全为 0，即 smoothstep：`a + (b−a)(3r² − 2r³)`。

## 来源

| 精灵 | 来源 |
|---|---|
| `habahiro/note_*`、`habahiro/simultaneous_line` | Bestdori `ingameskin/noteskin/habahiro`（`.sprites`，图集 RhythmGameSprites1..5、16） |
| `habahiro/longNoteLine*` | 同一 bundle 的 `longNoteLine.png`、`longNoteLine2.png` |
| `habahiro/bg_line_rhythm`、`habahiro/game_play_line*` | Bestdori `ingameskin/fieldskin/habahiro` |
| `habahiro/*_flash*` | Bestdori `ingameskin/tapeffect/habahiro`（`.sprites`） |

每个精灵的名称、来源、`w`/`h`、pivot、PPU 与说明都记录在 HABAHIRO 条目的契约 `out/contracts/notgarupa-habahiro-<方向键>.json` 里。分组如下：`habahiro`（音符与长条），`habahiro_field`，`habahiro_flash`。
