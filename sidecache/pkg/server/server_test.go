package server_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"github.com/Trendyol/sidecache/pkg/cache"
	"github.com/Trendyol/sidecache/pkg/metric"
	"github.com/Trendyol/sidecache/pkg/model"
	"github.com/Trendyol/sidecache/pkg/server"
	"github.com/golang/mock/gomock"
	"github.com/minio/highwayhash"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	cacheServerForCacheableApi    *server.CacheServer
	cacheServerForNonCacheableApi *server.CacheServer
	httpClient                    = http.Client{Transport: &http.Transport{DisableCompression: true}} // default transport adds Accept-Encoding=gzip
	nonCacheableApiResponseBody   = map[string]string{
		"Id":    "0",
		"Name":  "makina",
		"Email": "makina@trendyol.com",
	}
	cacheableApiResponseBody = map[string]string{
		"Id":    "1",
		"Name":  "Emre Savcı",
		"Email": "emre.savci@trendyol.com",
		"Phone": "000099999",
	}
	cacheableApiResponseHeaders = map[string]string{
		"Content-Type":              "application/json",
		"Cache-Ttl":                 "300",
		"Tysidecarcachable":         "ttl=300",
		"Sidecache-Headers-Enabled": "true",
		"X-Custom-Header":           "abc,xyz",
	}
)

func writeGzip(a *[]byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(*a); err != nil {
		gz.Close()
		panic(err)
	}
	gz.Close()
	return b.Bytes()
}

func TestMain(m *testing.M) {
	stopChan := make(chan os.Signal)
	logger, _ := zap.NewProduction()
	prometheusClient := metric.NewPrometheusClient()
	prometheusClient.CacheHitCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "cache_hit_counter",
			Help:      "This is my counter",
		})

	cacheableApiUrl, _ := url.Parse("localhost:8080")
	cacheableProxy := &fasthttp.HostClient{
		Addr: cacheableApiUrl.String(),
	}
	cacheServerForCacheableApi = server.NewServer(nil, cacheableProxy, prometheusClient, logger)
	cacheableServerListener, _ := net.Listen("tcp", "127.0.0.1:8080")
	cacheableHttpServer := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for k, v := range cacheableApiResponseHeaders {
				w.Header().Set(k, v)
			}
			if strings.Contains(r.URL.String(), "broken-endpoint") {
				w.WriteHeader(500)
				w.Write([]byte("I'm an error response, don't cache me."))
				return
			}
			if strings.Contains(r.Header.Get("accept-encoding"), "gzip") {
				bs, _ := json.Marshal(cacheableApiResponseBody)
				w.Header().Set("content-encoding", "gzip")
				w.Write(writeGzip(&bs))
			} else {
				json.NewEncoder(w).Encode(cacheableApiResponseBody)
			}
		}))
	go cacheServerForCacheableApi.Start(stopChan)

	time.Sleep(3 * time.Second)

	os.Setenv("SIDE_CACHE_PORT", "9292")
	nonCacheableApiUrl, _ := url.Parse("localhost:8081")
	nonCacheableProxy := &fasthttp.HostClient{
		Addr: nonCacheableApiUrl.String(),
	}
	cacheServerForNonCacheableApi = server.NewServer(nil, nonCacheableProxy, prometheusClient, logger)
	nonCacheableServerListener, _ := net.Listen("tcp", "127.0.0.1:8081")
	nonCacheableHttpServer := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.Header.Get("accept-encoding"), "gzip") {
				bs, _ := json.Marshal(cacheableApiResponseBody)
				w.Header().Set("content-encoding", "gzip")
				w.Write(writeGzip(&bs))
			} else {
				json.NewEncoder(w).Encode(nonCacheableApiResponseBody)
			}
		}))
	go cacheServerForNonCacheableApi.Start(stopChan)

	time.Sleep(5 * time.Second)

	cacheableHttpServer.Listener.Close()
	cacheableHttpServer.Listener = cacheableServerListener
	cacheableHttpServer.Start()

	nonCacheableHttpServer.Listener.Close()
	nonCacheableHttpServer.Listener = nonCacheableServerListener
	nonCacheableHttpServer.Start()

	code := m.Run()

	cacheableHttpServer.Close()
	nonCacheableHttpServer.Close()
	stopChan <- os.Interrupt
	os.Exit(code)
}

func TestReorderQueryString(t *testing.T) {
	var firstURL *url.URL
	var reorderQueryString string
	firstURL, _ = url.Parse("http://localhost:8080/api?year=2020&name=emre&name=makina")

	reorderQueryString = cacheServerForCacheableApi.ReorderQueryString(firstURL)

	assert.Equal(t, "/api?name=emre&name=makina&year=2020", reorderQueryString)
}

func TestReorderQueryStringFasthttp(t *testing.T) {
	firstURL := fasthttp.URI{}
	var reorderQueryString string
	err := firstURL.Parse([]byte("http://localhost:8080"), []byte("/api?year=2020&name=emre&name=makina"))
	assert.Nil(t, err)

	reorderQueryString = cacheServerForCacheableApi.ReorderQueryStringFasthttp(&firstURL)

	assert.Equal(t, "/api?name=emre&name=makina&year=2020", reorderQueryString)
}

