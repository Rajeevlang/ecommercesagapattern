package graph

import (
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
	orderv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/order/v1"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	AuthClient    authv1.AuthServiceClient
	AccountClient accountv1.AccountServiceClient
	CatalogClient catalogv1.CatalogServiceClient
	OrderClient   orderv1.OrderServiceClient
}
