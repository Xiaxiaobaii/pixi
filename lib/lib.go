package pixi

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tool "github.com/Xiaxiaobaii/autotool"
	"github.com/Xiaxiaobaii/lrys"
	"github.com/Xiaxiaobaii/sqlite"
)

var Wg = sync.WaitGroup{}
var Wgd int
var Debug bool
var SQL *sqlite.Sql
var Blacklist *sqlite.Sql
var Root = tool.Findfile()

type User struct {
	homeid         string
	cookie         string
	proxyType      string
	proxyIp        string
	Header         map[string]string
	NoCookieHeader map[string]string
}

var PixivType map[string]sqlite.DataType = map[string]sqlite.DataType{
	"name":       sqlite.TEXT,
	"id":         sqlite.TEXT,
	"Author":     sqlite.TEXT,
	"Authorid":   sqlite.TEXT,
	"R18":        sqlite.INT,
	"createDate": sqlite.TEXT,
	"tags":       sqlite.TEXT,
	"size":       sqlite.TEXT,
}

type ImgBody struct {
	AiType        int       `json:"aiType"`
	Alt           string    `json:"alt"`
	CreateDate    time.Time `json:"createDate"`
	Description   string    `json:"description"`
	Height        int       `json:"height"`
	ID            string    `json:"id"`
	IllustComment string    `json:"illustComment"`
	IllustID      string    `json:"illustId"`
	IllustTitle   string    `json:"illustTitle"`
	IllustType    int       `json:"illustType"`
	LikeCount     int       `json:"likeCount"`
	LikeData      bool      `json:"likeData"`
	PageCount     int       `json:"pageCount"`
	Sl            int       `json:"sl"`
	Tags          ImgTags   `json:"tags"`
	R18           int
	Size          string
	Title         string    `json:"title"`
	UploadDate    time.Time `json:"uploadDate"`
	Urls          ImgUrls   `json:"urls"`
	UserAccount   string    `json:"userAccount"`
	UserID        string    `json:"userId"`
	UserName      string    `json:"userName"`
	ViewCount     int       `json:"viewCount"`
	Width         int       `json:"width"`
}

type ImgTags struct {
	AuthorID string    `json:"authorId"`
	IsLocked bool      `json:"isLocked"`
	Tags     []TagItem `json:"tags"`
	Writable bool      `json:"writable"`
}

type TagItem struct {
	Tag string `json:"tag"`
}

type ImgUrls struct {
	Mini     string `json:"mini"`
	Thumb    string `json:"thumb"`
	Small    string `json:"small"`
	Regular  string `json:"regular"`
	Original string `json:"original"`
}

func init() {
	SQL = sqlite.New("pixiv.db")
	SQL.CreateTables(map[string]map[string]sqlite.DataType{
		"imgs": PixivType,
		"star": PixivType,
	})
	Blacklist = sqlite.New("Blacklist.db")
	Blacklist.CreateTable("bad", map[string]sqlite.DataType{
		"id": sqlite.TEXT,
	})

	if !ExistFile("img") {
		os.Mkdir(Root+"img", 0755)
	}
}

func NewUser(homeid, cookie, proxyType, proxyIp string) User {
	return User{
		homeid:         homeid,
		cookie:         cookie,
		proxyType:      proxyType,
		proxyIp:        proxyIp,
		Header:         GetHeader("", cookie),
		NoCookieHeader: GetHeader("", ""),
	}
}

func GetHeader(pattern, cookie string) map[string]string {
	if cookie == "" {
		return map[string]string{
			"User-Agent": lrys.GetUa(),
			"Referer":    "https://www.pixiv.net/" + pattern,
		}
	}
	return map[string]string{
		"User-Agent": lrys.GetUa(),
		"Cookie":     cookie,
		"Referer":    "https://www.pixiv.net/" + pattern,
	}
}

func DeleteBadImageFromRootfs() {
	Dirs, _ := os.ReadDir(Root + "img/")
	for _, i := range Dirs {
		if i.IsDir() {
			uid := i.Name()
			fmt.Printf("检查uid: %s\n", uid)
			Dir, _ := os.ReadDir(Root + "img/" + uid)

			for _, i := range Dir {
				pid := i.Name()[0 : len(i.Name())-4]
				if !ExistSQL(pid) {
					e := os.Remove(Root + "img/" + uid + "/" + i.Name())
					if e != nil {
						fmt.Printf("Warn: 删除图片错误: %v(%s)\n", e, pid)
					} else {
						fmt.Println("文件夹检测到非法图片：" + pid + " 已删除")
					}
				}
			}

		}
	}
}

func (user User) GetUserid(userid string) ([]string, []string, string) {
	list_id := make([]string, 0)
	list_name := make([]string, 0)

	url := "https://www.pixiv.net/ajax/user/" + userid + "/following?offset=0&limit=99&rest=show"
	data, err := user.Download("GET", url, user.Header)
	if (err != nil && err != io.EOF) || data.StatusCode != 200 {
		fmt.Println("err: ", err)
		time.Sleep(time.Second * 5)
		return user.GetUserid(userid)
	}
	defer data.Body.Close()
	Body, _ := io.ReadAll(data.Body)

	var Data struct {
		Body struct {
			Users []struct {
				UserId   string `json:"userId"`
				UserName string `json:"userName"`
			} `json:"users"`
		} `json:"body"`
		Err     bool   `json:"error"`
		Message string `json:"message"`
	} = struct {
		Body struct {
			Users []struct {
				UserId   string "json:\"userId\""
				UserName string "json:\"userName\""
			} "json:\"users\""
		} "json:\"body\""
		Err     bool   "json:\"error\""
		Message string "json:\"message\""
	}{}
	json.Unmarshal(Body, &Data)
	if Data.Err {
		return []string{}, []string{}, Data.Message
	}
	users := Data.Body.Users
	for _, i := range users {
		list_id = append(list_id, i.UserId)
		list_name = append(list_name, i.UserName)
	}
	return list_id, list_name, ""
}