func TestHashUrl(t *testing.T) {
	cacheServerForCacheableApi.CacheKeyPrefix = "test-prefix"

	testUrl := "testurl"

	keyToHash := []byte("test-prefix" + "/" + testUrl)
	sum := highwayhash.Sum(keyToHash, []byte("000102030405060708090A0B0C0D0E0F"))
	expectedHash := string(sum[:])

	actualHash := cacheServerForCacheableApi.HashURL(testUrl)

	assert.Equal(t, expectedHash, actualHash)
}

func TestGetTTL(t *testing.T) {
	var value int
	value = cacheServerForCacheableApi.GetHeaderTTL("max-age=100")

	assert.Equal(t, 100, value)
}

func TestReturnProxyResponseWhenRepoReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	repo.
		EXPECT().
		Get(gomock.Any()).
		Return(nil)

	repo.
		EXPECT().
		SetKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1)

	resp, err := httpClient.Get("http://localhost:9191/api?name=emre&year=2020")
	if err != nil {
		t.Errorf("Error occurred while acquiring response err: %+v", err)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)

	time.Sleep(3 * time.Second)
	actual := make(map[string]string)
	json.Unmarshal(respBody, &actual)

	assert.EqualValues(t, cacheableApiResponseBody, actual)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	assert.Empty(t, resp.Header.Get("X-Cache-Response-For"))
}

func TestReturnCacheResponseWhenRepoReturnsData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	body := []byte(`{"key":"value"}`)
	buf := bytes.NewBuffer([]byte{})
	gzipWriter := gzip.NewWriter(buf)
	gzipWriter.Write(body)
	gzipWriter.Close()

	cacheData := model.CacheData{
		Body:    buf.Bytes(),
		Headers: map[string]string{"key1": "value1"},
	}

	cacheDataBytes, _ := json.Marshal(cacheData)

	repo.
		EXPECT().
		Get(gomock.Any()).
		Return(cacheDataBytes)

	resp, _ := httpClient.Get("http://localhost:9191/api?name=emre&year=2020")
	respBody, _ := ioutil.ReadAll(resp.Body)

	assert.EqualValues(t, body, respBody)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	assert.Equal(t, "value1", resp.Header.Get("key1"))
	assert.Equal(t, "/api?name=emre&year=2020", resp.Header.Get("X-Cache-Response-For"))
}

func TestReturnCompressedProxyResponseWhenServerReturnsGzip(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	repo.EXPECT().
		Get(gomock.Any()).
		Return(nil)

	repo.EXPECT().
		SetKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:9191/api?name=emre&year=2020", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, _ := httpClient.Do(req)

	time.Sleep(3 * time.Second)
	gzipReader, _ := gzip.NewReader(resp.Body)
	respBody, _ := ioutil.ReadAll(gzipReader)
	actualBody := map[string]string{}
	json.Unmarshal(respBody, &actualBody)

	assert.EqualValues(t, cacheableApiResponseBody, actualBody)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	assert.Empty(t, resp.Header.Get("X-Cache-Response-For"))
}

func TestReturnCompressedCacheResponseWhenClientAcceptsGzip(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	body := []byte(`{"key":"value"}`)
	buf := bytes.NewBuffer([]byte{})
	gzipWriter := gzip.NewWriter(buf)
	gzipWriter.Write(body)
	gzipWriter.Close()

	cacheData := model.CacheData{
		Body:    buf.Bytes(),
		Headers: map[string]string{"key1": "value1"},
	}

	cacheDataBytes, _ := json.Marshal(cacheData)

	repo.
		EXPECT().
		Get(gomock.Any()).
		Return(cacheDataBytes)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:9191/api?name=emre&year=2020", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, _ := httpClient.Do(req)

	gzipReader, _ := gzip.NewReader(resp.Body)
	respBody, _ := ioutil.ReadAll(gzipReader)

	assert.EqualValues(t, body, respBody)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	assert.Equal(t, "value1", resp.Header.Get("key1"))
	assert.Equal(t, "/api?name=emre&year=2020", resp.Header.Get("X-Cache-Response-For"))
}

func TestReturnCacheHeadersWhenCacheHeaderEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	repo.
		EXPECT().
		Get(gomock.Any()).
		Return(nil)

	repo.
		EXPECT().
		SetKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(key string, value []byte, ttl int) {
			cacheData := model.CacheData{}
			json.Unmarshal(value, &cacheData)

			reader, _ := gzip.NewReader(bytes.NewReader(cacheData.Body))
			cacheBodyBytes, _ := ioutil.ReadAll(reader)
			reader.Close()

			cacheBody := map[string]string{}
			json.Unmarshal(cacheBodyBytes, &cacheBody)

			assert.EqualValues(t, cacheableApiResponseBody, cacheBody)
			for key, value := range cacheableApiResponseHeaders {
				assert.Equal(t, value, cacheData.Headers[key])
			}
		})

	req, _ := http.NewRequest("GET", "http://localhost:9191/api?name=emre&year=2020", nil)
	resp, _ := httpClient.Do(req)

	time.Sleep(5 * time.Second)
	respBody, _ := ioutil.ReadAll(resp.Body)

	actual := make(map[string]string)
	json.Unmarshal(respBody, &actual)

	assert.EqualValues(t, cacheableApiResponseBody, actual)
}

