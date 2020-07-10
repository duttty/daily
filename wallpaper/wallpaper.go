package wallpaper

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/ini.v1"
)

type Wallpaper struct {
	Type        string
	AK          string
	Keywords    string
	ID          string
	ImageURL    string
	Description string
	Dir         string
	ExePath     string
}

type WallpaperConfig struct {
	PixabayAK  string
	UnsplashAK string
	Type       string
	Keywords   string
}

type Wallpaperface interface {
	GetImageInfo(w *Wallpaper)
}
type pixRes struct {
	Hits []struct {
		ID            int    `json:"id"`
		FullHDURL     string `json:"fullHDURL"`
		LargeImageURL string `json:"largeImageURL"`
		PageURL       string `json:"pageURL"`
	} `json:"hits"`
}

func (r *pixRes) GetImageInfo(w *Wallpaper) {
	var l string
	// 保存图片信息 id与url用,分离
	for _, v := range r.Hits {
		v.LargeImageURL = strings.Replace(v.LargeImageURL, "s", "", 1)
		l = fmt.Sprintf("%s\n%d,%s", l, v.ID, v.LargeImageURL)
	}
	w.ID = strconv.Itoa(r.Hits[0].ID)
	w.ImageURL = r.Hits[0].LargeImageURL

	// 保存配置
	fn := fmt.Sprintf("%s/temp.ini", w.Dir)
	cfg, err := ini.Load(fn)
	if err != nil {
		log.Println("[err GetImageInfo],Load err:\n", err)
	}

	cfg.Section("wallpaper").Key("pixabayList").SetValue(l)
	cfg.Section("wallpaper").Key("pixabayCount").SetValue("1")
	cfg.Section("wallpaper").Key("pixabayLen").SetValue(strconv.Itoa(len(r.Hits)))
	err = cfg.SaveTo(fn)
	if err != nil {
		log.Println("[err GetImageInfo],SaveTo:\n", err)
	}

}

type unsRes struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Urls        struct {
		Regular string `json:"regular"`
	} `json:"urls"`
}

func (r *unsRes) GetImageInfo(w *Wallpaper) {
	w.ID = r.ID
	w.ImageURL = strings.Replace(r.Urls.Regular, "s", "", 1)
	w.Description = r.Description
	return
}

// GetURL 根据关键字获取图片的URL
func (w *Wallpaper) GetURL() {
	fmt.Println()
	log.Println("[log GetURL],START")

	// 从缓存读取
	cfg, err := ini.Load(fmt.Sprintf("%s/temp.ini", w.Dir))
	if err != nil {
		log.Println("[err GetURL],Load:\n", err)
	}

	var imgURL string
	var r Wallpaperface
	switch w.Type {
	case "pixabay":
		// 缓存足够读取缓存
		count, err := cfg.Section("wallpaper").Key("pixabayCount").Int()
		pixLen, err := cfg.Section("wallpaper").Key("pixabayLen").Int()
		if err != nil {
			log.Println("[err GetURL],Int:\n", err)
		}
		if count < pixLen {
			count++
			pixList := cfg.Section("wallpaper").Key("pixabayList").Strings("\n")
			fmt.Println(pixList)
			pixInfo := strings.Split(pixList[count], ",")
			w.ID = pixInfo[0]
			w.ImageURL = pixInfo[1]
			cfg.Section("wallpaper").Key("pixabayCount").SetValue(strconv.Itoa(count))
			err = cfg.SaveTo(fmt.Sprintf("%s/temp.ini", w.Dir))
			if err != nil {
				log.Println("[err GetURL],Save pixabayCount:\n", err)
			}
			return
		}
		r = &pixRes{}
		imgURL = fmt.Sprintf("https://pixabay.com/api/?key=%s&q=%s&image_type=photo&orientation=horizontal&min_height=1920&order=latest&pretty=true",
			w.AK, w.Keywords)

	default:
		r = &unsRes{}
		imgURL = fmt.Sprintf("https://api.unsplash.com/photos/random?query=%s&orientation=landscape&featured=true&client_id=%s",
			w.Keywords, w.AK)
	}
	log.Println("[log GetURL],imgURL:\n", imgURL)
	res, err := http.Get(imgURL)
	if err != nil {
		log.Println("[err GetURL],err imgURL:\n", imgURL)
		return
	}
	defer res.Body.Close()
	bRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("[err GetURL],err ReadAll:\n", err)
		return
	}
	err = jsoniter.Unmarshal(bRes, r)
	if err != nil {
		log.Println("[err GetURL],err Unmarshal:\n", err)
		return
	}
	r.GetImageInfo(w)
	log.Println("[log GetURL],ImageURL:\n", w.ImageURL)
	fmt.Println("[log GetURL],Succ ID = ", w.ID)
	fmt.Print("\n")
}

func (w *Wallpaper) Download() {
	fmt.Println()
	log.Println("[log Download],START")
	res, err := http.Get(w.ImageURL)
	if err != nil {
		log.Println("[err Download],Get ImageURL err:\n", err)
		return
	}
	log.Println("[log Download],ImageURL:\n", w.ImageURL)
	defer res.Body.Close()
	// 查看文件夹
	err = os.Mkdir(fmt.Sprintf("%s/images", w.Dir), 0777)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Fatalln("[err download],Mkdir err:\n", err)
		}
	}
	fName := fmt.Sprintf("%s/images/%s.jpg", w.Dir, w.ID)
	file, err := os.Create(fName)
	if err != nil {
		log.Println("[err Download],Create file err:\n, ", err)
		return
	}
	defer file.Close()
	_, err = io.Copy(file, res.Body)
	if err != nil {
		log.Println("[err Download],Copy file err:\n", err)
		return
	}
	fmt.Println()
	log.Println("[log Download],SUCC")
}

func (w *Wallpaper) ChangeWallpaper() {
	fpath := filepath.ToSlash(fmt.Sprintf("%s/images/%s.jpg", w.Dir, w.ID))
	_, err := os.Stat(fpath)
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalln("[err ChangeWallpaper],can't find file.")
		return
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
		log.Fatalln(r2, err, fpath)
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
	nk.SetStringValue("", w.ExePath)

}
func (w *Wallpaper) Run() {
	// 加载配置
	w.LoadConfig()

	w.GetURL()
	w.Download()
	w.ChangeWallpaper()

}

func (w *Wallpaper) LoadConfig() {
	// 读取文件执行路径
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	w.ExePath = path
	dir := filepath.Dir(path)
	w.Dir = dir

	//读取wallpaper配置

	cfg, err := ini.Load(fmt.Sprintf("%s/config.ini", w.Dir))
	if err != nil {
		log.Fatalf("[err LoadConfig],Load path:\n%s\n,%s\n", w.Dir, err)
	}
	w.Type = cfg.Section("wallpaper").Key("type").String()
	w.Keywords = cfg.Section("wallpaper").Key("keywords").String()
	switch w.Type {
	case "pixabay":
		w.AK = cfg.Section("wallpaper").Key("pixabayAK").String()
		// 格式化关键字
		w.Keywords = strings.ReplaceAll(w.Keywords, ",", "+")
	default:
		w.AK = cfg.Section("wallpaper").Key("unsplashAK").String()
	}

}
