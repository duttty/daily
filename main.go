package main

import (
	"daily/wallpaper"
)

var (
	Engines = make(map[string]interface{}, 1)
)

type Daily struct {
	// 程序引擎
	Engines map[string]Engineface
}
type Engineface interface {
	Run()
}

func main() {
	d := &Daily{}
	// 载入壁纸引擎
	d.Engines = make(map[string]Engineface)
	d.Engines["wallpaper"] = &wallpaper.Wallpaper{}
	for _, v := range d.Engines {
		v.Run()
	}
}
