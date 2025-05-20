package models

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Product struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title          string             `json:"title" validate:"required" bson:"title"`
	Description    string             `json:"description" validate:"required" bson:"description"`
	Price          float64            `json:"price" validate:"required,min=0" bson:"price"`
	Discount       float64            `json:"discount" bson:"discount"`
	Slug           string             `json:"slug" bson:"slug"` // Made slug optional for creation
	Category       []string           `json:"category" validate:"required,dive,min=1,max=50" bson:"category"`
	Images         []string           `json:"images" validate:"required,dive,url" bson:"images"`
	Tags           []string           `json:"tags" validate:"required,dive,min=1,max=50" bson:"tags"`
	IsAvailable    bool               `json:"is_available" bson:"is_available"`
	IsNew          bool               `json:"is_new" bson:"is_new"`
	IsOnSale       bool               `json:"is_on_sale" bson:"is_on_sale"`
	SalesStartDate time.Time          `json:"sales_start_date,omitempty" bson:"sales_start_date,omitempty"`
	SalesEndDate   time.Time          `json:"sales_end_date,omitempty" bson:"sales_end_date,omitempty"`
	Models         []string           `json:"models" validate:"required,dive,min=1,max=50" bson:"models"`
	Colors         []string           `json:"colors" validate:"required,dive,min=1,max=50" bson:"colors"`
	Materials      []string           `json:"materials,omitempty" validate:"required" bson:"materials,omitempty"`
	Warranty       int                `json:"warranty,omitempty" bson:"warranty,omitempty"`
	Details        []string           `json:"details,omitempty" validate:"required" bson:"details,omitempty"`
	Features       []string           `json:"features,omitempty" validate:"required" bson:"features,omitempty"`
	Stock          int                `json:"stock" validate:"required" bson:"stock"`
	Comments       []Comments         `json:"comments,omitempty" bson:"comments,omitempty"`
	Rating         float64            `json:"rating,omitempty" bson:"rating,omitempty"`
	ReviewCount    int                `json:"reviewCount,omitempty" bson:"reviewCount,omitempty"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"UpdatedAt" bson:"UpdatedAt"`
}

type Comments struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProductID primitive.ObjectID `json:"productId" bson:"productId"`
	UserId    string             `json:"userId" bson:"userId" validate:"required"`
	Comment   string             `json:"comment" bson:"comment" validate:"required"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"UpdatedAt" bson:"UpdatedAt"`
}

// type Dimensions struct {
// 	Length float64 `json:"length,omitempty"` // In mm
// 	Width  float64 `json:"width,omitempty"`  // In mm
// 	Height float64 `json:"height,omitempty"` // In mm
// }

type Address struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type         string             `json:"type" validate:"required,eq=home|eq=work|eq=other"` // home, work, other
	AddressLine1 string             `json:"address_line1" validate:"required"`
	AddressLine2 string             `json:"address_line2,omitempty"`
	City         string             `json:"city" validate:"required"`
	State        string             `json:"state" validate:"required"`
	PostalCode   string             `json:"postal_code" validate:"required"`
	Country      string             `json:"country" validate:"required"`
	IsDefault    bool               `json:"is_default"`
}

type Order struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID `json:"user_id" bson:"user_id"`
	OrderItems      []OrderItem        `json:"order_items" validate:"required,dive"`
	TotalAmount     float64            `json:"total_amount" validate:"required,min=0"`
	Discount        float64            `json:"discount"`
	ShippingFee     float64            `json:"shipping_fee"`
	FinalAmount     float64            `json:"final_amount"`
	Status          string             `json:"status" validate:"required,eq=pending|eq=processing|eq=shipped|eq=delivered|eq=cancelled"`
	PaymentID       primitive.ObjectID `json:"payment_id,omitempty" bson:"payment_id,omitempty"`
	PaymentStatus   string             `json:"payment_status" validate:"required,eq=pending|eq=completed|eq=failed|eq=refunded"`
	ShippingAddress Address            `json:"shipping_address" validate:"required"`
	TrackingNumber  string             `json:"tracking_number,omitempty"`
	ShippedAt       time.Time          `json:"shipped_at,omitempty"`
	DeliveredAt     time.Time          `json:"delivered_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type OrderItem struct {
	ProductID    primitive.ObjectID `json:"product_id" validate:"required"`
	ProductTitle string             `json:"product_title" validate:"required"`
	Quantity     int                `json:"quantity" validate:"required,min=1"`
	Price        float64            `json:"price" validate:"required,min=0"`
	TotalPrice   float64            `json:"total_price"`
	Color        string             `json:"color,omitempty"`
}

type Cart struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Items       []CartItem         `json:"items"`
	TotalAmount float64            `json:"total_amount" bson:"total_amount"` // Explicitly set the BSON tag
	CreatedAt   time.Time          `json:"created_at" bson:"createdat"`      // Match the createdat field in DB
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`     // This one is already correct
}

