// Command sonolus-notgarupa-skin-generator builds the NotGarupa skin pack (see
// package skin): the ordinary item and the #HABAHIRO item as one sonolus-pack
// source tree and servable pack, plus their sprite contracts.
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
	fs.BoolVar(&o.Startup, "startup", false, "include the live startup direction art (startup_line_star, startup_title_base)")
	fs.BoolVar(&o.SkillAdjust, "skill-adjust", false, "include game_play_line_skill_adjust_effect")
	fs.IntVar(&o.MaxTexture, "max-texture", 4096, "maximum texture side of the ordinary item (power of two); the HABAHIRO item uses at least 8192")
	fs.StringVar(&o.Name, "name", "", "ordinary skin item name (default notgarupa-<note>-<field>); the HABAHIRO item is <name>-habahiro")
	contract := fs.String("contract", "contract.json", "write the ordinary item's sprite contract JSON here (empty: skip)")
	habahiroContract := fs.String("contract-habahiro", "contract-habahiro.json", "write the HABAHIRO item's sprite contract JSON here (empty: skip)")
	_ = fs.Parse(os.Args[1:])
	if common.Version {
		fmt.Println(version)
		return
	}
	o.Source = skin.Source{CacheDir: common.Cache, Regions: common.Regions, Offline: common.Offline, Log: common.Logf()}

	results, err := skin.Generate(context.Background(), o)
	if err != nil {
		cli.Fatal(err)
	}
	if err := skin.WriteSource(common.Out, results...); err != nil {
		cli.Fatal(err)
	}
	for _, r := range results {
		path := *contract
		if r.Habahiro {
			path = *habahiroContract
		}
		if path == "" {
			continue
		}
		if err := cli.WriteJSON(path, r.Contract); err != nil {
			cli.Fatal(err)
		}
	}
	if common.Pack != "" {
		if err := cli.Pack(common.Out, common.Pack); err != nil {
			cli.Fatal(err)
		}
	}
	for _, r := range results {
		fmt.Printf("skin %s: %d sprites, texture %dx%d\n", r.Item.Name, len(r.Contract.Sprites), r.Contract.Texture[0], r.Contract.Texture[1])
	}
	fmt.Printf("%d skins -> %s\n", len(results), common.Out)
}
