package mongo

import (
	"context"
	"time"

	"github.com/diploma/product-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepository struct {
	col *mongo.Collection
}

func NewProductRepository(db *mongo.Database) *ProductRepository {
	col := db.Collection("products")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _ = col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "sku", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "price", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "description", Value: "text"},
				{Key: "tags", Value: "text"},
			},
		},
	})

	return &ProductRepository{col: col}
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	if p.ID.IsZero() {
		p.ID = primitive.NewObjectID()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.IsActive = true

	_, err := r.col.InsertOne(ctx, p)
	return err
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrProductNotFound
	}

	var p domain.Product
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrProductNotFound
	}
	return &p, err
}

func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	var p domain.Product
	err := r.col.FindOne(ctx, bson.M{"sku": sku}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrProductNotFound
	}
	return &p, err
}

func (r *ProductRepository) List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error) {
	query := bson.M{}

	if filter.Category != "" {
		query["category"] = filter.Category
	}
	if filter.MinPrice > 0 {
		if _, ok := query["price"]; !ok {
			query["price"] = bson.M{}
		}
		query["price"].(bson.M)["$gte"] = filter.MinPrice
	}
	if filter.MaxPrice > 0 {
		if _, ok := query["price"]; !ok {
			query["price"] = bson.M{}
		}
		query["price"].(bson.M)["$lte"] = filter.MaxPrice
	}
	if filter.IsActive != nil {
		query["is_active"] = *filter.IsActive
	}
	if filter.Search != "" {
		query["$text"] = bson.M{"$search": filter.Search}
	}

	total, err := r.col.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	sortField := "created_at"
	sortDir := -1
	if filter.SortBy != "" {
		sortField = filter.SortBy
	}
	if filter.SortDir == "asc" {
		sortDir = 1
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortDir}}).
		SetLimit(int64(filter.Limit)).
		SetSkip(int64((filter.Page - 1) * filter.Limit))

	cursor, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	if products == nil {
		products = []*domain.Product{}
	}

	return products, total, nil
}

func (r *ProductRepository) Update(ctx context.Context, id string, updates bson.M) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrProductNotFound
	}

	updates["updated_at"] = time.Now()
	result, err := r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{"$set": updates},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrProductNotFound
	}

	result, err := r.col.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *ProductRepository) UpdateStock(ctx context.Context, id string, delta int) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrProductNotFound
	}

	_, err = r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{
			"$inc": bson.M{"stock": delta},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

func (r *ProductRepository) AddImage(ctx context.Context, id, imageURL string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrProductNotFound
	}

	_, err = r.col.UpdateOne(ctx,
		bson.M{"_id": oid},
		bson.M{
			"$push": bson.M{"images": imageURL},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	return err
}