type CartItem struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID  primitive.ObjectID `json:"product_id" bson:"productid,omitempty" validate:"required"` // Changed field tag to match DB
	Quantity   int                `json:"quantity" bson:"quantity" validate:"required,min=1"`
	Color      string             `json:"color,omitempty" bson:"color" validate:"required"`
	Image      string             `json:"image" bson:"image"`
	Slug       string             `json:"slug" bson:"slug"`
	Title      string             `json:"title" bson:"title"`
	Price      float64            `json:"price" bson:"price"`
	Model      string             `json:"model,omitempty" bson:"model"`
	TotalPrice float64            `json:"total_price" bson:"totalprice"` // Changed field tag to match DB
}

type Review struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID  primitive.ObjectID `json:"product_id" validate:"required"`
	UserID     primitive.ObjectID `json:"user_id" validate:"required"`
	UserName   string             `json:"user_name" validate:"required"`
	Rating     float64            `json:"rating" validate:"required,min=1,max=5"`
	Title      string             `json:"title" validate:"required,min=4,max=100"`
	Comment    string             `json:"comment" validate:"required,min=10,max=500"`
	Images     []string           `json:"images,omitempty"`
	IsVerified bool               `json:"is_verified"` // If the user purchased the product
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type Payment struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID       primitive.ObjectID `json:"order_id" bson:"order_id"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	Amount        float64            `json:"amount" validate:"required,min=0"`
	PaymentMethod string             `json:"payment_method" validate:"required,eq=credit_card|eq=paypal|eq=apple_pay|eq=google_pay|eq=bank_transfer"`
	TransactionID string             `json:"transaction_id,omitempty"`
	Status        string             `json:"status" validate:"required,eq=pending|eq=completed|eq=failed|eq=refunded"`
	PaymentDate   time.Time          `json:"payment_date,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type CartActions struct {
	Increment bool `json:"increment"`
	Decrement bool `json:"decrement"`
}
type ShopCalls interface {
	// Product Operations
	AddProduct(ctx context.Context, product Product) (string, error)
	GetProductByID(ctx context.Context, id primitive.ObjectID) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, id primitive.ObjectID, product map[string]interface{}) (string, error)
	DeleteProduct(ctx context.Context, id primitive.ObjectID) error
	ListProducts(ctx context.Context, page, limit int, filter map[string]interface{}) ([]Product, int64, error)
	BuildQuery(ctx context.Context, filter map[string]interface{}) (primitive.M, error)
	GetSimilarProducts(ctx context.Context, product primitive.ObjectID) ([]Product, error)
	// Comment Operations
	AddComment(ctx context.Context, comment Comments, userId string, productId primitive.ObjectID) error
	GetCommentsByProductID(ctx context.Context, productID primitive.ObjectID) ([]Comments, error)
	UpdateComment(ctx context.Context, comment Comments) (string, error)
	GetCommentByID(ctx context.Context, id primitive.ObjectID) (*Comments, error)
	DeleteComment(ctx context.Context, id primitive.ObjectID, userId string) error

	// Cart Operations
	GetUserCart(ctx context.Context, userID primitive.ObjectID) (*Cart, error)
	AddToCart(ctx context.Context, userID primitive.ObjectID, item CartItem) error
	UpdateCartItem(ctx context.Context, userID primitive.ObjectID, item CartItem, actions CartActions) error
	RemoveCartItem(ctx context.Context, userID primitive.ObjectID, cartItemID primitive.ObjectID) error
	ClearCart(ctx context.Context, userID primitive.ObjectID) error

	// Order Operations
	CreateOrder(ctx context.Context, order Order) error
	GetOrderByID(ctx context.Context, id primitive.ObjectID) (*Order, error)
	UpdateOrderStatus(ctx context.Context, id primitive.ObjectID, status string) error
	GetUserOrders(ctx context.Context, userID primitive.ObjectID) ([]Order, error)

	// Review Operations
	AddReview(ctx context.Context, review Review) error
	GetProductReviews(ctx context.Context, productID primitive.ObjectID) ([]Review, error)

	// Payment Operations
	ProcessPayment(ctx context.Context, payment Payment) error
	GetPaymentByID(ctx context.Context, id primitive.ObjectID) (*Payment, error)

	// Coupon Operations
	// ValidateCoupon(code string, amount float64) (*Coupon, error)
	ApplyCoupon(code string, orderID primitive.ObjectID) error
}

func NewMongoClient(client *mongo.Client) *MongoClient {
	return &MongoClient{client: client}
}

type MongoClient struct {
	client *mongo.Client
}

var validate = validator.New()
