package graph

import (
	"context"
	"fmt"

	"github.com/Rajeevlang/ecommercesagapattern/apigateway/graph/model"
	"github.com/Rajeevlang/ecommercesagapattern/apigateway/middleware"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	authv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/auth/v1"
	catalogv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/catalog/v1"
	orderv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/order/v1"
)

// Register is the resolver for the register field.
func (r *mutationResolver) Register(ctx context.Context, email string, password string, name string) (*model.AuthPayload, error) {
	resp, err := r.AuthClient.Register(ctx, &authv1.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		return nil, err
	}

	// Pre-create account profile
	_, _ = r.AccountClient.UpdateProfile(ctx, &accountv1.UpdateProfileRequest{
		UserId: resp.GetUserId(),
		Name:   name,
		Phone:  "",
	})

	loginResp, err := r.AuthClient.Login(ctx, &authv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:    resp.GetUserId(),
		Email: email,
		Name:  name,
	}

	return &model.AuthPayload{
		Token: loginResp.GetToken(),
		User:  user,
	}, nil
}

// Login is the resolver for the login field.
func (r *mutationResolver) Login(ctx context.Context, email string, password string) (*model.AuthPayload, error) {
	loginResp, err := r.AuthClient.Login(ctx, &authv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	profileResp, err := r.AccountClient.GetProfile(ctx, &accountv1.GetProfileRequest{
		UserId: loginResp.GetUserId(),
	})

	user := &model.User{
		ID:    loginResp.GetUserId(),
		Email: email,
	}

	if err == nil && profileResp != nil {
		user.Name = profileResp.GetName()
		phone := profileResp.GetPhone()
		user.Phone = &phone
		if profileResp.GetDefaultShippingAddress() != nil {
			addr := profileResp.GetDefaultShippingAddress()
			user.Address = &model.Address{
				Street:  addr.GetStreet(),
				City:    addr.GetCity(),
				State:   addr.GetState(),
				Country: addr.GetCountry(),
				ZipCode: addr.GetZipCode(),
			}
		}
	} else {
		user.Name = "User"
	}

	return &model.AuthPayload{
		Token: loginResp.GetToken(),
		User:  user,
	}, nil
}

// UpdateProfile is the resolver for the updateProfile field.
func (r *mutationResolver) UpdateProfile(ctx context.Context, input model.UpdateProfileInput) (*model.User, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthorized: missing auth context")
	}

	// Fetch existing profile
	profileResp, err := r.AccountClient.GetProfile(ctx, &accountv1.GetProfileRequest{UserId: userID})
	name := ""
	phone := ""
	email := ""
	if err == nil && profileResp != nil {
		name = profileResp.GetName()
		phone = profileResp.GetPhone()
		email = profileResp.GetEmail()
	}

	if input.Name != nil {
		name = *input.Name
	}
	if input.Phone != nil {
		phone = *input.Phone
	}

	_, err = r.AccountClient.UpdateProfile(ctx, &accountv1.UpdateProfileRequest{
		UserId: userID,
		Name:   name,
		Phone:  phone,
	})
	if err != nil {
		return nil, err
	}

	var finalAddr *model.Address
	if input.Address != nil {
		addrResp, err := r.AccountClient.CreateAddress(ctx, &accountv1.CreateAddressRequest{
			UserId:      userID,
			Street:      input.Address.Street,
			City:        input.Address.City,
			State:       input.Address.State,
			Country:     input.Address.Country,
			ZipCode:     input.Address.ZipCode,
			IsDefault:   true,
			AddressType: "SHIPPING",
		})
		if err == nil && addrResp != nil && addrResp.GetAddress() != nil {
			addr := addrResp.GetAddress()
			finalAddr = &model.Address{
				Street:  addr.GetStreet(),
				City:    addr.GetCity(),
				State:   addr.GetState(),
				Country: addr.GetCountry(),
				ZipCode: addr.GetZipCode(),
			}
		}
	} else if err == nil && profileResp != nil && profileResp.GetDefaultShippingAddress() != nil {
		addr := profileResp.GetDefaultShippingAddress()
		finalAddr = &model.Address{
			Street:  addr.GetStreet(),
			City:    addr.GetCity(),
			State:   addr.GetState(),
			Country: addr.GetCountry(),
			ZipCode: addr.GetZipCode(),
		}
	}

	return &model.User{
		ID:      userID,
		Email:   email,
		Name:    name,
		Phone:   &phone,
		Address: finalAddr,
	}, nil
}

