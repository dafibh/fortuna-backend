package service

import (
	"testing"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/domain"
	"github.com/dafibh/fortuna/fortuna-backend/internal/testutil"
	"github.com/shopspring/decimal"
)

func TestCreatePrice_Success(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(99.99),
	}

	price, err := svc.CreatePrice(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if price.PlatformName != "Lazada" {
		t.Errorf("expected platform name 'Lazada', got '%s'", price.PlatformName)
	}
	if !price.Price.Equal(decimal.NewFromFloat(99.99)) {
		t.Errorf("expected price 99.99, got %s", price.Price.String())
	}
	if price.ItemID != 1 {
		t.Errorf("expected itemID 1, got %d", price.ItemID)
	}
}

func TestCreatePrice_TrimsPlatformName(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "  Shopee  ",
		Price:        decimal.NewFromFloat(50.00),
	}

	price, err := svc.CreatePrice(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if price.PlatformName != "Shopee" {
		t.Errorf("expected trimmed platform name 'Shopee', got '%s'", price.PlatformName)
	}
}

func TestCreatePrice_EmptyPlatformName(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "",
		Price:        decimal.NewFromFloat(50.00),
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPricePlatformEmpty {
		t.Errorf("expected ErrPricePlatformEmpty, got %v", err)
	}
}

func TestCreatePrice_WhitespaceOnlyPlatformName(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "   ",
		Price:        decimal.NewFromFloat(50.00),
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPricePlatformEmpty {
		t.Errorf("expected ErrPricePlatformEmpty, got %v", err)
	}
}

func TestCreatePrice_PlatformNameTooLong(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}
	input := CreatePriceInput{
		PlatformName: string(longName),
		Price:        decimal.NewFromFloat(50.00),
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPricePlatformLong {
		t.Errorf("expected ErrPricePlatformLong, got %v", err)
	}
}

func TestCreatePrice_ZeroPrice(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.Zero,
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPriceNotPositive {
		t.Errorf("expected ErrPriceNotPositive, got %v", err)
	}
}

func TestCreatePrice_NegativePrice(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(-10.00),
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPriceNotPositive {
		t.Errorf("expected ErrPriceNotPositive, got %v", err)
	}
}

func TestCreatePrice_FutureDate(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	futureDate := time.Now().AddDate(0, 0, 7) // 7 days in the future
	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(50.00),
		PriceDate:    &futureDate,
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPriceDateFuture {
		t.Errorf("expected ErrPriceDateFuture, got %v", err)
	}
}

func TestCreatePrice_ValidPastDate(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	pastDate := time.Now().AddDate(0, 0, -7) // 7 days ago
	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(50.00),
		PriceDate:    &pastDate,
	}

	price, err := svc.CreatePrice(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !price.PriceDate.Equal(pastDate) {
		t.Errorf("expected price date %v, got %v", pastDate, price.PriceDate)
	}
}

func TestCreatePrice_InvalidImageURL(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(50.00),
		ImageURL:     strPtr("not-a-valid-url"),
	}

	_, err := svc.CreatePrice(1, 1, input)
	if err != domain.ErrPriceInvalidImageURL {
		t.Errorf("expected ErrPriceInvalidImageURL, got %v", err)
	}
}

func TestCreatePrice_ValidImageURL(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(50.00),
		ImageURL:     strPtr("https://example.com/price-screenshot.png"),
	}

	price, err := svc.CreatePrice(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if *price.ImageURL != "https://example.com/price-screenshot.png" {
		t.Errorf("expected image URL 'https://example.com/price-screenshot.png', got '%s'", *price.ImageURL)
	}
}

func TestCreatePrice_ItemNotFound(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	// Note: item not added

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	input := CreatePriceInput{
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(50.00),
	}

	_, err := svc.CreatePrice(1, 999, input)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestListPricesByItem_Success(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           1,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(100.00),
		PriceDate:    time.Now(),
	})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           2,
		ItemID:       1,
		PlatformName: "Shopee",
		Price:        decimal.NewFromFloat(90.00),
		PriceDate:    time.Now(),
	})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	prices, err := svc.ListPricesByItem(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(prices) != 2 {
		t.Errorf("expected 2 prices, got %d", len(prices))
	}
}

