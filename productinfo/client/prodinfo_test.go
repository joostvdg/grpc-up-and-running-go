package main

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"productinfo/client/ecommerce"
	"productinfo/client/mock_ecommerce"
	"testing"
	"time"
)

func TestAddProduct(t *testing.T) {
	t.Log("TestAddProduct")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProductInfoClient := mock_ecommerce.NewMockProductInfoClient(ctrl)

	req := &ecommerce.Product{Name: "Apple iPhone 11", Description: "Meet Apple iPhone 11. All-new dual-camera system with Ultra Wide and Night mode. All-day battery. Six new colors.", Price: 1000.00}
	mockProductInfoClient.EXPECT().AddProduct(gomock.Any(), req).Return(&ecommerce.ProductID{Value: "12345"}, nil)
	testAddProduct(t, mockProductInfoClient)
}

func testAddProduct(t *testing.T, mockProductInfoClient *mock_ecommerce.MockProductInfoClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r, err := mockProductInfoClient.AddProduct(ctx, &ecommerce.Product{Name: "Apple iPhone 11", Description: "Meet Apple iPhone 11. All-new dual-camera system with Ultra Wide and Night mode. All-day battery. Six new colors.", Price: 1000.00})
	if err != nil {
		t.Fatalf("Could not add product: %v", err)
	}
	t.Logf("Product ID: %s added successfully", r.Value)
	assert.NotEmpty(t, r.Value)
}
