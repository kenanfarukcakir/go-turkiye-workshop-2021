package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Trendyol/sidecache/pkg/cache"
	"github.com/Trendyol/sidecache/pkg/model"
	"github.com/klauspost/compress/gzip"
	"github.com/minio/highwayhash"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
)

const CacheHeaderKey = "cachable"
const CacheHeaderEnabledKey = "Sidecache-Headers-Enabled"
const applicationDefaultPort = ":9191"
const DefaultReadBufferSize = 8 * 1024

var (
	gzipValueBytes      = []byte("gzip")
	hashKey             = []byte("000102030405060708090A0B0C0D0E0F")
	fiveMinute          = time.Minute * 5
	lastLoggedTimestamp = time.Now().Add(-fiveMinute)
)

type CacheServer struct {
	Repo           cache.CacheRepository
	Proxy          *fasthttp.HostClient
	Logger         *zap.Logger
	CacheKeyPrefix string
}

func NewServer(repo cache.CacheRepository, proxy *fasthttp.HostClient, logger *zap.Logger) *CacheServer {
	return &CacheServer{
		Repo:           repo,
		Proxy:          proxy,
		Logger:         logger,
		CacheKeyPrefix: os.Getenv("CACHE_KEY_PREFIX"),
	}
}

func (server *CacheServer) Start(stopChan chan os.Signal) {
	promHandler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	handler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/metrics":
			promHandler(ctx)
		case "/purge":
			server.PurgeHandler(ctx)
		default:
			server.CacheHandler(ctx)
		}
	}

	s := fasthttp.Server{
		Handler:        handler,
		ReadBufferSize: DefaultReadBufferSize,
	}
	port := determinatePort()
	server.Logger.Info(fmt.Sprintf("SideCache process started on address: %s", port))

	go func() {
		server.Logger.Warn("Server closed: ", zap.Error(s.ListenAndServe(port)))
	}()

	<-stopChan
	err := s.Shutdown()
	if err != nil {
		server.Logger.Error("shutdown hook error", zap.Error(err))
	}

	server.Logger.Info("http server shut down complete")
}

func (server *CacheServer) cacheResponse(hashedUrl string, headers map[string]string, body []byte) {
	cacheData := model.CacheData{Body: body, Headers: headers}
	cacheDataBytes, _ := cacheData.MarshalJSON()
	server.Repo.SetKey(hashedUrl, cacheDataBytes)
}

func determinatePort() string {
	customPort := "9191"
	if customPort == "" {
		return applicationDefaultPort

	}
	return ":" + customPort
}

func (server CacheServer) gzipWriter(b []byte) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})
	gzipWriter, _ := gzip.NewWriterLevel(buf, gzip.BestSpeed)
	_, err := gzipWriter.Write(b)
	if err != nil {
		server.Logger.Error("Gzip Writer Encountered With an Error", zap.Error(err))
	}
	gzipWriter.Close()
	return buf
}

func (server *CacheServer) CacheHandler(ctx *fasthttp.RequestCtx) {

	defer func() {
		if rec := recover(); rec != nil {
			var err error
			switch x := rec.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}

			server.Logger.Info("Recovered from panic", zap.Error(err))
			ctx.Error(err.Error(), http.StatusInternalServerError)
		}
	}()

	req := &ctx.Request
	resp := &ctx.Response
	hashedURL := server.HashURL(server.ReorderQueryStringFasthttp(req.URI()))

	reqMethod := string(ctx.Method())
	if reqMethod == fasthttp.MethodPost || reqMethod == fasthttp.MethodPut || reqMethod == fasthttp.MethodPatch {
		go server.Repo.Remove(hashedURL)

		if err := server.Proxy.Do(req, resp); err != nil {
			allowed := time.Since(lastLoggedTimestamp) > fiveMinute
			if allowed {
				server.Logger.Error("reverse proxy error occurred", zap.Error(err), zap.ByteString("request url", req.RequestURI()))
				lastLoggedTimestamp = time.Now()
			}
			resp.SetStatusCode(http.StatusBadGateway)
		}
		return
	}

	cachedDataBytes := server.CheckCache(hashedURL)

	requestAcceptEncodingHeaderVal := string(req.Header.Peek("Accept-Encoding"))

	if cachedDataBytes != nil {
		resp.Header.Add("X-Cache-Response-For", string(req.RequestURI()))
		resp.Header.Add("Content-Type", "application/json;charset=UTF-8") //todo get from cache?

		var cachedData model.CacheData
		err := cachedData.UnmarshalJSON(cachedDataBytes)
		if err != nil {
			//backward compatibility
			//if we can not marshall cached data to new structure
			//we write previously cached byte data
			if !strings.Contains(requestAcceptEncodingHeaderVal, "gzip") {
				reader, _ := gzip.NewReader(bytes.NewReader(cachedDataBytes))
				ctx.SetBodyStream(reader, -1)
			} else {
				resp.Header.Add("Content-Encoding", "gzip")
				ctx.SetBody(cachedDataBytes)
			}
		} else {
			if !strings.Contains(requestAcceptEncodingHeaderVal, "gzip") {
				reader, _ := gzip.NewReader(bytes.NewReader(cachedData.Body))
				delete(cachedData.Headers, "Content-Encoding")
				writeHeaders(&resp.Header, cachedData.Headers)
				ctx.SetBodyStream(reader, -1)
			} else {
				writeHeaders(&resp.Header, cachedData.Headers)
				if _, ok := cachedData.Headers["Content-Encoding"]; !ok {
					resp.Header.Add("Content-Encoding", "gzip")
				}
				ctx.SetBody(cachedData.Body)
			}
		}
	} else {
		server.ReverseProxyHandler(req, resp, hashedURL)
	}
}

