package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

// 获取程序路径和名称
func getExePath() (string, string) {
	exePath, _ := os.Executable()
	fp, fn := filepath.Split(exePath)
	return fp, fn
}

func main() {
	// 设置日志输出
	logFile, err := os.OpenFile("tary.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("无法创建日志文件:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// 运行任务栏程序
	systray.Run(onReady, onExit)
}

func onReady() {
	// 设置图标和提示文字
	iconData, err := getIcon()
	if err != nil {
		// 记录错误但继续执行
		log.Printf("设置图标时出现错误: %v，但程序将继续运行", err)
	} else {
		systray.SetIcon(iconData)
	}

	systray.SetTitle("APIGO服务")
	systray.SetTooltip("APIGO服务管理工具")

	// 创建菜单项
	mStart := systray.AddMenuItem("启动服务", "启动APIGO服务")
	mStop := systray.AddMenuItem("停止服务", "停止APIGO服务")
	systray.AddSeparator()
	mConfig := systray.AddMenuItem("编辑配置", "编辑配置文件")
	mLog := systray.AddMenuItem("查看日志", "查看服务日志")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	// 处理菜单点击事件
	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				startService()
			case <-mStop.ClickedCh:
				stopService()
			case <-mConfig.ClickedCh:
				openConfig()
			case <-mLog.ClickedCh:
				openLog()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// 清理资源
	log.Println("退出程序")
}

// startService 启动APIGO服务
func startService() {
	log.Println("正在启动服务...")
	cmd := exec.Command("nssm", "start", "APIGO")
	if err := cmd.Run(); err != nil {
		log.Println("启动服务失败:", err)
	} else {
		log.Println("服务启动成功")
	}
}

// stopService 停止APIGO服务
func stopService() {
	log.Println("正在停止服务...")
	cmd := exec.Command("nssm", "stop", "APIGO")
	if err := cmd.Run(); err != nil {
		log.Println("停止服务失败:", err)
	} else {
		log.Println("服务停止成功")
	}
}

// openConfig 打开配置文件
func openConfig() {
	log.Println("正在打开配置文件...")
	fp, fn := getExePath()
	configPath := fp + strings.Replace(fn, ".exe", ".json", 1)

	// 打开配置文件
	if err := open.Run(configPath); err != nil {
		log.Println("打开配置文件失败:", err)
	}
}

// openLog 打开日志文件
func openLog() {
	log.Println("正在打开日志文件...")
	fp, _ := getExePath()
	logPath := filepath.Join(fp, "apigo.log")

	// 打开日志文件
	if err := open.Run(logPath); err != nil {
		log.Println("打开日志文件失败:", err)
	}
}

// getIcon 获取图标数据
func getIcon() ([]byte, error) {
	// 从文件中读取图标
	fp, _ := getExePath()
	iconPath := filepath.Join(fp, "src", "favicon.ico")

	// 尝试读取图标文件
	iconData, err := ioutil.ReadFile(iconPath)
	if err != nil {
		// 如果在可执行文件目录中找不到，尝试在当前目录下的src目录查找
		iconPath = filepath.Join("src", "favicon.ico")
		iconData, err = ioutil.ReadFile(iconPath)
		if err != nil {
			return nil, fmt.Errorf("读取图标文件失败: %w", err)
		}
	}

	return iconData, nil
}
