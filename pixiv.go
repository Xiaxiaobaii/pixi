package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"pixi/lib"
	"syscall"
	"time"
)

type Config struct {
	OnlyBad   bool
	Debug     bool
	Proxy     string
	ProxyType string
	Thread    int
}

func main() {
	var config = Config{}
	config.OnlyBad = *flag.Bool("onlyBad", false, "启用onlyBad")

	flag.BoolVar(&config.Debug, "debug", false, "启用debug模式")

	config.ProxyType = *flag.String("proxyType", "none", "指定代理类型，none则不代理")

	config.Proxy = *flag.String("proxy", "", "设置代理地址")

	Thread := flag.Int("thread", 10, "设置并行线程数")

	homeid := ""
	cookie := ""
	fmt.Printf("请输入主页id: ")
	fmt.Scanln(&homeid)
	fmt.Printf("请输入cookie: ")
	fmt.Scanln(&cookie)

	user := pixi.NewUser(homeid, cookie, config.ProxyType, config.Proxy)
	startTime := time.Now()

	fmt.Println("开始清除黑名单及非法文件")
	// 删除未在数据库中存在的文件
	pixi.DeleteBadImageFromRootfs()
	fmt.Println("黑名单与非法文件清除完毕")
	if config.OnlyBad {
		os.Exit(0)
	}
	uids, names, err := user.GetUserid(homeid)
	if err != "" {
		fmt.Println("GetUserId Error: ", err)
		os.Exit(1)
	}
	fmt.Println("User数据已获取")
	go ExitTunnel()
	fmt.Println("总关注数：", len(uids))
	for i := range uids {
		uidfile := fmt.Sprintf("img/%s", uids[i])
		if !pixi.ExistFile(uidfile) {
			os.Mkdir(pixi.Root+uidfile, 0755)
		}
		pids := user.GetUserPid(uids[i])
		fmt.Printf("开始下载：%s(%s)\n", names[i], uids[i])
		for _, pid := range pids {
			if pixi.ExistBad(pid) {
				if pixi.ExistFile("img/" + uids[i] + "/" + pid) {
					_ = os.Remove(pixi.Root + "img/" + uids[i] + "/" + pid)
					fmt.Println("文件夹检测到非法图片：" + pid + " 已删除")
				}
				continue
			} else if pixi.ExistSQL(pid) {
				continue
			}
			if pixi.Wgd > *Thread {
				fmt.Println("等待过往协程结束...")
				pixi.Wg.Wait()
			}
			user.WgDownloadImg(pid)
		}
		pixi.Wg.Wait()
		fmt.Printf("下载完毕: %s(%s)\n", names[i], uids[i])
	}
	fmt.Println("Download Pixiv Home jpg All Success!")
	runTime := time.Since(startTime)
	fmt.Println("Running Time is " + runTime.String())
	fmt.Println("please close window")
	fmt.Println("If you not close window,It will automatically shut down in 6 hours")
	time.Sleep(time.Minute * 360)
}

func ExitTunnel() {
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-exitChan
	fmt.Println("即将清理碎片并退出...")
	pixi.Wg.Wait()
	fmt.Println("by:侍者")
	fmt.Println("good bye")
	time.Sleep(2 * time.Second)
	os.Exit(0)
}
