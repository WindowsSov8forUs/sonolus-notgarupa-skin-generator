// Command sonolus-notgarupa-skin-generator builds every NotGarupa skin item
// (see package skin) into <out>/source, packs it into <out>/pack and writes
// each item's sprite contract to <out>/contracts/<item>.json.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	skin "github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator"
	"github.com/WindowsSov8forUs/sonolus-notgarupa-skin-generator/internal/cli"
)

var version = "dev"

func main() {
	var common cli.Common
	fs := flag.NewFlagSet("sonolus-notgarupa-skin-generator", flag.ExitOnError)
	common.Register(fs)
	_ = fs.Parse(os.Args[1:])
	if common.Version {
		fmt.Println(version)
		return
	}
	opts := skin.Options{
		Source:  skin.Source{CacheDir: common.Cache, Regions: common.Regions, Offline: common.Offline, Log: common.Logf()},
		Workers: common.Workers,
	}
	source, contracts := filepath.Join(common.Out, "source"), filepath.Join(common.Out, "contracts")
	tree, err := skin.CreateSourceTree(source)
	if err != nil {
		cli.Fatal(err)
	}
	if err := os.RemoveAll(contracts); err != nil {
		cli.Fatal(err)
	}
	total, err := skin.Items()
	if err != nil {
		cli.Fatal(err)
	}
	done := 0
	err = skin.Generate(context.Background(), opts, func(r *skin.Result) error {
		if err := tree.Add(r); err != nil {
			return err
		}
		if err := cli.WriteJSON(filepath.Join(contracts, r.Item.Name+".json"), r.Contract); err != nil {
			return err
		}
		done++
		fmt.Printf("[%d/%d] skin %s: %d sprites, texture %dx%d\n", done, len(total), r.Item.Name, len(r.Contract.Sprites), r.Contract.Texture[0], r.Contract.Texture[1])
		return nil
	})
	if err != nil {
		cli.Fatal(err)
	}
	if err := cli.Pack(source, filepath.Join(common.Out, "pack")); err != nil {
		cli.Fatal(err)
	}
	fmt.Printf("%d skins -> %s\n", done, common.Out)
}
