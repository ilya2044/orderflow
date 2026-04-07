package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/diploma/product-service/internal/domain"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

const indexName = "products"

type SearchRepository struct {
	client *elasticsearch.Client
	log    *zap.Logger
}

func NewSearchRepository(client *elasticsearch.Client, log *zap.Logger) *SearchRepository {
	return &SearchRepository{client: client, log: log}
}

func (r *SearchRepository) CreateIndex(ctx context.Context) error {
	mapping := `{
		"mappings": {
			"properties": {
				"id":          { "type": "keyword" },
				"name":        { "type": "text",    "analyzer": "standard" },
				"description": { "type": "text",    "analyzer": "standard" },
				"price":       { "type": "float" },
				"category":    { "type": "keyword" },
				"sku":         { "type": "keyword" },
				"tags":        { "type": "keyword" },
				"is_active":   { "type": "boolean" },
				"rating":      { "type": "float" },
				"created_at":  { "type": "date" }
			}
		},
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 0
		}
	}`

	res, err := r.client.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	res.Body.Close()

	if res.StatusCode == 200 {
		return nil
	}

	res, err = r.client.Indices.Create(
		indexName,
		r.client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}
	return nil
}

func (r *SearchRepository) IndexProduct(ctx context.Context, p *domain.Product) error {
	doc := map[string]interface{}{
		"id":          p.ID.Hex(),
		"name":        p.Name,
		"description": p.Description,
		"price":       p.Price,
		"category":    p.Category,
		"sku":         p.SKU,
		"tags":        p.Tags,
		"is_active":   p.IsActive,
		"rating":      p.Rating,
		"created_at":  p.CreatedAt,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	res, err := r.client.Index(
		indexName,
		bytes.NewReader(body),
		r.client.Index.WithDocumentID(p.ID.Hex()),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch index error: %s", res.String())
	}
	return nil
}

func (r *SearchRepository) Search(ctx context.Context, req *domain.SearchRequest) ([]string, int64, error) {
	must := []map[string]interface{}{}
	filter := []map[string]interface{}{
		{"term": map[string]interface{}{"is_active": true}},
	}

	if req.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  req.Query,
				"fields": []string{"name^3", "description", "tags"},
				"type":   "best_fields",
			},
		})
	}

	if req.Category != "" {
		filter = append(filter, map[string]interface{}{
			"term": map[string]interface{}{"category": req.Category},
		})
	}

	priceRange := map[string]interface{}{}
	if req.MinPrice > 0 {
		priceRange["gte"] = req.MinPrice
	}
	if req.MaxPrice > 0 {
		priceRange["lte"] = req.MaxPrice
	}
	if len(priceRange) > 0 {
		filter = append(filter, map[string]interface{}{
			"range": map[string]interface{}{"price": priceRange},
		})
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   must,
				"filter": filter,
			},
		},
		"from": (req.Page - 1) * req.Limit,
		"size": req.Limit,
		"sort": []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"rating": map[string]interface{}{"order": "desc"}},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}

	res, err := r.client.Search(
		r.client.Search.WithIndex(indexName),
		r.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch search error: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, err
	}

	hits := result["hits"].(map[string]interface{})
	total := int64(hits["total"].(map[string]interface{})["value"].(float64))

	var ids []string
	for _, hit := range hits["hits"].([]interface{}) {
		h := hit.(map[string]interface{})
		ids = append(ids, h["_id"].(string))
	}

	return ids, total, nil
}

func (r *SearchRepository) DeleteProduct(ctx context.Context, id string) error {
	res, err := r.client.Delete(indexName, id)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (r *SearchRepository) UpdateProduct(ctx context.Context, id string, updates map[string]interface{}) error {
	body, err := json.Marshal(map[string]interface{}{"doc": updates})
	if err != nil {
		return err
	}

	res, err := r.client.Update(indexName, id, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch update error: %s", res.String())
	}
	return nil
}
