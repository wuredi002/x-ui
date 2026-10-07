# Xray 面板

支持多协议多用户的 xray 面板

# 功能介绍

- 系统状态监控
- 支持多用户多协议，网页可视化操作
- 支持的协议：vmess、vless、trojan、shadowsocks（含 2022）、Hysteria 2、dokodemo-door、socks、http
- 支持 tcp、kcp、ws、http、quic、grpc 和 xhttp 传输，以及 VLESS Reality
- 支持生成 Clash Meta（Mihomo）订阅，包含 VMess、VLESS、Trojan、Shadowsocks 和 Hysteria 2 节点
- 分享链接和订阅节点名格式为“国家-公网 IPv4-wuredi002-网路跳越-节点备注”
- 在入站列表点击“Clash Meta 订阅”即可复制订阅地址；该地址包含访问令牌，请勿公开分享
- TLS 可配置最低/最高版本及 Cipher Suites；Reality 支持随机生成伪装目标/SNI 与 uTLS 分享链接指纹
- 可配置嗅探目标类型、metadataOnly 与 routeOnly
- 简体中文界面，不限制设备数量
- 安装时自动安装 Xray 最新稳定版
- 流量统计，限制流量，限制到期时间
- 可自定义 xray 配置模板
- 支持 https 访问面板（自备域名 + ssl 证书）
- 支持一键SSL证书申请且自动续签
- 更多高级配置项，详见面板

# 安装&升级

```
bash <(curl -Ls https://raw.githubusercontent.com/wuredi002/xray/main/install.sh)
```

## 手动安装&升级

1. [releases](https://github.com/wuredi002/xray/releases)下载最新的压缩包，一般选择`amd64`架构
2. 然后将这个压缩包上传到服务器的`/root/`目录下，并使用`root`用户登录服务器

> 如果你的服务器 cpu 架构不是 `amd64`，自行将命令中的 `amd64`替换为其他架构

```
cd /root/
rm xray/ /usr/local/xray/ /usr/bin/xray -rf
tar zxvf xray-linux-amd64.tar.gz
chmod +x xray/xray xray/bin/xray-linux-* xray/xray.sh
cp xray/xray.sh /usr/bin/xray
cp -f xray/xray.service /etc/systemd/system/
mv xray/ /usr/local/
systemctl daemon-reload
systemctl enable xray
systemctl restart xray
```

## 使用docker安装

> 此 docker 教程与 docker 镜像由[Chasing66](https://github.com/Chasing66)提供

1. 安装docker

```shell
curl -fsSL https://get.docker.com | sh
```

2. 安装xray

```shell
git clone https://github.com/wuredi002/xray.git xray-panel
cd xray-panel
docker build -t xray-panel:latest .
docker run -itd --network=host \
    -v $PWD/db/:/etc/xray/ \
    -v $PWD/cert/:/root/cert/ \
    --name xray-panel --restart=unless-stopped \
    xray-panel:latest
```

## SSL证书申请

> 此功能与教程由[FranzKafkaYu](https://github.com/FranzKafkaYu)提供

脚本内置SSL证书申请功能，使用该脚本申请证书，需满足以下条件:

- 知晓Cloudflare 注册邮箱
- 知晓Cloudflare Global API Key
- 域名已通过cloudflare进行解析到当前服务器

获取Cloudflare Global API Key的方法:
    ![](media/bda84fbc2ede834deaba1c173a932223.png)
    ![](media/d13ffd6a73f938d1037d0708e31433bf.png)

使用时只需输入 `域名`, `邮箱`, `API KEY`即可，示意图如下：
        ![](media/2022-04-04_141259.png)

注意事项:

- 该脚本使用DNS API进行证书申请
- 默认使用Let'sEncrypt作为CA方
- 证书安装目录为/root/cert目录
- 本脚本申请证书均为泛域名证书

## 建议系统

- CentOS 7+
- Ubuntu 16+
- Debian 8+

# 常见问题

## 从 v2-ui 迁移

首先在安装了 v2-ui 的服务器上安装最新版 xray，然后使用以下命令进行迁移，将迁移本机 v2-ui 的 `所有 inbound 账号数据`至 xray，`面板设置和用户名密码不会迁移`

> 迁移成功后请 `关闭 v2-ui`并且 `重启 xray`，否则 v2-ui 的 inbound 会与 xray 的 inbound 会产生 `端口冲突`

```
xray v2-ui
```

## issue 关闭

各种小白问题看得血压很高

## Stargazers over time

[![Stargazers over time](https://starchart.cc/wuredi002/xray.svg)](https://starchart.cc/wuredi002/xray)