func (user User) GetUserPid(uid string) []string {
	relist := make([]string, 0)

	url := "https://www.pixiv.net/ajax/user/" + uid + "/profile/all"
	data, err := user.Download("GET", url, user.Header)
	if (err != nil && err != io.EOF) || data.StatusCode != 200 {
		time.Sleep(time.Second * 5)
		return user.GetUserPid(uid)
	}
	defer data.Body.Close()
	Body, _ := io.ReadAll(data.Body)
	var Data map[string]any
	json.Unmarshal(Body, &Data)
	rs := Data["body"].(map[string]any)["illusts"].(map[string]any)
	for i := range rs {
		relist = append(relist, i)
	}
	return relist
}

func (user User) Download(method, Url string, Header map[string]string) (data *http.Response, err error) {
	if Header == nil {
		Header = user.NoCookieHeader
	}
	switch user.proxyType {
	case "http":
		return lrys.HttpRequest(method, Url, user.proxyIp, true, Header)
	default:
		return lrys.HttpRequest(method, Url, "", false, Header)
	}
}

func (user User) GetImgUrl(pid string) ImgBody {
	Url := "https://www.pixiv.net/ajax/illust/" + pid
	data, err := user.Download("GET", Url, user.Header)
	if (err != nil && err != io.EOF) || data.StatusCode != 200 {
		fmt.Printf("Warn: GetImgUrl err: %v\n", err)
		time.Sleep(time.Second * 20)
		return user.GetImgUrl(pid)
	}
	defer data.Body.Close()
	Body, _ := io.ReadAll(data.Body)
	var DataMap struct {
		Body ImgBody `json:"body"`
	}
	_ = json.Unmarshal(Body, &DataMap)
	Data := DataMap.Body
	for _, i := range Data.Tags.Tags {
		switch i.Tag {
		case "R-18":
			Data.R18 = 1
			return Data
		case "R-18G":
			Data.R18 = 2
			return Data
		}
	}
	Data.R18 = 0
	return Data
}

func (user User) DownloadImg(url, pattern string) (int, int) {
	data, err := user.Download("GET", url, user.Header)
	if err != nil || data.StatusCode != 200 {
		fmt.Printf("Warn: DownloadImg err: %v\n", err)
		time.Sleep(time.Second * 20)
		return user.DownloadImg(url, pattern)
	}
	defer data.Body.Close()
	Body, _ := io.ReadAll(data.Body)

	file, err := os.OpenFile(pattern, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		fmt.Printf("Error: %s File Create error: %v\n", pattern, err)
		return user.DownloadImg(url, pattern)
	}
	file.Write(Body)
	file.Close()
	return GetWidthHeightForJpg(pattern)
}

func WindowsFileRe(str string) string {
	ill := []string{
		"<", ">", ":", "\"", "/", "\\", "|", "?", "*",
	}
	for _, i := range ill {
		str = strings.ReplaceAll(str, i, "-")
	}
	return str
}

func ReBuild(types, arg string) string {
	switch types {
	case "artworks":
		return "artworks/" + arg
	case "users":
		return fmt.Sprintf("user/%s/following", arg)
	default:
		return ""
	}
}

func ExistFile(name string) bool {
	_, err := os.Stat(Root + name)
	return err == nil
}

func GetWidthHeightForJpg(pattern string) (int, int) {
	f, _ := os.Open(pattern)
	c, _, _ := image.DecodeConfig(f)
	return c.Width, c.Height
}

// 检查id是否在黑名单中
func ExistBad(pid string) bool {
	return Blacklist.CheckData("bad", "id = '"+pid+"'")
}

func ExistSQL(pid string) bool {

	return SQL.CheckData("imgs", "id = '"+pid+"'")
}

func (user User) DownloadImgs(pid string) {
	Data := user.GetImgUrl(pid)
	x, y := user.DownloadImg(Data.Urls.Original, Root+"/img/"+Data.UserID+"/"+Data.ID+".jpg")
	size := strconv.Itoa(x) + "x" + strconv.Itoa(y)
	fmt.Printf("Download Success %s\n", Data.Title)
	tags := make([]string, len(Data.Tags.Tags))
	for _, i := range Data.Tags.Tags {
		tags = append(tags, i.Tag)
	}

	SqlData := map[string]any{
		"name":       Data.Title,
		"id":         Data.ID,
		"Author":     Data.UserName,
		"Authorid":   Data.UserID,
		"R18":        Data.R18,
		"createDate": Data.CreateDate.String(),
		"tags":       fmt.Sprintf("[%s]", strings.Join(tags, ",")),
		"size":       size,
	}
	SQL.Insert("imgs", SqlData, sqlite.ORREPLACE)
	Wgd -= 1
	Wg.Done()
}

func (user User) WgDownloadImg(pid string) {
	Wg.Add(1)
	Wgd += 1
	go user.DownloadImgs(pid)

}
