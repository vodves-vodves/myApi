package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func handleRequests() {
	r := router.New()
	r.GET("/test", testQuery)
	r.GET("/zap", returnZapperRequest)

	log.Fatal(fasthttp.ListenAndServe(":10000", r.Handler))
}

// Эндпоинт
func returnZapperRequest(ctx *fasthttp.RequestCtx) {
	t1 := time.Now()
	wallet := string(ctx.QueryArgs().Peek("wallet"))
	proxy := string(ctx.QueryArgs().Peek("proxy"))
	if wallet == "" {
		ctx.Error("Miss wallet", 400)
		log.Printf("[%s] %s\n", ctx.Method(), "Miss wallet")
		return
	} else if proxy == "" {
		ctx.Error("Miss proxy", 400)
		log.Printf("[%s] %s\n", ctx.Method(), "Miss proxy")
		return
	}
	res, err := zapperRequest(wallet, proxy)
	if err != nil {
		res, err = zapperRequest(wallet, proxy)
		if err != nil {
			ctx.Error(err.Error(), 400)
			return
		}
	}
	ctx.Write(res)
	t2 := time.Now()
	log.Printf("[%s] %v\n", ctx.Method(), t2.Sub(t1))
}

func testQuery(ctx *fasthttp.RequestCtx) {
	wallet := string(ctx.QueryArgs().Peek("wallet"))
	proxy := string(ctx.QueryArgs().Peek("proxy"))

	if wallet == "" {
		wallet = "nil"
	}
	if proxy == "" {
		proxy = "nil"
	}
	fmt.Fprintf(ctx, "Your wallet %s, your proxy %s", wallet, proxy)
}

func notFound(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Page not found")
	log.Printf("[%s] %s\n", ctx.Method(), "Called not found page")
}

func main() {
	num := runtime.NumCPU()
	runtime.GOMAXPROCS(num)
	fmt.Println("Server started!")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		handleRequests()
		defer wg.Done()
	}()
	wg.Wait()
}

// Получение ответа
func zapperRequest(wallet string, proxy string) ([]byte, error) {
	url := "https://web.zapper.fi/v2/balances?addresses%5B0%5D=" + wallet + "&networks%5B0%5D=ethereum&networks%5B1%5D=polygon&networks%5B2%5D=optimism&networks%5B3%5D=gnosis&networks%5B4%5D=binance-smart-chain&networks%5B5%5D=fantom&networks%5B6%5D=avalanche&networks%5B7%5D=arbitrum&networks%5B8%5D=celo&networks%5B9%5D=harmony&networks%5B10%5D=moonriver&networks%5B11%5D=bitcoin&networks%5B12%5D=cronos&networks%5B13%5D=aurora&networks%5B14%5D=evmos&nonNilOnly=true&useNewBalancesFormat=true&useNftService=true"
	query := make(map[string]string)
	json, err := sendRequest(url, query, proxy)
	if err != nil {
		log.Println("err: ", err)
		return nil, err
	}
	return json, nil
}

// Отправка запроса
func sendRequest(urlLink string, query map[string]string, proxy string) ([]byte, error) {
	headers := map[string]string{
		"Accept":     "*/*",
		"User-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36",
	}
	proxySplit := strings.Split(proxy, ":")
	proxyURL, _ := url.Parse(
		fmt.Sprintf(
			"http://%s:%s@%s:%s",
			url.QueryEscape(proxySplit[2]), url.QueryEscape(proxySplit[3]),
			url.QueryEscape(proxySplit[0]), url.QueryEscape(proxySplit[1]),
		),
	)
	client := &http.Client{
		Timeout:   time.Second * 5,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
	client.Transport = cloudflarebp.AddCloudFlareByPass(client.Transport)
	req, _ := http.NewRequest("GET", urlLink, nil)
	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}
