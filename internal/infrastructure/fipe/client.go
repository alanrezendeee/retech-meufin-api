package fipe

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/retechfin/retechfin-api/internal/infrastructure/cache"
	"github.com/retechfin/retechfin-api/pkg/httpclient"
)

const (
	DefaultBaseURL = "https://parallelum.com.br/fipe/api/v1"

	cacheTTLBrands = 24 * time.Hour
	cacheTTLModels = 24 * time.Hour
	cacheTTLYears  = 24 * time.Hour
	cacheTTLPrice  = 2 * time.Hour
)

// Searcher define a interface de consulta FIPE usada pelo application layer.
// Permite mock em testes e troca futura de provedor.
type Searcher interface {
	ListBrands(ctx context.Context, vehicleType string) ([]Brand, error)
	ListModels(ctx context.Context, vehicleType, brandCode string) ([]Model, error)
	ListYears(ctx context.Context, vehicleType, brandCode, modelCode string) ([]Year, error)
	GetPrice(ctx context.Context, vehicleType, brandCode, modelCode, yearCode string) (*Price, error)
	GetAllYearPrices(ctx context.Context, vehicleType, brandCode, modelCode string) ([]Price, error)
}

// Client consulta a tabela FIPE via HTTP com cache Redis opcional.
type Client struct {
	http  *httpclient.Client
	cache *cache.Cache
}

// New cria um Client FIPE.
// baseURL padrão: parallelum. cache pode ser nil (sem caching — chama API direto).
func New(baseURL string, c *cache.Cache) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		http:  httpclient.New(baseURL, httpclient.WithTimeout(15*time.Second)),
		cache: c,
	}
}

// ListBrands retorna todas as marcas para o tipo de veículo.
func (c *Client) ListBrands(ctx context.Context, vehicleType string) ([]Brand, error) {
	key := fmt.Sprintf("fipe:brands:%s", vehicleType)
	if cached := c.getCache(ctx, key); cached != "" {
		var out []Brand
		if json.Unmarshal([]byte(cached), &out) == nil {
			return out, nil
		}
	}
	var out []Brand
	if err := c.http.Get(ctx, fmt.Sprintf("/%s/marcas", vehicleType), &out); err != nil {
		return nil, fmt.Errorf("fipe: marcas: %w", err)
	}
	c.setCache(ctx, key, out, cacheTTLBrands)
	return out, nil
}

// ListModels retorna os modelos de uma marca.
func (c *Client) ListModels(ctx context.Context, vehicleType, brandCode string) ([]Model, error) {
	key := fmt.Sprintf("fipe:models:%s:%s", vehicleType, brandCode)
	if cached := c.getCache(ctx, key); cached != "" {
		var out []Model
		if json.Unmarshal([]byte(cached), &out) == nil {
			return out, nil
		}
	}
	var resp ModelsResponse
	if err := c.http.Get(ctx, fmt.Sprintf("/%s/marcas/%s/modelos", vehicleType, brandCode), &resp); err != nil {
		return nil, fmt.Errorf("fipe: modelos: %w", err)
	}
	c.setCache(ctx, key, resp.Models, cacheTTLModels)
	return resp.Models, nil
}

// ListYears retorna os anos-combustível disponíveis para um modelo.
func (c *Client) ListYears(ctx context.Context, vehicleType, brandCode, modelCode string) ([]Year, error) {
	key := fmt.Sprintf("fipe:years:%s:%s:%s", vehicleType, brandCode, modelCode)
	if cached := c.getCache(ctx, key); cached != "" {
		var out []Year
		if json.Unmarshal([]byte(cached), &out) == nil {
			return out, nil
		}
	}
	var out []Year
	if err := c.http.Get(ctx, fmt.Sprintf("/%s/marcas/%s/modelos/%s/anos", vehicleType, brandCode, modelCode), &out); err != nil {
		return nil, fmt.Errorf("fipe: anos: %w", err)
	}
	c.setCache(ctx, key, out, cacheTTLYears)
	return out, nil
}

// GetPrice retorna o preço FIPE atual para uma combinação veículo/marca/modelo/ano.
func (c *Client) GetPrice(ctx context.Context, vehicleType, brandCode, modelCode, yearCode string) (*Price, error) {
	key := fmt.Sprintf("fipe:price:%s:%s:%s:%s", vehicleType, brandCode, modelCode, yearCode)
	if cached := c.getCache(ctx, key); cached != "" {
		var out Price
		if json.Unmarshal([]byte(cached), &out) == nil {
			return &out, nil
		}
	}
	var out Price
	path := fmt.Sprintf("/%s/marcas/%s/modelos/%s/anos/%s", vehicleType, brandCode, modelCode, yearCode)
	if err := c.http.Get(ctx, path, &out); err != nil {
		return nil, fmt.Errorf("fipe: preço: %w", err)
	}
	c.setCache(ctx, key, out, cacheTTLPrice)
	return &out, nil
}

// GetAllYearPrices retorna o preço FIPE para todos os anos disponíveis de um modelo.
// Constrói a curva "desde o lançamento" para o indicador de depreciação histórica.
// Melhor esforço: ignora anos individuais que falham.
func (c *Client) GetAllYearPrices(ctx context.Context, vehicleType, brandCode, modelCode string) ([]Price, error) {
	years, err := c.ListYears(ctx, vehicleType, brandCode, modelCode)
	if err != nil {
		return nil, err
	}
	prices := make([]Price, 0, len(years))
	for _, y := range years {
		p, err := c.GetPrice(ctx, vehicleType, brandCode, modelCode, y.Code)
		if err != nil {
			continue
		}
		prices = append(prices, *p)
	}
	return prices, nil
}

// ParseFipeValue converte "R$ 62.839,00" → 62839.00.
func ParseFipeValue(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "R$ ")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (c *Client) getCache(ctx context.Context, key string) string {
	v, _ := c.cache.Get(ctx, key)
	return v
}

func (c *Client) setCache(ctx context.Context, key string, v any, ttl time.Duration) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = c.cache.Set(ctx, key, string(b), ttl)
}
