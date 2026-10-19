// Command sonolus-notgarupa-skin-generator builds the NotGarupa skin pack (see
// package skin) as a sonolus-pack source tree, a servable pack and the sprite
// contract.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	skin "github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/cli"
)

var version = "dev"

func main() {
	var common cli.Common
	fs := flag.NewFlagSet("sonolus-notgarupa-skin-generator", flag.ExitOnError)
	common.Register(fs)
	var o skin.Options
	fs.StringVar(&o.Note, "note", "skin00", "note skin bundle (ingameskin/noteskin/<id>)")
	fs.StringVar(&o.Directional, "directional", "directionalflickskin00", "directional flick note skin bundle")
	fs.StringVar(&o.Field, "field", "skin00", "field (lane) skin bundle (ingameskin/fieldskin/<id>)")
	fs.StringVar(&o.Judge, "judge", "skin00", "judge atlas bundle (ingameskin/judgeskin/<id>)")
	variant := fs.String("variant", "normal", "skin item to build: normal, or habahiro (normal content plus the HABAHIRO note/field/flash sprites; see HABAHIRO.md)")
	habahiro := fs.Bool("habahiro", false, "same as -variant habahiro")
	fs.BoolVar(&o.Startup, "startup", false, "include the live startup direction art (startup_line_star, startup_title_base)")
	fs.BoolVar(&o.SkillAdjust, "skill-adjust", false, "include game_play_line_skill_adjust_effect")
	fs.IntVar(&o.MaxTexture, "max-texture", 0, "maximum skin texture side (power of two; default 4096, -variant habahiro 8192: full-resolution HABAHIRO notes need 4096×8192)")
	fs.StringVar(&o.Name, "name", "", "skin item name (default notgarupa-<note>-<field>, plus -habahiro for that variant)")
	contract := fs.String("contract", "contract.json", "write the sprite contract JSON here (empty: skip); -variant habahiro defaults to contract-habahiro.json")
	_ = fs.Parse(os.Args[1:])
	if common.Version {
		fmt.Println(version)
		return
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	o.Variant = skin.Variant(*variant)
	if *habahiro {
		o.Variant = skin.Habahiro
	}
	if o.Variant == skin.Habahiro {
		// Separate outputs so that the normal pack stays untouched.
		defaults := map[string]*string{"out": &common.Out, "pack": &common.Pack, "contract": contract}
		values := map[string]string{"out": "out/source-habahiro", "pack": "out/pack-habahiro", "contract": "contract-habahiro.json"}
		for k, p := range defaults {
			if !set[k] {
				*p = values[k]
			}
		}
	}
	o.Source = skin.Source{CacheDir: common.Cache, Region: common.Region, Offline: common.Offline, Log: common.Logf()}

	r, err := skin.Generate(context.Background(), o)
	if err != nil {
		cli.Fatal(err)
	}
	if err := skin.WriteSource(common.Out, r); err != nil {
		cli.Fatal(err)
	}
	if *contract != "" {
		if err := cli.WriteJSON(*contract, r.Contract); err != nil {
			cli.Fatal(err)
		}
	}
	if common.Pack != "" {
		if err := cli.Pack(common.Out, common.Pack); err != nil {
			cli.Fatal(err)
		}
	}
	fmt.Printf("skin %s: %d sprites, texture %dx%d -> %s\n", r.Item.Name, len(r.Contract.Sprites), r.Contract.Texture[0], r.Contract.Texture[1], common.Out)
}