// CreateOrder is the resolver for the createOrder field.
func (r *mutationResolver) CreateOrder(ctx context.Context, input model.CreateOrderInput) (*model.Order, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	var items []*orderv1.OrderItem
	for _, it := range input.Items {
		items = append(items, &orderv1.OrderItem{
			ProductId: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	resp, err := r.OrderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		UserId:             userID,
		Items:              items,
		PaymentMethodToken: input.PaymentMethodToken,
	})
	if err != nil {
		return nil, err
	}

	q := &queryResolver{r.Resolver}
	return q.Order(ctx, resp.GetOrderId())
}

// Me is the resolver for the me field.
func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	profileResp, err := r.AccountClient.GetProfile(ctx, &accountv1.GetProfileRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	phone := profileResp.GetPhone()
	user := &model.User{
		ID:    userID,
		Email: profileResp.GetEmail(),
		Name:  profileResp.GetName(),
		Phone: &phone,
	}

	if profileResp.GetDefaultShippingAddress() != nil {
		addr := profileResp.GetDefaultShippingAddress()
		user.Address = &model.Address{
			Street:  addr.GetStreet(),
			City:    addr.GetCity(),
			State:   addr.GetState(),
			Country: addr.GetCountry(),
			ZipCode: addr.GetZipCode(),
		}
	}

	return user, nil
}

// Product is the resolver for the product field.
func (r *queryResolver) Product(ctx context.Context, id string) (*model.Product, error) {
	resp, err := r.CatalogClient.GetProduct(ctx, &catalogv1.GetProductRequest{Id: id})
	if err != nil {
		return nil, err
	}

	p := resp.GetProduct()
	return &model.Product{
		ID:          p.GetId(),
		Name:        p.GetName(),
		Description: p.GetDescription(),
		PriceCents:  int32(p.GetPriceCents()),
		Stock:       p.GetStock(),
	}, nil
}

// Products is the resolver for the products field.
func (r *queryResolver) Products(ctx context.Context, first *int32, after *string) (*model.ProductConnection, error) {
	limit := int32(10)
	if first != nil {
		limit = *first
	}
	cursor := ""
	if after != nil {
		cursor = *after
	}

	resp, err := r.CatalogClient.ListProducts(ctx, &catalogv1.ListProductsRequest{
		PageSize:  limit,
		PageToken: cursor,
	})
	if err != nil {
		return nil, err
	}

	var edges []*model.ProductEdge
	for _, p := range resp.GetProducts() {
		edges = append(edges, &model.ProductEdge{
			Cursor: p.GetId(),
			Node: &model.Product{
				ID:          p.GetId(),
				Name:        p.GetName(),
				Description: p.GetDescription(),
				PriceCents:  int32(p.GetPriceCents()),
				Stock:       p.GetStock(),
			},
		})
	}

	hasNextPage := false
	var endCursor *string
	if len(edges) > 0 && len(edges) == int(limit) {
		hasNextPage = true
		c := edges[len(edges)-1].Cursor
		endCursor = &c
	}

	return &model.ProductConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasNextPage: hasNextPage,
			EndCursor:   endCursor,
		},
	}, nil
}

// Order is the resolver for the order field.
func (r *queryResolver) Order(ctx context.Context, id string) (*model.Order, error) {
	o, err := r.OrderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: id})
	if err != nil {
		return nil, err
	}

	profileResp, err := r.AccountClient.GetProfile(ctx, &accountv1.GetProfileRequest{UserId: o.GetUserId()})
	var orderUser *model.User
	if err == nil && profileResp != nil {
		phone := profileResp.GetPhone()
		orderUser = &model.User{
			ID:    o.GetUserId(),
			Email: profileResp.GetEmail(),
			Name:  profileResp.GetName(),
			Phone: &phone,
		}
		if profileResp.GetDefaultShippingAddress() != nil {
			addr := profileResp.GetDefaultShippingAddress()
			orderUser.Address = &model.Address{
				Street:  addr.GetStreet(),
				City:    addr.GetCity(),
				State:   addr.GetState(),
				Country: addr.GetCountry(),
				ZipCode: addr.GetZipCode(),
			}
		}
	} else {
		orderUser = &model.User{
			ID:   o.GetUserId(),
			Name: "User",
		}
	}

	var items []*model.OrderItem
	for _, it := range o.GetItems() {
		p, err := r.CatalogClient.GetProduct(ctx, &catalogv1.GetProductRequest{Id: it.GetProductId()})
		var prod *model.Product
		if err == nil && p != nil && p.GetProduct() != nil {
			prodItem := p.GetProduct()
			prod = &model.Product{
				ID:          prodItem.GetId(),
				Name:        prodItem.GetName(),
				Description: prodItem.GetDescription(),
				PriceCents:  int32(prodItem.GetPriceCents()),
				Stock:       prodItem.GetStock(),
			}
		} else {
			prod = &model.Product{
				ID:   it.GetProductId(),
				Name: "Product",
			}
		}

		items = append(items, &model.OrderItem{
			Product:    prod,
			Quantity:   it.GetQuantity(),
			PriceCents: int32(it.GetPriceCents()),
		})
	}

	// Map Proto Status to GraphQL Schema Status
	gqlStatus := model.OrderStatusPending
	switch o.GetStatus() {
	case orderv1.OrderStatus_ORDER_STATUS_COMPLETED:
		gqlStatus = model.OrderStatusCompleted
	case orderv1.OrderStatus_ORDER_STATUS_FAILED:
		gqlStatus = model.OrderStatusFailed
	case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
		gqlStatus = model.OrderStatusCancelled
	}

	// format timestamps
	createdAtStr := o.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00")
	updatedAtStr := o.GetUpdatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00")

	return &model.Order{
		ID:               o.GetOrderId(),
		User:             orderUser,
		Items:            items,
		TotalAmountCents: int32(o.GetTotalAmountCents()),
		Status:           gqlStatus,
		CreatedAt:        createdAtStr,
		UpdatedAt:        updatedAtStr,
	}, nil
}

// MyOrders is the resolver for the myOrders field.
func (r *queryResolver) MyOrders(ctx context.Context, limit *int32, offset *int32) ([]*model.Order, error) {
	// The order gRPC microservice currently does not expose a list endpoint.
	// We return an empty slice to keep queries from failing.
	return []*model.Order{}, nil
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
