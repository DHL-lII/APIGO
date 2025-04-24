#!/bin/bash

echo "APIGO 编译脚本"
echo "==============================="

# 设置环境变量
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# 创建输出目录
mkdir -p build

echo "正在编译主程序 m.go..."
go build -o build/m src/m.go src/nx.go src/wx.go
if [ $? -ne 0 ]; then
    echo "编译主程序失败！"
    exit 1
else
    echo "编译主程序成功：build/m"
fi

echo "正在编译任务栏程序 tary.go..."
go build -o build/tary tary.go
if [ $? -ne 0 ]; then
    echo "编译任务栏程序失败！"
    exit 1
else
    echo "编译任务栏程序成功：build/tary"
fi

echo "正在复制配置文件..."
cp src/m.json build/m.json
if [ $? -ne 0 ]; then
    echo "复制配置文件失败！"
    exit 1
else
    echo "复制配置文件成功：build/m.json"
fi

echo "正在复制测试页面..."
cp test.html build/test.html
if [ $? -ne 0 ]; then
    echo "复制测试页面失败！"
    exit 1
else
    echo "复制测试页面成功：build/test.html"
fi

echo "正在设置可执行权限..."
chmod +x build/m
chmod +x build/tary

echo "编译和复制完成！"
echo "所有文件已生成到 build 目录"