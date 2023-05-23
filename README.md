# hersql
通过http(s)隧道来访问Mysql
# 使用场景
假设您的测试环境只对外开放了http(s)服务，因此在本地无法直接访问到测试环境的诸多Mysql。您可以通过`hersql`来架设一条从本地到测试环境的隧道，通过该隧道来转发Mysql数据包，从而在本地能够直接访问到之前无法访问到的Mysql。

![hersql架构](https://github.com/Orlion/hersql/blob/main/resources/architecture.png)

# 使用
1. 在一台可以直连目标Mysql的机器上运行hersql transport
```
$ go run cmd/transport/main.go -conf=transport.example.yaml
```
2. 在本地运行hersql sidecar
```
$ go run cmd/sidecar/main.go -conf=sidecar.example.yaml
```
3. 在mysql客户端连接配置中，服务器地址与端口号按照`sidecar.example.yaml`中的`addr`填写，数据库必须要填写成dsn，格式：
```
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
```
eg.
```
root:123456@tcp(localhost:3306)/blog?param=value
```
hersql sidecar会从握手包中解析dsn然后告知hersql transport连接到解析出的目标Mysql服务器
# 站在巨人的肩膀上
本项目代码采用了大量[github.com/siddontang/mixer](https://github.com/siddontang/mixer)的代码，在此表示感谢