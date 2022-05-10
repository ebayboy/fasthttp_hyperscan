package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"syscall"

	"github.com/flier/gohs/hyperscan"
	"github.com/valyala/fasthttp"
)

//TODO: 目前lighthttpd是多协程使用 一个scratch， 会导致scrach in use 错误

var (
	addr     = flag.String("addr", ":9999", "TCP address to listen to")
	compress = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

var hsMatcher HSMatcher

func init() {
	hsMatcher.Init()
}

func main() {

	//set rlimit
	var rLimit syscall.Rlimit
	rLimit.Cur = 65535
	rLimit.Max = 65535
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Fatal("err:", err.Error())
	}

	//set procx
	runtime.GOMAXPROCS(4)

	flag.Parse()

	h := requestHandler
	if *compress {
		h = fasthttp.CompressHandler(h)
	}

	if err := fasthttp.ListenAndServe(*addr, h); err != nil {
		//if err := fasthttp.ListenAndServeUNIX("/tmp/fasthttp_hyperscan.sock", 666, h); err != nil {
		log.Fatalf("Error in ListenAndServeUNIX: %v", err)
	}

	log.Println("ListenAndServe:", addr)

	hsMatcher.Fini()
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "Hello, world!\n\n")

	fmt.Fprintf(ctx, "Request method is %q\n", ctx.Method())
	fmt.Fprintf(ctx, "RequestURI is %q\n", ctx.RequestURI())
	fmt.Fprintf(ctx, "Requested path is %q\n", ctx.Path())
	fmt.Fprintf(ctx, "Host is %q\n", ctx.Host())
	fmt.Fprintf(ctx, "Query string is %q\n", ctx.QueryArgs())
	fmt.Fprintf(ctx, "User-Agent is %q\n", ctx.UserAgent())
	fmt.Fprintf(ctx, "Connection has been established at %s\n", ctx.ConnTime())
	fmt.Fprintf(ctx, "Request has been started at %s\n", ctx.Time())
	fmt.Fprintf(ctx, "Serial request number for the current connection is %d\n", ctx.ConnRequestNum())
	fmt.Fprintf(ctx, "Your ip is %q\n\n", ctx.RemoteIP())

	fmt.Fprintf(ctx, "Raw request is:\n---CUT---\n%s\n---CUT---", &ctx.Request)

	//Dont use hyperscan match : wrk测试性能： Requests/sec:  77309.85
	/*
		hsctx := HSContext{Data: ctx.RequestURI()}
		hsMatcher.Match(&hsctx)
		if hsctx.Id > 0 {
			ctx.Response.Header.Set("waf-hit-id", strconv.Itoa(int(hsctx.Id)))
			ctx.Response.SetStatusCode(403)
		}
	*/

	ctx.SetContentType("text/plain; charset=utf8")

	// Set arbitrary headers
	ctx.Response.Header.Set("X-My-Header", "my-header-value")

	// Set cookies
	var c fasthttp.Cookie
	c.SetKey("cookie-name")
	c.SetValue("cookie-value")
	ctx.Response.Header.SetCookie(&c)
}

func onMatch(id uint, from, to uint64, flags uint, context interface{}) error {
	hsctx := context.(*HSContext)
	hsctx.Id = id
	hsctx.From = from
	hsctx.To = to

	return nil
}

type HSContext struct {
	Data []byte
	Id   uint
	From uint64
	To   uint64
}

type HSMatcher struct {
	HSDB      hyperscan.BlockDatabase
	HSScratch *hyperscan.Scratch
}

func (self *HSMatcher) Init() (err error) {
	pattern := hyperscan.NewPattern("request_uri", hyperscan.DotAll|hyperscan.SomLeftMost)
	pattern.Id = 10001

	self.HSDB, err = hyperscan.NewBlockDatabase(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Unable to compile pattern \"%s\": %s\n", pattern, err.Error())
		return err
	}

	self.HSScratch, err = hyperscan.NewScratch(self.HSDB)
	if err != nil {
		fmt.Fprint(os.Stderr, "ERROR: Unable to allocate scratch space. Exiting.\n")
		return err
	}

	return nil
}

func (self *HSMatcher) Fini() error {

	self.HSDB.Close()
	self.HSScratch.Free()

	return nil
}

// Test: curl http://localhost:9999/0123456
func (self *HSMatcher) Match(ctx *HSContext) (err error) {
	err = self.HSDB.Scan(ctx.Data, self.HSScratch, onMatch, ctx)
	if err != nil {
		fmt.Println("ERROR: Unable to scan input buffer. Exiting. err:", err.Error())
		fmt.Println("Data::", string(ctx.Data))
		return err
	}
	//fmt.Printf("Scanning %d bytes %s with Hyperscan Id:%d from:%d to:%d hit:[%s]\n", len(hsctx.Data), hsctx.Data, hsctx.Id, hsctx.From, hsctx.To, hsctx.Data[hsctx.From:hsctx.To])

	return nil
}
