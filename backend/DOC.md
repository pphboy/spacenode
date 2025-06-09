## 技术

1. win(wintun), linux, 以及容器内的运行的，都是此局域网中的一个结点

## 流程

1. 写一个配置文件

current_app: fiai.lazycat.
service: app
moon_app: foward.xxx:8080
net: 
  type: ipv4,ipv6,ALL
  ip_req: auto,static
  ip_addr: ipv4

2. app_client -> moon_server

3. moon_server -> ip -> app_client

4. moon_server -> ip -> choose app ->  app service
5. app service -> moon_server ->  ip -> forward_client 

## 配置
