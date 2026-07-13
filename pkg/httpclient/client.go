package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client é um cliente HTTP reutilizável para integrações externas.
// Configure com New() e use Get() para chamadas com decode JSON automático.
type Client struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// Option configura um Client.
type Option func(*Client)

// WithTimeout define o timeout do cliente (padrão: 30s).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithHeader adiciona um header padrão enviado em todas as requisições.
func WithHeader(key, value string) Option {
	return func(c *Client) { c.headers[key] = value }
}

// New cria um Client para baseURL.
// Padrão: 30 s de timeout, sem headers fixos.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		headers:    make(map[string]string),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Get executa GET em baseURL+path e decodifica a resposta JSON em out.
// Passe nil em out para ignorar o corpo da resposta.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("httpclient: montar requisição: %w", err)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("httpclient: GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("httpclient: upstream retornou %d em %s%s", resp.StatusCode, c.baseURL, path)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("httpclient: decodificar resposta de %s: %w", path, err)
		}
	}
	return nil
}
