package wallpaper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type unsplash struct {
	AK       string
	Keywords string
	Res      []unsplashTemp
}
type unsplashTemp struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Urls        struct {
		Regular string `json:"regular"`
	} `json:"urls"`
}

func (u *unsplash) GetImagesURLs(num int) (imgs [][]string, err error) {

	log.Println("[start GetImagesURLs]")

	url := fmt.Sprintf("https://api.unsplash.com/photos/random?count=%d&query=%s&orientation=landscape&featured=true&client_id=%s",
		num, u.Keywords, u.AK)
	res, err := http.Get(url)
	if err != nil {
		log.Println("[err GetImagesURLs] url\n", url)
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

	u.Res = make([]unsplashTemp, 0, 60)
	imgs = make([][]string, num, 60)
	jsoniter.Unmarshal(b, &u.Res)
	for k, v := range u.Res {
		imgs[k] = make([]string, 2, 2)
		imgs[k][0] = v.ID
		imgs[k][1] = strings.Replace(v.Urls.Regular, "s", "", 1)
	}

	return
}
