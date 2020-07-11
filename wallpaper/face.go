package wallpaper

type Wallpaperface interface {
	LoadConfig() error
	ChangeWallpaper(path string) error
	SetHotKey()
	DownloadImage(id, url string, cbk chan string)
}

type engineface interface {
	GetImagesURLs(num int) (imgs [][]string, err error)
}
