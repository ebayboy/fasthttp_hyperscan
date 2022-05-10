# fasthttp_hyperscan

fasthttp + hyperscan

## Loglevel

## 架构
+ Data flow: dist_worker -> scan_worker -> scanners -> matchers
+ 一共有N个不同的matcher, 一个matcher对应一组规则,

## golang libs
+ 逻辑表达式计算:
    + govaluate : express compute
    + hyperscan 5.0 内置逻辑表达式
+ BODY:
    + 流式json解析： jsonparser gjson
    + xml解析
+ fasthttp: contain fasthttp goroutine pool
+ go-cache

## BufFix:
+ 解决多规则命中只打印一个规则问题

## TODO
+ pid写入到文件， 防止启动多个
+ 性能优化： 目前开启性能从7w下降到2w
+ 守护进程实现
+ 每个scanworker对一组matchers, 有多个scanworker
+ 一个数据包分发distworker对应多个scanworker对应
    +  distworker :qa

+ policiers封装实现 valueate
+ 并发处理： 函数调用方式（同步） &&  通道方式(异步)
+ 通道方式： 每个协程 +  一个通道 +  一个scratch
+ 支持变量： nginx原版变量

+ 匹配域matcher实现： 每个匹配域对应一个pool，可以配置worker数量(也不行， matcher需要单例的scratch）
+ json config 解析 及 模块化
    + 移植 statbot mutl goroutine model
    + MainConfig module
    + RulesConfig : Match zone && regex && flag && logic
    + HSMatcher 模块化 
+ 实现规则引擎RuleEngine
+ 实现逻辑引擎LogicEngine(govaluate)
+ 多match对输入数据包（URI、BODY、HEADER等）集中快速查找匹配问题
+ matcher 前面增加流式函数插件处理, 以及开关(spider_ip_rdns/router)
+ 原文匹配支持
+ 解码匹配支持 $u__
+ 自适应匹配
+ 协程池tunny实现对同一个matcher的匹配
+ 异步：通过通道传递数据到matcher的多个协程
+ 高效性能设计： 请求头+ URI + BODY 高效匹配设计
    + 请求头并发匹配(避免嵌套循环，计算消耗会指数下降)
    + 请求BODY多协程匹配
+ body的读取tunny pool

## TODO2:
+ apisix + fastscaner
+ 黑名单功能
+ 白名单功能

## DONE
+ worker dev && test  [done]
+ logrus + file-rotatelogs
+ fasthttp + hyperscan 
+ hyperscan static lib 
+ nginx性能测试 并发 + 延迟
+ fasthttp性能测试 并发 + 延迟
+ nginx + (http socket | unix socket)  + fasthttp 延迟测试

## OTHER
+ tunny goroutine pool(无法实现对每个matcher传递对应的独立scratch)

## hyperscan编译
+ hyperscan下载:https://launchpad.net/ubuntu/+archive/primary/+sourcefiles/hyperscan/5.4.0-2/hyperscan_5.4.0.orig.tar.gz
+ boost下载: https://boostorg.jfrog.io/artifactory/main/release/1.69.0/source/boost_1_69_0.tar.gz
+ pcre 8.45下载: https://sourceforge.net/projects/pcre/files/pcre/8.45/pcre-8.45.tar.bz2
+ 编译方法：
    + mkdir build && cd build
    + cmake .. -G Ninja -DBUILD_STATIC_LIBS=on
    + ninja && ninja install
    + go get -u -tags chimera github.com/flier/gohs/hyperscan

+ gohs使用
    + 需要指定hyerscan 安装的libhs.pc路径， libhs.pc文件包含库和头文件的路径, 默认安装到/usr/local/hyperscan
    + export PKG_CONFIG_PATH=/usrl/local/hyperscan
    + 多个协程使用一个scratch会报错： ERROR: Unable to scan input buffer. Exiting. err: The scratch region was already in use.

