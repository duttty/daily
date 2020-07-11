package wallpaper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type pixabay struct {
	AK       string
	Keywords string
}
type pixabayTemp struct {
	Hits []struct {
		ID            int    `json:"id"`
		FullHDURL     string `json:"fullHDURL"`
		LargeImageURL string `json:"largeImageURL"`
		PageURL       string `json:"pageURL"`
	} `json:"hits"`
}

func (u *pixabay) GetImagesURLs(num int) (imgs [][]string, err error) {

	url := fmt.Sprintf("https://pixabay.com/api/?key=%s&q=%s&per_page=%d&image_type=photo&orientation=horizontal&min_height=1280&order=latest",
		u.AK, u.Keywords, num)
	log.Println("[start GetImagesURLs] url \n", url)
	res, err := http.Get(url)
	if err != nil {
		log.Printf("[err GetImagesURLs],url\n%s\ncode:%d\n", url, res.StatusCode)
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("[err code]")
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("[err GetImagesURLs],ReadAll")
	}
	tp := &pixabayTemp{}
	imgs = make([][]string, num, 60)
	err = jsoniter.Unmarshal(b, &tp)
	if err != nil {
		log.Println("[err unmarshal] \n", err)
	}
	for k, v := range tp.Hits {
		imgs[k] = make([]string, 2, 2)
		imgs[k][0] = strconv.Itoa(v.ID)
		imgs[k][1] = strings.Replace(v.LargeImageURL, "s", "", 1)
	}
	log.Println("[info] get urls succ\n", len(imgs))
	return
}
