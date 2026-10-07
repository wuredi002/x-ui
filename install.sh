#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)

# check root
[[ $EUID -ne 0 ]] && echo -e "${red}错误：${plain} 必须使用root用户运行此脚本！\n" && exit 1

# check os
if [[ -f /etc/redhat-release ]]; then
    release="centos"
elif cat /etc/issue | grep -Eqi "debian"; then
    release="debian"
elif cat /etc/issue | grep -Eqi "ubuntu"; then
    release="ubuntu"
elif cat /etc/issue | grep -Eqi "centos|red hat|redhat"; then
    release="centos"
elif cat /proc/version | grep -Eqi "debian"; then
    release="debian"
elif cat /proc/version | grep -Eqi "ubuntu"; then
    release="ubuntu"
elif cat /proc/version | grep -Eqi "centos|red hat|redhat"; then
    release="centos"
else
    echo -e "${red}未检测到系统版本，请联系脚本作者！${plain}\n" && exit 1
fi

arch=$(arch)

if [[ $arch == "x86_64" || $arch == "x64" || $arch == "amd64" ]]; then
    arch="amd64"
elif [[ $arch == "aarch64" || $arch == "arm64" ]]; then
    arch="arm64"
elif [[ $arch == "s390x" ]]; then
    arch="s390x"
else
    arch="amd64"
    echo -e "${red}检测架构失败，使用默认架构: ${arch}${plain}"
fi

echo "架构: ${arch}"

if [ $(getconf WORD_BIT) != '32' ] && [ $(getconf LONG_BIT) != '64' ]; then
    echo "本软件不支持 32 位系统(x86)，请使用 64 位系统(x86_64)，如果检测有误，请联系作者"
    exit -1
fi

os_version=""

# os version
if [[ -f /etc/os-release ]]; then
    os_version=$(awk -F'[= ."]' '/VERSION_ID/{print $3}' /etc/os-release)
fi
if [[ -z "$os_version" && -f /etc/lsb-release ]]; then
    os_version=$(awk -F'[= ."]+' '/DISTRIB_RELEASE/{print $2}' /etc/lsb-release)
fi

if [[ x"${release}" == x"centos" ]]; then
    if [[ ${os_version} -le 6 ]]; then
        echo -e "${red}请使用 CentOS 7 或更高版本的系统！${plain}\n" && exit 1
    fi
elif [[ x"${release}" == x"ubuntu" ]]; then
    if [[ ${os_version} -lt 16 ]]; then
        echo -e "${red}请使用 Ubuntu 16 或更高版本的系统！${plain}\n" && exit 1
    fi
elif [[ x"${release}" == x"debian" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        echo -e "${red}请使用 Debian 8 或更高版本的系统！${plain}\n" && exit 1
    fi
fi

install_base() {
    if [[ x"${release}" == x"centos" ]]; then
        yum install wget curl tar unzip -y
    else
        apt install wget curl tar unzip -y
    fi
}

download_latest_xray() {
    local xray_arch
    case "$arch" in
        amd64) xray_arch="64" ;;
        arm64) xray_arch="arm64-v8a" ;;
        s390x) xray_arch="s390x" ;;
        *)
            echo -e "${red}不支持下载 Xray 的系统架构: ${arch}${plain}"
            return 1
            ;;
    esac

    xray_version=$(curl -fsSL https://api.github.com/repos/XTLS/Xray-core/releases/latest \
        | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p' | head -n 1)
    if [[ -z "$xray_version" ]]; then
        echo -e "${red}获取 Xray 最新稳定版失败${plain}"
        return 1
    fi

    xray_tmp_dir=$(mktemp -d)
    local xray_archive="Xray-linux-${xray_arch}.zip"
    local xray_url="https://github.com/XTLS/Xray-core/releases/download/${xray_version}/${xray_archive}"
    if ! curl -fLsS --retry 3 -o "${xray_tmp_dir}/${xray_archive}" "$xray_url"; then
        echo -e "${red}下载 Xray ${xray_version} 失败${plain}"
        rm -rf -- "$xray_tmp_dir"
        xray_tmp_dir=""
        return 1
    fi
    if ! unzip -oq "${xray_tmp_dir}/${xray_archive}" -d "$xray_tmp_dir"; then
        echo -e "${red}解压 Xray ${xray_version} 失败${plain}"
        rm -rf -- "$xray_tmp_dir"
        xray_tmp_dir=""
        return 1
    fi
    if [[ ! -s "${xray_tmp_dir}/xray" || ! -s "${xray_tmp_dir}/geosite.dat" || ! -s "${xray_tmp_dir}/geoip.dat" ]]; then
        echo -e "${red}Xray ${xray_version} 安装包缺少必要文件${plain}"
        rm -rf -- "$xray_tmp_dir"
        xray_tmp_dir=""
        return 1
    fi
    echo -e "检测到 Xray 最新稳定版：${xray_version}"
}

