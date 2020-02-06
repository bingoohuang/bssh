https://xiking.win/2018/07/13/ssh-tunnel-proxy-and-port-forwarding/

SSH是一种安全的传输协议，ssh命令也是linux和mac上最常用的远程登录命令。出了远程登录，ssh还有一些其他的功能。比如隧道网络穿透。

什么是网络穿透？
其实网络穿透就是一种网络代理，比如在内网中有一台机器A，这台机器无法访问互联网，但是局域网内有一台其他机器B可以访问外网权限，所以我们可以把A的请求通过SSH隧道转发到机器B，机器B和互联网交互，数据再通过SSH隧道转发给机器A，这样机器A就可以通过可以连接外网的机器B来访问互联网了。

SSH中有两种端口转发模式：

本地端口转发(Local Port Forwarding)

从ssh client机器执行命令, ssh client没有直接访问host的权限，通过ssh_server来转发，ssh_server提供转发服务：

`ssh -L bind_addr:port:host:hostport user@ssh_server`

远程端口转发(Remote Port Forwarding)

从ssh client机器执行命令， ssh client有访问host权限，而ssh_server无访问权限，也就是ssh client要代理远程的访问，提供转发服务。

`ssh -R bind_addr:port:host:hostport user@ssh_server`

远程端口转发需要注意的是，命令需要早ssh server上监听端口，一些情况下可能需要root权限，所以普通用户可能登录ssh_server会无法监听端口。

没有什么是一张图说明不了的。

![image](https://user-images.githubusercontent.com/1940588/73902945-6ac74e00-48d2-11ea-80ba-7d583d59a73a.png)