func (server *CacheServer) ReverseProxyHandler(req *fasthttp.Request, resp *fasthttp.Response, hashedURL string) {
	if err := server.Proxy.Do(req, resp); err != nil {
		allowed := time.Since(lastLoggedTimestamp) > fiveMinute
		if allowed {
			server.Logger.Error("reverse proxy error occurred", zap.Error(err), zap.ByteString("request url", req.RequestURI()))
			lastLoggedTimestamp = time.Now()
		}
		resp.SetStatusCode(http.StatusBadGateway)
		return
	}

	cacheHeaderValue := resp.Header.Peek(CacheHeaderKey)
	shouldCache := len(cacheHeaderValue) > 0 && !is5xxStatusCode(resp.StatusCode())

	if shouldCache {
		var (
			gzippedRespBody []byte
			respBody        = resp.Body()
			responseGzipped = bytes.Equal(resp.Header.Peek("Content-Encoding"), gzipValueBytes)
		)

		if responseGzipped {
			gzippedRespBody = respBody
		} else {
			gzippedRespBody = server.gzipWriter(respBody).Bytes()
		}

		resp.Header.Del("Content-Length")

		cacheHeadersEnabled := string(resp.Header.Peek(CacheHeaderEnabledKey)) == "true"

		var headers map[string]string
		if cacheHeadersEnabled {
			headers = make(map[string]string)
			resp.Header.VisitAll(func(k, v []byte) {
				key := string(k)
				if len(headers[key]) > 0 {
					headers[key] += fmt.Sprintf(";%s", v)
				} else {
					headers[key] = string(v)
				}
			})
		}

		go server.cacheResponse(hashedURL, headers, gzippedRespBody)
	}
}

func (server *CacheServer) PurgeHandler(ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	resp := &ctx.Response
	if string(req.Header.Method()) != http.MethodPost {
		resp.SetStatusCode(http.StatusMethodNotAllowed)
		return
	}

	defer func() {
		if rec := recover(); rec != nil {
			var err error
			switch x := rec.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}

			server.Logger.Info("Recovered from panic", zap.Error(err))
			resp.SetStatusCode(http.StatusInternalServerError)
			resp.SetBodyString(err.Error())
		}
	}()

	purgeRequest := model.PurgeRequest{}
	err := json.Unmarshal(req.Body(), &purgeRequest)
	if err != nil {
		resp.SetStatusCode(http.StatusBadRequest)
		resp.SetBodyString("could not parse the request body")
		return
	}

	purgeRequest.EnsureHasSlashPrefix()

	purgeUrl, err := url.Parse(purgeRequest.Url)
	if err != nil {
		server.Logger.Info("Failed to parse purge url: ", zap.String("url", purgeRequest.Url), zap.Error(err))
		resp.SetStatusCode(http.StatusBadRequest)
		resp.SetBodyString(fmt.Sprintf("failed to parse url: %s", purgeRequest.Url))
		return
	}

	hashedURL := server.HashURL(server.ReorderQueryString(purgeUrl))

	err = server.Repo.Remove(hashedURL)
	if err != nil {
		resp.SetStatusCode(http.StatusInternalServerError)
		resp.SetBodyString("error occurred while removing the cache")
		return
	}

}

func writeHeaders(header *fasthttp.ResponseHeader, headers map[string]string) {
	if headers != nil {
		for h, v := range headers {
			header.Set(h, v)
		}
	}
}

func (server CacheServer) GetHeaderTTL(cacheHeaderValue string) int {
	cacheValues := strings.Split(cacheHeaderValue, "=")
	var maxAgeInSecond = 0
	if len(cacheValues) > 1 {
		maxAgeInSecond, _ = strconv.Atoi(cacheValues[1])
	}
	return maxAgeInSecond
}

func (server CacheServer) HashURL(url string) string {
	keyToHash := []byte(server.CacheKeyPrefix + "/" + url)
	sum := highwayhash.Sum(keyToHash, hashKey)
	return string(sum[:])
}

func (server CacheServer) CheckCache(url string) []byte {
	if server.Repo == nil {
		return nil
	}
	return server.Repo.Get(url)
}

func (server CacheServer) ReorderQueryString(url *url.URL) string {
	return url.Path + "?" + url.Query().Encode()
}

func (server CacheServer) ReorderQueryStringFasthttp(uri *fasthttp.URI) string {
	args := uri.QueryArgs()
	var buf strings.Builder
	keys := make([]string, 0, args.Len())
	kvs := map[string][]string{}
	args.VisitAll(func(key, value []byte) {
		sKey := string(key)
		kvs[sKey] = append(kvs[sKey], string(value))
	})

	for k := range kvs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vs := kvs[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}

	sortedQueryString := buf.String()
	return string(uri.Path()) + "?" + sortedQueryString
}

func is5xxStatusCode(statusCode int) bool {
	return statusCode >= 500 && statusCode < 600
}
