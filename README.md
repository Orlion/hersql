# hersql
通过http(s)隧道来访问Mysql
# 使用场景
假设您的测试环境只对外开放了http(s)服务，因此在本地无法直接访问到测试环境的诸多Mysql。您可以通过`hersql`来架设一条从本地到测试环境的隧道，通过该隧道来转发Mysql数据包，从而在本地能够直接访问到之前无法访问到的Mysql。

![hersql架构](https://github.com/Orlion/hersql/blob/main/resources/architecture.png)

# 数据流转过程

![image.png](https://s2.loli.net/2023/05/24/7ou64BgpsCjXOMz.png)

# 使用
## 1. 配置
`transport`的配置：
```
server:
  # transport http服务监听的地址
  addr: :8080

log:
  # 标准输出的日志的日志级别
  stdout_level: debug
  # 文件日志的日志级别
  level: error
  # 文件日志的文件地址
  filename: ./storage/transport.log
  # 日志文件的最大大小（以MB为单位）, 默认为 100MB。日志文件超过此大小会创建个新文件继续写入
  maxsize: 100
  # maxage 是根据文件名中编码的时间戳保留旧日志文件的最大天数。 
  maxage: 168
  # maxbackups 是要保留的旧日志文件的最大数量。默认是保留所有旧日志文件。
  maxbackups: 3
  # 是否应使用 gzip 压缩旋转的日志文件。默认是不执行压缩。
  compress: false
```
`sidecar`的配置：
```
server:
  # sidecar 监听的地址，之后mysql client会连接这个地址
  addr: 127.0.0.1:3306
  # transport http server的地址
  transport_addr: http://x.x.x.x:xxxx
  # 请求transport时是否绕过证书验证
  insecure_skip_verify: false
  # sidecar伪装mysql服务器版本，不同的mysql server版本有不同的特性，客户端可能会依赖mysql server版本，所以请尽量与被代理的mysql server保持相同的版本
  version: 8.0.11-hersql-0.1.0
log:
  # 与sidecar配置相同
```
## 2. 在一台能够请求目标mysql server的机器上部署hersql transport
```
$ git clone https://github.com/Orlion/hersql
$ vim transport.example.yaml // 按照需求修改配置文件
$ go run cmd/transport/main.go -conf=transport.example.yaml
```

> 建议先编译为可执行文件然后由systemd之类的工具托管transport进程，保证transport存活，这里简单起见直接用go run起来

## 3. 在本地机器部署启动hersql sidecar
```
$ git clone https://github.com/Orlion/hersql
$ vim transport.example.yaml // 按照需求修改配置文件
$ go run cmd/sidecar/main.go -conf=sidecar.example.yaml
```

> 还是建议先编译为可执行文件然后由launchctl之类的工具托管sidecar进程，保证sidecar存活，这里简单起见直接用go run起来

## 4. 客户端连接

上面的步骤都执行完成后，就可以打开mysql客户端使用了。数据库地址和端口号需要填写`sidecar`配置文件中的`addr`地址，`sidercar`不会校验用户名和密码，因此用户名密码可以随意填写

注意: **数据库名必须要填写，且必须要按照以下格式填写**
```
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
```
举个例子：
```
root:123456@tcp(10.10.123.123:3306)/BlogDB
```
如图所示：
![image.png](https://s2.loli.net/2023/05/24/YIQ51xFpEfMso7N.png)

> 

## 5. 举个例子
目标mysql服务器
* 地址：10.10.123.123:3306
* 数据库：BlogDB
* 用户名：root
* 密码：123456

可以直连目标mysql服务器的机器
* 地址：10.10.123.100
* 开放端口：8080

那么`transport`可以配置为
```
server:
  addr: :8080
```
`sidecar`可以配置为
```
server:
  addr: 127.0.0.1:3306
  transport_addr: http://10.10.123.100:8080
```
客户端连接配置
* 服务器地址：127.0.0.1
* 端口: 3306
* 数据库名`root:123456@tcp(10.10.123.123:3306)/BlogDB`


# 一些已知问题
1. 桌面客户端一般提供了快速取消执行的按钮，会执行kill query {thread_id}的命令，由于握手协议的限制客户端拿到的并非真实mysql server端的thread_id，因此可能会出现意想不到的问题
2. TablePlus 会忽略服务端的`CLIENT_DEPRECATE_EOF` Capability Flag，可能会导致一些包识别的问题；TablePlus会使用连接配置中的`Database`作为数据库执行类似于`SELECT table_name, table_type FROM information_schema.tables WHERE table_schema = '{Database}';`这样的SQL，但是`Database`我们又要求填写成`[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]`这样的格式，因此TablePlus上无法显示所有表
3. DBeaver 上如果被代理的mysql server版本为8.0+，而在sidecar.yaml中配置的server.version为\<8.0的版本，那么会报一些类似于`Unknown system variable 'query_cache_size'`这样高低版本不同导致的问题

# 站在巨人的肩膀上
本项目采用了[github.com/siddontang/mixer](https://github.com/siddontang/mixer)的mysql包下的代码，感谢。