func TestReturnProxyResponseWhenRepoReturnsNilForNonCacheableApi(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForNonCacheableApi.Repo = repo

	repo.
		EXPECT().
		Get(gomock.Any()).
		Return(nil)

	resp, err := httpClient.Get("http://localhost:9292/api?name=emre&year=2020")
	if err != nil {
		t.Errorf("Error occurred while acquiring response err: %+v", err)
	}
	respBody, _ := ioutil.ReadAll(resp.Body)

	time.Sleep(3 * time.Second)
	actual := make(map[string]string)
	json.Unmarshal(respBody, &actual)

	assert.EqualValues(t, nonCacheableApiResponseBody, actual)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	assert.Empty(t, resp.Header.Get("X-Cache-Response-For"))
}

func TestShouldNotCacheIfResponseStatusCodeIs5xx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	repo.
		EXPECT().
		Get(gomock.Any()).
		Return(nil)

	repo.
		EXPECT().
		SetKey(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	resp, err := httpClient.Get("http://localhost:9191/broken-endpoint")
	if err != nil {
		t.Errorf("Error occurred while acquiring response err: %+v", err)
	}

	time.Sleep(3 * time.Second)
	assert.Equal(t, 500, resp.StatusCode)
}

func TestPurge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	expectedPurgeUrl, _ := url.Parse("/users?id=1")

	repo.
		EXPECT().
		Remove(cacheServerForCacheableApi.HashURL(cacheServerForCacheableApi.ReorderQueryString(expectedPurgeUrl))).
		Return(nil).
		Times(1)

	httpClient := &http.Client{}
	requestBody := []byte(`{"url":"/users?id=1"}`)

	req, _ := http.NewRequest("POST", "http://localhost:9191/purge", bytes.NewBuffer(requestBody))
	resp, _ := httpClient.Do(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPurgeWhenMethodIsNotPost(t *testing.T) {
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:9191/purge", nil)
	resp, _ := httpClient.Do(req)

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestPurgeWhenInvalidRequestBody(t *testing.T) {
	httpClient := &http.Client{}
	requestBody := []byte(`*`)

	req, _ := http.NewRequest("POST", "http://localhost:9191/purge", bytes.NewBuffer(requestBody))
	resp, _ := httpClient.Do(req)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPurgeWhenInvalidUrl(t *testing.T) {
	httpClient := &http.Client{}
	requestBody := []byte(`{"url":")(/&%"}`)

	req, _ := http.NewRequest("POST", "http://localhost:9191/purge", bytes.NewBuffer(requestBody))
	resp, _ := httpClient.Do(req)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPurgeWhenErrorFromRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := cache.NewMockCacheRepository(ctrl)
	cacheServerForCacheableApi.Repo = repo

	expectedPurgeUrl, _ := url.Parse("/users?id=1")

	repo.
		EXPECT().
		Remove(cacheServerForCacheableApi.HashURL(cacheServerForCacheableApi.ReorderQueryString(expectedPurgeUrl))).
		Return(errors.New("error-text")).
		Times(1)

	httpClient := &http.Client{}
	requestBody := []byte(`{"url":"/users?id=1"}`)

	req, _ := http.NewRequest("POST", "http://localhost:9191/purge", bytes.NewBuffer(requestBody))
	resp, _ := httpClient.Do(req)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRemoveDataWhenRequestMethodPostPutPatch(t *testing.T) {
	tests := []struct {
		Name string
		ReqMethod string
	}{
		{
			Name: "TestRemoveDataWhenRequestMethodPost",
			ReqMethod: "POST",
		},
		{
			Name: "TestRemoveDataWhenRequestMethodPut",
			ReqMethod: "PUT",
		},
		{
			Name: "TestRemoveDataWhenRequestMethodPatch",
			ReqMethod: "PATCH",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := cache.NewMockCacheRepository(ctrl)
			cacheServerForCacheableApi.Repo = repo

			expectedPurgeUrl, _ := url.Parse("/users")

			repo.
				EXPECT().
				Remove(cacheServerForCacheableApi.HashURL(cacheServerForCacheableApi.ReorderQueryString(expectedPurgeUrl))).
				Return(nil).
				Times(1)

			httpClient := &http.Client{}
			requestBody := []byte(`{"url":"/users"}`)

			req, _ := http.NewRequest(test.ReqMethod, "http://localhost:9191/users", bytes.NewBuffer(requestBody))
			resp, _ := httpClient.Do(req)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}
