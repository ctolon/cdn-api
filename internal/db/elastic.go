package db

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/rs/zerolog"
)

type CreateIndexError struct {
	msg string
}

func (e *CreateIndexError) Error() string {
	return e.msg
}

type (

	// Config is the configuration for ElasticSearch client
	Config struct {
		Hosts                 []string
		DefaultIndex          string
		DisableCompression    bool
		MaxSearchQueryTimeout time.Duration
		PathToMappingSchema   string
		IsTrackTotalHits      bool
	}

	// EsClient is the Customized ElasticSearch client
	EsClient struct {
		Log                   *EsLogger
		esCfg                 elasticsearch.Config
		Client                *elasticsearch.Client
		defaultIndex          string
		maxSearchQueryTimeout time.Duration
		isTrackTotalHits      bool
	}

	// EsLogger is the custom logger for ElasticSearch
	EsLogger struct {
		zerolog.Logger
	}

	// MappingFrom is the interface for mapping
	MappingFrom interface {
		[]byte | string
	}
)

// LogRoundTrip prints the information about request and response.
func (l *EsLogger) LogRoundTrip(
	req *http.Request,
	res *http.Response,
	err error,
	start time.Time,
	dur time.Duration,
) error {
	var (
		e    *zerolog.Event
		nReq int64
		nRes int64
	)

	// Set error level.
	//
	switch {
	case err != nil:
		e = l.Error()
	case res != nil && res.StatusCode > 0 && res.StatusCode < 300:
		e = l.Info()
	case res != nil && res.StatusCode > 299 && res.StatusCode < 500:
		e = l.Warn()
	case res != nil && res.StatusCode > 499:
		e = l.Error()
	default:
		e = l.Error()
	}

	// Count number of bytes in request and response.
	//
	if req != nil && req.Body != nil && req.Body != http.NoBody {
		nReq, _ = io.Copy(io.Discard, req.Body)
	}
	if res != nil && res.Body != nil && res.Body != http.NoBody {
		nRes, _ = io.Copy(io.Discard, res.Body)
	}

	// Log event.
	//
	e.Str("method", req.Method).
		Int("status_code", res.StatusCode).
		Dur("duration", dur).
		Int64("req_bytes", nReq).
		Int64("res_bytes", nRes).
		Msg(req.URL.String())

	return nil
}

// RequestBodyEnabled makes the client pass request body to logger
func (l *EsLogger) RequestBodyEnabled() bool { return true }

// RequestBodyEnabled makes the client pass response body to logger
func (l *EsLogger) ResponseBodyEnabled() bool { return true }

// NewEsLogger create new ElasticSearch logger
func NewEsLogger(logger zerolog.Logger) *EsLogger {
	return &EsLogger{
		Logger: logger,
	}
}

// InitESClient initiliazr ElasticSearch client
func InitESClient(cfg elasticsearch.Config, logger zerolog.Logger) (*elasticsearch.Client, error) {

	lg := NewEsLogger(logger)

	es, err := elasticsearch.NewClient(cfg)

	if err != nil {
		lg.Logger.Err(err).Msg("Error creating the Elasticsearch client")
		return nil, err
	}

	_, resErr := es.API.Ping()

	if resErr != nil {
		lg.Logger.Err(err).Msg("Error pinging the Elasticsearch server")
		return nil, resErr
		//panic(err.Error())
	}

	return es, nil

}

// NewEsClient create new ElasticSearch client
func NewEsClient(logger zerolog.Logger, esCfg elasticsearch.Config, customCfg Config) (*EsClient, error) {

	log := NewEsLogger(logger)

	es, err := InitESClient(esCfg, logger)
	if es == nil {
		log.Logger.Error().Msg("Could not create new ElasticSearch client due error")
		return nil, err
	}

	//log.Logger.Info().Msgf("Successfully connected to ElasticSearch cluster. Hosts: %v", customCfg.Hosts)
	//log.Logger.Info().Str("Try to create defaultIndex (if not exist)").Msgf("DefaultIndex: %s", customCfg.DefaultIndex)

	c := &EsClient{
		Log:                   log,
		esCfg:                 esCfg,
		Client:                es,
		defaultIndex:          customCfg.DefaultIndex,
		maxSearchQueryTimeout: customCfg.MaxSearchQueryTimeout,
		isTrackTotalHits:      true,
	}

	return c, nil
}

