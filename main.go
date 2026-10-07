package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	_ "unsafe"
	"xray/config"
	"xray/database"
	"xray/logger"
	"xray/v2ui"
	"xray/web"
	"xray/web/global"
	"xray/web/service"

	"github.com/op/go-logging"
)

func runWebServer() {
	log.Printf("%v %v", config.GetName(), config.GetVersion())

	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		log.Fatal("unknown log level:", config.GetLogLevel())
	}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		log.Fatal(err)
	}

	var server *web.Server

	server = web.NewServer()
	global.SetWebServer(server)
	err = server.Start()
	if err != nil {
		log.Println(err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	//信号量捕获处理
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			err := server.Stop()
			if err != nil {
				logger.Warning("stop server err:", err)
			}
			server = web.NewServer()
			global.SetWebServer(server)
			err = server.Start()
			if err != nil {
				log.Println(err)
				return
			}
		default:
			server.Stop()
			return
		}
	}
}

func resetSetting() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	settingService := service.SettingService{}
	err = settingService.ResetSettings()
	if err != nil {
		fmt.Println("重置设置失败：", err)
	} else {
		fmt.Println("设置已重置")
	}
}

func showSetting(show bool) {
	if show {
		settingService := service.SettingService{}
		port, err := settingService.GetPort()
		if err != nil {
			fmt.Println("读取面板端口失败：", err)
		}
		userService := service.UserService{}
		userModel, err := userService.GetFirstUser()
		if err != nil {
			fmt.Println("读取用户信息失败：", err)
		}
		username := userModel.Username
		userpasswd := userModel.Password
		if (username == "") || (userpasswd == "") {
			fmt.Println("当前用户名或密码为空")
		}
		fmt.Println("当前面板设置：")
		fmt.Println("用户名：", username)
		fmt.Println("密码：", userpasswd)
		fmt.Println("端口：", port)
	}
}

func updateSetting(port int, username string, password string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	settingService := service.SettingService{}

	if port > 0 {
		err := settingService.SetPort(port)
		if err != nil {
			fmt.Println("设置端口失败：", err)
		} else {
			fmt.Printf("面板端口已设置为 %v\n", port)
		}
	}
	if username != "" || password != "" {
		userService := service.UserService{}
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("设置用户名或密码失败：", err)
		} else {
			fmt.Println("用户名和密码已更新")
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		runWebServer()
		return
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "显示版本")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	v2uiCmd := flag.NewFlagSet("v2-ui", flag.ExitOnError)
	var dbPath string
	v2uiCmd.StringVar(&dbPath, "db", "/etc/v2-ui/v2-ui.db", "设置 v2-ui 数据库路径")

	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	var port int
	var username string
	var password string
	var reset bool
	var show bool
	settingCmd.BoolVar(&reset, "reset", false, "重置所有设置")
	settingCmd.BoolVar(&show, "show", false, "显示当前设置")
	settingCmd.IntVar(&port, "port", 0, "设置面板端口")
	settingCmd.StringVar(&username, "username", "", "设置登录用户名")
	settingCmd.StringVar(&password, "password", "", "设置登录密码")

	runCmd.Usage = func() {
		fmt.Println("用法：xray run")
	}
	v2uiCmd.Usage = func() {
		fmt.Println("用法：xray v2-ui [-db 数据库路径]")
		v2uiCmd.PrintDefaults()
	}
	settingCmd.Usage = func() {
		fmt.Println("用法：xray setting [选项]")
		settingCmd.PrintDefaults()
	}
	flag.Usage = func() {
		fmt.Println("用法：xray [选项] <命令>")
		fmt.Println("全局选项：")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("可用命令：")
		fmt.Println("    run            启动面板")
		fmt.Println("    v2-ui          从 v2-ui 导入数据")
		fmt.Println("    setting        管理面板设置")
	}

	flag.Parse()
	if showVersion {
		fmt.Println(config.GetVersion())
		return
	}

	switch os.Args[1] {
	case "run":
		err := runCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		runWebServer()
	case "v2-ui":
		err := v2uiCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		err = v2ui.MigrateFromV2UI(dbPath)
		if err != nil {
			fmt.Println("migrate from v2-ui failed:", err)
		}
	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		if reset {
			resetSetting()
		} else {
			updateSetting(port, username, password)
		}
		if show {
			showSetting(show)
		}
	default:
		fmt.Println("请使用 run、v2-ui 或 setting 命令")
		fmt.Println()
		runCmd.Usage()
		fmt.Println()
		v2uiCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}
}
