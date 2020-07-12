package wallpaper

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
	"gopkg.in/ini.v1"
)

type Wallpaper struct {
	Dir         string
	Exec        string
	CfgPath     string
	ImgSavePath string
	Flag        string
	Type        string
	Engine      engineface
	TempNum     int
	NowCount    int
	MaxCount    int
	NextImageID string
}

func (w *Wallpaper) LoadConfig() {
	// 读取文件执行路径
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	w.Exec = path
	w.Dir = filepath.Dir(path)
	w.CfgPath = filepath.ToSlash(fmt.Sprintf("%s/config.ini", w.Dir))
	w.Flag = "n"
	//读取wallpaper配置
	cfg, err := ini.Load(w.CfgPath)
	if err != nil {
		log.Fatalf("[err LoadConfig],Load path:\n%s\n,%s\n", w.Dir, err)
	}
	w.Type = cfg.Section("wallpaper").Key("type").String()
	w.TempNum = cfg.Section("wallpaperTemp").Key("tempNum").RangeInt(10, 4, 30)
	w.ImgSavePath = filepath.ToSlash(cfg.Section("wallpaper").Key("imgSavePath").String())
	if !filepath.IsAbs(w.ImgSavePath) {
		log.Println("[info LoadCFG] imgSavePath is not Abs\n", w.ImgSavePath)
		w.ImgSavePath = filepath.ToSlash(fmt.Sprintf("%s/images", w.Dir))
		cfg.Section("wallpaper").Key("imgSavePath").SetValue(w.ImgSavePath)
		err = cfg.SaveTo(w.CfgPath)
		if err != nil {
			log.Println("[err save new abs path] \n", w.ImgSavePath)
		}
	}
	// 引擎配置
	imageIDs := cfg.Section("wallpaperTemp").Key(fmt.Sprintf("%sImageIDs", w.Type)).Strings(",")
	w.MaxCount = len(imageIDs)
	w.NowCount = cfg.Section("wallpaperTemp").Key(fmt.Sprintf("%sNowCount", w.Type)).RangeInt(0, 0, 30)
	// 根据用户路径创建文件
	err = os.Mkdir(w.ImgSavePath, 0777)
	if err == nil {
		// 新建文件成功 初始化配置
		log.Println("[info loadcfg] mkdir succ\nPATH = ", w.ImgSavePath)
		w.NowCount = 0
		w.MaxCount = 0
	} else if !errors.Is(err, os.ErrExist) {
		// 路径改变
		deft := filepath.ToSlash(fmt.Sprintf("%s/images", w.Dir))
		cfg.Section("wallpaper").Key("imgSavePath").SetValue(deft)
		cfg.SaveTo(w.CfgPath)
		w.ImgSavePath = deft
		if err = os.Mkdir(deft, 0777); err != nil && !errors.Is(err, os.ErrExist) {
			log.Fatalln("[err create images file in path] \n", deft, "\n", err)
		}
		// 新建文件成功 初始化配置
		w.NowCount = 0
		w.MaxCount = 0
	}
	if w.NowCount >= w.MaxCount-1 {
		w.NextImageID = ""
	} else {
		w.NextImageID = imageIDs[w.NowCount+1]
	}
	switch w.Type {
	case "pixabay":
		w.Engine = &pixabay{
			AK:       cfg.Section("wallpaper").Key("pixabayAK").String(),
			Keywords: strings.ReplaceAll(cfg.Section("wallpaper").Key("keywords").String(), ",", "+"),
		}
	default:
		w.Engine = &unsplash{
			AK:       cfg.Section("wallpaper").Key("unsplashAK").String(),
			Keywords: cfg.Section("wallpaper").Key("keywords").String(),
		}
	}

}

