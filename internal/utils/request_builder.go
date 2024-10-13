package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RequestBuilder is the main interface with some methods to build and handle a request.
type RequestBuilder interface {
	SetMethodGet() RequestBuilder
	SetMethodPut() RequestBuilder
	SetMethodPost() RequestBuilder
	SetMethodPatch() RequestBuilder
	SetMethodDelete() RequestBuilder
	SetMethodOptions() RequestBuilder
	SetMethod(method string) RequestBuilder
	SetBaseURL(baseURL string) RequestBuilder
	SetPath(path string) RequestBuilder
	SetBody(body interface{}) RequestBuilder
	AddHeader(key, value string) RequestBuilder
	AddQueryParameter(key, value string) RequestBuilder
	SetBasicAuthentication(username, password string) RequestBuilder
	Build() (*http.Request, error)
}

// requestBuilderImpl is the implementation of RequestBuilder.
type requestBuilderImpl struct {
	headers         map[string]string
	queryParameters map[string]string
	body            interface{}
	baseURL         string
	path            string
	fullPath        string
	httpMethod      string
}

// NewRequestBuilder is the constructor of an empty RequestBuilder.
func NewRequestBuilder() RequestBuilder {
	return &requestBuilderImpl{
		headers:         make(map[string]string),
		queryParameters: make(map[string]string),
	}
}

//** BaseURL, Path implementations **//

// SetBaseURL sets the base URL of the request.
func (r requestBuilderImpl) SetBaseURL(baseURL string) RequestBuilder {
	r.baseURL = baseURL
	return &r
}

// SetPath sets the path of the request.
func (r requestBuilderImpl) SetPath(path string) RequestBuilder {
	r.path = path
	return &r
}

// SetFullPath sets the path of the request to be the full path.
func (r requestBuilderImpl) SetFullPath(fullPath string) RequestBuilder {
	r.fullPath = fullPath
	return &r
}

//** Method implementations **//

// SetMethodGet sets the HTTP method to GET.
func (r requestBuilderImpl) SetMethodGet() RequestBuilder {
	r.httpMethod = http.MethodGet
	return &r
}

// SetMethodPut sets the HTTP method to PUT.
func (r requestBuilderImpl) SetMethodPut() RequestBuilder {
	r.httpMethod = http.MethodPut
	return &r
}

// SetMethodPost sets the HTTP method to POST.
func (r requestBuilderImpl) SetMethodPost() RequestBuilder {
	r.httpMethod = http.MethodPost
	return &r
}

// SetMethodPatch sets the HTTP method to PATCH.
func (r requestBuilderImpl) SetMethodPatch() RequestBuilder {
	r.httpMethod = http.MethodPatch
	return &r
}

// SetMethodDelete sets the HTTP method to DELETE.
func (r requestBuilderImpl) SetMethodDelete() RequestBuilder {
	r.httpMethod = http.MethodDelete
	return &r
}

// SetMethodOptions sets the HTTP method to OPTIONS.
func (r requestBuilderImpl) SetMethodOptions() RequestBuilder {
	r.httpMethod = http.MethodOptions
	return &r
}

// SetMethod sets the HTTP method with a custom value (defaults to GET).
func (r requestBuilderImpl) SetMethod(method string) RequestBuilder {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		r.httpMethod = method
	default:
		r.httpMethod = http.MethodGet
	}
	return &r
}

//** Body, Header, QueryParameter implementations **//

// SetBody sets the body of the request.
func (r requestBuilderImpl) SetBody(body interface{}) RequestBuilder {
	r.body = body
	return &r
}

// AddHeader adds a header to the request.
func (r requestBuilderImpl) AddHeader(key, value string) RequestBuilder {
	r.headers[key] = value
	return &r
}

// AddQueryParameter adds a query parameter to the request.
func (r requestBuilderImpl) AddQueryParameter(key, value string) RequestBuilder {
	r.queryParameters[key] = value
	return &r
}

// ** Basic Authentication implementation **//

func (r requestBuilderImpl) SetBasicAuthentication(username, password string) RequestBuilder {
	rawCredentials := fmt.Sprintf("%s:%s", username, password)
	credentials := base64.StdEncoding.EncodeToString([]byte(rawCredentials))
	r.headers["Authorization"] = fmt.Sprintf("Basic %s", credentials)
	return &r
}

// Build creates the request with the given parameters.
func (r requestBuilderImpl) Build() (*http.Request, error) {
	if err := validateEmptyStringField(r.httpMethod, "HTTP method"); err != nil {
		return nil, err
	}
	if err := validateEmptyStringField(r.baseURL, "base URL"); err != nil {
		return nil, err
	}

	reader, errReader := parseBodyJSONToReader(r.body)
	if errReader != nil {
		return nil, errReader
	}

	request, errRequest := http.NewRequest(r.httpMethod, r.baseURL+r.path, reader)
	if errRequest != nil {
		return nil, errRequest
	}

	setHeaders(request, r.headers)
	setQueryParameters(request, r.queryParameters)

	return request, nil
}

// validateEmptyStringField validates if a field is empty.
func validateEmptyStringField(fieldValue, errMessage string) error {
	if fieldValue == "" {
		return errors.New(errMessage + " is not defined")
	}

	return nil
}

// parseBodyJSONToReader parses the body to a reader.
func parseBodyJSONToReader(body interface{}) (io.Reader, error) {
	var reader io.Reader
	if body == nil {
		return reader, nil
	}

	jsonBody, err := json.MarshalIndent(body, "", " ")
	if err != nil {
		return nil, err
	}
	bytesReader := bytes.NewReader(jsonBody)

	return bytesReader, nil
}

// setHeaders sets the headers of the request.
func setHeaders(request *http.Request, headers map[string]string) {
	if len(headers) != 0 {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}
}

// setQueryParameters sets the query parameters of the request.
func setQueryParameters(request *http.Request, queryParameters map[string]string) {
	if len(queryParameters) != 0 {
		query := request.URL.Query()
		for key, value := range queryParameters {
			query.Set(key, value)
		}
		request.URL.RawQuery = query.Encode()
	}
}