func TestListPricesByItem_ItemNotFound(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	// Note: item not added

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	_, err := svc.ListPricesByItem(1, 999)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}

func TestGetPricesGroupedByPlatform_Success(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           1,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(100.00),
		PriceDate:    time.Now().AddDate(0, 0, -7),
	})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           2,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(95.00),
		PriceDate:    time.Now(),
	})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           3,
		ItemID:       1,
		PlatformName: "Shopee",
		Price:        decimal.NewFromFloat(90.00),
		PriceDate:    time.Now(),
	})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	groups, err := svc.GetPricesGroupedByPlatform(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 platform groups, got %d", len(groups))
	}

	// Find Shopee - should be marked as lowest
	var shopeeGroup *domain.PriceByPlatform
	for _, g := range groups {
		if g.PlatformName == "Shopee" {
			shopeeGroup = g
			break
		}
	}
	if shopeeGroup == nil {
		t.Fatal("expected to find Shopee group")
	}
	if !shopeeGroup.IsLowestPrice {
		t.Error("expected Shopee to be marked as lowest price")
	}
}

func TestGetPricesGroupedByPlatform_Empty(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	groups, err := svc.GetPricesGroupedByPlatform(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 platform groups, got %d", len(groups))
	}
}

func TestGetBestPriceForItem_Success(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           1,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(100.00),
		PriceDate:    time.Now(),
	})
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           2,
		ItemID:       1,
		PlatformName: "Shopee",
		Price:        decimal.NewFromFloat(90.00),
		PriceDate:    time.Now(),
	})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	bestPrice, err := svc.GetBestPriceForItem(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bestPrice == nil {
		t.Fatal("expected best price, got nil")
	}
	if *bestPrice != "90" {
		t.Errorf("expected best price '90', got '%s'", *bestPrice)
	}
}

func TestGetBestPriceForItem_NoPrices(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	bestPrice, err := svc.GetBestPriceForItem(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bestPrice != nil {
		t.Errorf("expected nil best price, got '%s'", *bestPrice)
	}
}

func TestDeletePrice_Success(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           1,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(100.00),
		PriceDate:    time.Now(),
	})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	err := svc.DeletePrice(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = priceRepo.GetByID(1, 1)
	if err != domain.ErrPriceEntryNotFound {
		t.Errorf("expected ErrPriceEntryNotFound after delete, got %v", err)
	}
}

func TestDeletePrice_NotFound(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	err := svc.DeletePrice(1, 999)
	if err != domain.ErrPriceEntryNotFound {
		t.Errorf("expected ErrPriceEntryNotFound, got %v", err)
	}
}

func TestGetCurrentPricesByItem_Success(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()
	itemRepo.AddItem(&domain.WishlistItem{ID: 1, WishlistID: 1, Title: "Test Item"})
	// Add older price for Lazada
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           1,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(100.00),
		PriceDate:    time.Now().AddDate(0, 0, -7),
	})
	// Add newer price for Lazada
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           2,
		ItemID:       1,
		PlatformName: "Lazada",
		Price:        decimal.NewFromFloat(95.00),
		PriceDate:    time.Now(),
	})
	// Add price for Shopee
	priceRepo.AddPrice(&domain.WishlistItemPrice{
		ID:           3,
		ItemID:       1,
		PlatformName: "Shopee",
		Price:        decimal.NewFromFloat(90.00),
		PriceDate:    time.Now(),
	})

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	currentPrices, err := svc.GetCurrentPricesByItem(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(currentPrices) != 2 {
		t.Errorf("expected 2 current prices, got %d", len(currentPrices))
	}
}

func TestGetCurrentPricesByItem_ItemNotFound(t *testing.T) {
	priceRepo := testutil.NewMockWishlistPriceRepository()
	itemRepo := testutil.NewMockWishlistItemRepository()

	svc := NewWishlistPriceService(priceRepo, itemRepo)

	_, err := svc.GetCurrentPricesByItem(1, 999)
	if err != domain.ErrWishlistItemNotFound {
		t.Errorf("expected ErrWishlistItemNotFound, got %v", err)
	}
}