func (w *Wallpaper) DownloadImage(id, url string, cbk chan string) {
	imgName := filepath.ToSlash(fmt.Sprintf("%s/%s.jpg", w.ImgSavePath, id))
	// 查看本地储存
	_, err := os.Stat(imgName)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Println("[err os STAT] \n", err)
			cbk <- ""
			return
		}
	} else {
		cbk <- id
		log.Println("[info] load local img \n", id)
		return
	}

	res, err := http.Get(url)
	if err != nil {
		log.Println("[err download image],url:\n", url)
		cbk <- ""
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Println("[err download image],code:\n", res.Status)
		cbk <- ""
		return
	}

	f, err := os.Create(imgName)
	if err != nil {
		log.Println("[err create file] \n", err)
		cbk <- ""
		return
	}
	defer f.Close()

	if err != nil {
		fmt.Println(err)
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		log.Println("[err copy image] \n", err)
		cbk <- ""
		return
	}

	log.Println("[info] download img\n", id)
	cbk <- id
}
func (w *Wallpaper) ChangeWallpaper(path string) {
	log.Println("[info] change wallpaper start \n", path)
	cfg, err := ini.Load(w.CfgPath)
	if err != nil {
		log.Println("[err load cfg] \n", err)
		return
	}
	fpath := filepath.ToSlash(path)
	_, err = os.Stat(fpath)
	if errors.Is(err, os.ErrNotExist) {
		cfg.Section("wallpaperTemp").Key(fmt.Sprintf("%sImageIDs", w.Type)).SetValue("0")
		if err = cfg.SaveTo(w.CfgPath); err != nil {
			log.Println("[err save cfg] \n", err)
		}
		log.Fatalln("[err ChangeWallpaper],can't find file.")
		return
	}
	w.NowCount++
	cfg, err = ini.Load(w.CfgPath)
	if err != nil {
		log.Println("[err load cfg] \n", err)
		return
	}
	cfg.Section("wallpaperTemp").Key(fmt.Sprintf("%sNowCount", w.Type)).SetValue(strconv.Itoa(w.NowCount))
	if err = cfg.SaveTo(w.CfgPath); err != nil {
		log.Println("[err save cfg] \n", err)
	}

	h := syscall.MustLoadDLL("user32.dll")
	c := h.MustFindProc("SystemParametersInfoW")
	defer syscall.FreeLibrary(h.Handle)
	uiAction := 0x0014
	uiParam := 0
	pvParam := syscall.StringToUTF16Ptr(fpath)
	fWinIni := 1
	r2, _, err := c.Call(uintptr(uiAction),
		uintptr(uiParam),
		uintptr(unsafe.Pointer(pvParam)),
		uintptr(fWinIni))
	if r2 != 0 {
		log.Println(r2, err, fpath)
	}

}

func (w *Wallpaper) Hotkey() {
	nk, exist, err := registry.CreateKey(registry.CLASSES_ROOT, "DesktopBackground\\Shell\\下一张壁纸\\command", registry.ALL_ACCESS)
	if err != nil {
		log.Println("[err Hotkey],CreateKey err:\n", err)
		return
	}
	defer nk.Close()
	if exist {
		return
	}
	// 键入值运行程序
	nk.SetStringValue("", fmt.Sprintf("%s -n", w.Exec))

}

func (w *Wallpaper) downloadAndSaveImages() {
	urls, err := w.Engine.GetImagesURLs(w.TempNum)
	if err != nil {
		log.Println("[err get images] \n", err)
		return
	}
	cb := make(chan string, len(urls))
	for _, v := range urls {
		go w.DownloadImage(v[0], v[1], cb)
	}
	// 初始化缓存信息
	// 下载成功文件计数
	w.MaxCount = 0
	// 下载次数
	count := 0
	var idStrings string
	flg := false
	timeOutCH := time.After(time.Second * 60)
	for {

		if flg || count >= len(urls) {
			break
		}
		select {
		case name := <-cb:
			count++
			// 下载成功
			if name != "" {
				// 更换壁纸
				if w.MaxCount == 0 {
					w.ChangeWallpaper(fmt.Sprintf("%s/%s.jpg", w.ImgSavePath, name))
				}
				w.MaxCount++
				idStrings = fmt.Sprintf("%s,%s", idStrings, name)
			}
		case <-timeOutCH:
			flg = true
		default:
		}
	}
	idStrings = strings.TrimPrefix(idStrings, ",")

	// 保存配置
	cfg, err := ini.Load(w.CfgPath)
	if err != nil {
		log.Println("[err load cfg] \n", err)
	}
	cfg.Section("wallpaperTemp").Key(fmt.Sprintf("%sNowCount", w.Type)).SetValue("0")
	cfg.Section("wallpaperTemp").Key(fmt.Sprintf("%sImageIDs", w.Type)).SetValue(idStrings)
	err = cfg.SaveTo(w.CfgPath)
	if err != nil {
		log.Println("[err save cfg] \n", err)
	}
	return
}

func Run() {
	var flg bool
	flag.BoolVar(&flg, "n", false, "use -n change next wallpaper")
	flag.Parse()
	w := &Wallpaper{}
	w.LoadConfig()
	if !flg {
		// 加载热键
		w.Hotkey()
	}
	// 优先读取缓存
	if w.NextImageID != "" {
		log.Println("[info read temp] \n", w.NextImageID)
		w.ChangeWallpaper(fmt.Sprintf("%s/%s.jpg", w.ImgSavePath, w.NextImageID))

	} else {
		w.downloadAndSaveImages()
	}
}