#This function will be called when user installed xray out of sercurity
config_after_install() {
    echo -e "${yellow}出于安全考虑，安装/更新完成后需要强制修改端口与账户密码${plain}"
    read -p "确认是否继续?[y/n]": config_confirm
    if [[ x"${config_confirm}" == x"y" || x"${config_confirm}" == x"Y" ]]; then
        read -p "请设置您的账户名:" config_account
        echo -e "${yellow}您的账户名将设定为:${config_account}${plain}"
        read -p "请设置您的账户密码:" config_password
        echo -e "${yellow}您的账户密码将设定为:${config_password}${plain}"
        read -p "请设置面板访问端口:" config_port
        echo -e "${yellow}您的面板访问端口将设定为:${config_port}${plain}"
        echo -e "${yellow}确认设定,设定中${plain}"
        /usr/local/xray/xray setting -username ${config_account} -password ${config_password}
        echo -e "${yellow}账户密码设定完成${plain}"
        /usr/local/xray/xray setting -port ${config_port}
        echo -e "${yellow}面板端口设定完成${plain}"
    else
        echo -e "${red}已取消,所有设置项均为默认设置,请及时修改${plain}"
    fi
}

install_xray() {
    systemctl stop xray || true
    systemctl stop x-ui 2>/dev/null || true
    systemctl disable x-ui 2>/dev/null || true
    if [[ ! -e /etc/xray/xray.db && -f /etc/x-ui/x-ui.db ]]; then
        mkdir -p /etc/xray
        cp -p /etc/x-ui/x-ui.db /etc/xray/xray.db
    fi
    cd /usr/local/

    if [ $# == 0 ]; then
        last_version=$(curl -Ls "https://api.github.com/repos/wuredi002/xray/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ ! -n "$last_version" ]]; then
            echo -e "${red}检测 xray 版本失败，可能是超出 Github API 限制，请稍后再试，或手动指定 xray 版本安装${plain}"
            exit 1
        fi
        echo -e "检测到 xray 最新版本：${last_version}，开始安装"
        wget -N --no-check-certificate -O /usr/local/xray-linux-${arch}.tar.gz https://github.com/wuredi002/xray/releases/download/${last_version}/xray-linux-${arch}.tar.gz
        if [[ $? -ne 0 ]]; then
            echo -e "${red}下载 xray 失败，请确保你的服务器能够下载 Github 的文件${plain}"
            exit 1
        fi
    else
        last_version=$1
        url="https://github.com/wuredi002/xray/releases/download/${last_version}/xray-linux-${arch}.tar.gz"
        echo -e "开始安装 xray v$1"
        wget -N --no-check-certificate -O /usr/local/xray-linux-${arch}.tar.gz ${url}
        if [[ $? -ne 0 ]]; then
            echo -e "${red}下载 xray v$1 失败，请确保此版本存在${plain}"
            exit 1
        fi
    fi

    download_latest_xray || exit 1

    if [[ -e /usr/local/xray/ ]]; then
        rm /usr/local/xray/ -rf
    fi

    tar zxvf xray-linux-${arch}.tar.gz
    rm xray-linux-${arch}.tar.gz -f
    cd xray
    install -m 755 "${xray_tmp_dir}/xray" "bin/xray-linux-${arch}"
    install -m 644 "${xray_tmp_dir}/geosite.dat" "bin/geosite.dat"
    install -m 644 "${xray_tmp_dir}/geoip.dat" "bin/geoip.dat"
    rm -rf -- "$xray_tmp_dir"
    xray_tmp_dir=""
    chmod +x xray bin/xray-linux-${arch}
    cp -f xray.service /etc/systemd/system/
    wget --no-check-certificate -O /usr/bin/xray https://raw.githubusercontent.com/wuredi002/xray/main/xray.sh
    chmod +x /usr/local/xray/xray.sh
    chmod +x /usr/bin/xray
    rm -f /usr/bin/x-ui
    config_after_install
    #echo -e "如果是全新安装，默认网页端口为 ${green}54321${plain}，用户名和密码默认都是 ${green}admin${plain}"
    #echo -e "请自行确保此端口没有被其他程序占用，${yellow}并且确保 54321 端口已放行${plain}"
    #    echo -e "若想将 54321 修改为其它端口，输入 xray 命令进行修改，同样也要确保你修改的端口也是放行的"
    #echo -e ""
    #echo -e "如果是更新面板，则按你之前的方式访问面板"
    #echo -e ""
    systemctl daemon-reload
    rm -f /etc/systemd/system/x-ui.service
    systemctl enable xray
    systemctl start xray
    echo -e "${green}xray v${last_version}${plain} 安装完成，面板已启动，"
    echo -e ""
    echo -e "xray 管理脚本使用方法: "
    echo -e "----------------------------------------------"
    echo -e "xray              - 显示管理菜单 (功能更多)"
    echo -e "xray start        - 启动 xray 面板"
    echo -e "xray stop         - 停止 xray 面板"
    echo -e "xray restart      - 重启 xray 面板"
    echo -e "xray status       - 查看 xray 状态"
    echo -e "xray enable       - 设置 xray 开机自启"
    echo -e "xray disable      - 取消 xray 开机自启"
    echo -e "xray log          - 查看 xray 日志"
    echo -e "xray v2-ui        - 迁移本机器的 v2-ui 账号数据至 xray"
    echo -e "xray update       - 更新 xray 面板"
    echo -e "xray install      - 安装 xray 面板"
    echo -e "xray uninstall    - 卸载 xray 面板"
    echo -e "----------------------------------------------"
}

echo -e "${green}开始安装${plain}"
install_base
install_xray $1