// TODO There can be combined into one function
// CreateIndexFromFile create index w/mapping in elasticsearch from json
func (e *EsClient) CreateMappingFromFile(ctx context.Context, index string, mapping string) error {
	var file []byte

	if _, err := os.Stat(mapping); os.IsNotExist(err) {
		e.Log.Logger.Error().Err(err).Str("path_to_mapping_schema", mapping).Msg("File does not exist")
		return err
	}

	file, err := os.ReadFile(mapping)
	if err != nil || file == nil {
		e.Log.Logger.Error().Err(err).Str("path_to_mapping_schema", mapping).Msg("err reading mapping file")
		return err

	}
	indexMappingSchema := string(file)

	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  strings.NewReader(indexMappingSchema),
	}

	res, err := req.Do(ctx, e.Client)
	if err != nil {
		e.Log.Logger.Error().Err(err).Msg("err creating index")
		return err
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			e.Log.Logger.Error().Err(err).Msg("res.Body.Close() problem")
		}
	}()

	if res.IsError() {
		e.Log.Logger.Error().Msgf("err creating defaultIndex. res: %s", res.String())
		customErr := CreateIndexError{msg: res.String()}
		return &customErr

	}

	e.Log.Logger.Info().Msgf("Successfully created index: %s", index)
	e.Log.Logger.Info().Msgf("Response: %s", res.String())

	return nil
}

// CreateIndexFromString create index w/mapping in elasticsearch from string
func (e *EsClient) CreateMappingFromString(ctx context.Context, index string, mapping string) (*esapi.Response, error) {

	res, err := e.Client.Indices.Create(
		index,
		e.Client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)

	if err != nil {
		e.Log.Logger.Error().Err(err).Str("mapping", mapping).Msgf("err creating index: %s", index)
		return nil, err
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			e.Log.Logger.Error().Err(err).Msg("res.Body.Close() problem!")
		}
	}()

	return res, nil
}

// CreateIndexFromBytes create index w/mapping in elasticsearch from bytes
func (e *EsClient) CreateMappingFromBytes(ctx context.Context, index string, mapping []byte) (*esapi.Response, error) {

	e.Log.Logger.Info().Msgf("Trying Creating index: %s", index)

	// Check if the index already exists
	//
	resp, _ := e.Client.Indices.Get([]string{index})

	if resp.IsError() && resp.StatusCode == 404 {
		e.Log.Logger.Info().Msgf("Index: %s does not exist, it will be created.", index)
	} else if resp.StatusCode == 200 {
		e.Log.Logger.Info().Msgf("Index: %s already exists, it will not be created.", index)
		return nil, nil
	}

	_mapping := strings.NewReader(string(mapping))
	res, err := e.Client.Indices.Create(
		index,
		e.Client.Indices.Create.WithBody(_mapping),
	)

	if err != nil {
		//e.Log.Logger.Error().Err(err).Str("mapping", _mapping).Msgf("err creating index: %s", index)
		return nil, err
	}

	resMap := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&resMap); err != nil {
		e.Log.Logger.Error().Err(err).Msg("Error decoding response")
		return nil, err
	}

	if res.Status() != "200 OK" {
		e.Log.Logger.Error().Msgf("Error creating index: %s", index)
		e.Log.Logger.Error().Msgf("Response: %s", res.String())
		e.Log.Logger.Error().Msgf("Status Code: %s", res.Status())
		return nil, err
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			e.Log.Logger.Error().Err(err).Msg("res.Body.Close() problem!")
		}
	}()

	e.Log.Logger.Info().Msgf("Successfully created index: %s", index)

	return res, nil
}

// NewDefaultEsConfig create default ElasticSearch configuration
func NewDefaultEsConfig(dns string, logger elastictransport.Logger) *elasticsearch.Config {

	var t http.RoundTripper = &http.Transport{
		// Proxy:                 http.ProxyFromEnvironment,
		// ForceAttemptHTTP2:     false,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		// DisableCompression: true,
	}

	cfg := &elasticsearch.Config{
		Addresses: []string{dns},
		//Username:              esUsername,
		//Password:              esPassword,
		//CloudID:               "",
		//APIKey:                "",
		Header:        nil,
		CACert:        nil,
		RetryOnStatus: nil,
		DisableRetry:  false,
		//RetryOnError:  		nil,
		MaxRetries:            3,
		DiscoverNodesOnStart:  false,
		DiscoverNodesInterval: 0,
		EnableMetrics:         true,
		EnableDebugLogger:     true,
		RetryBackoff:          nil,
		Transport:             t,
		Logger:                logger,
		Selector:              nil,
		ConnectionPoolFunc:    nil,
	}

	return cfg
}
