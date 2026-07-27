package grpc

import (
	"context"

	"github.com/Rajeevlang/ecommercesagapattern/account/internal/domain"
	"github.com/Rajeevlang/ecommercesagapattern/account/internal/ports"
	accountv1 "github.com/Rajeevlang/ecommercesagapattern/shared/pb/account/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AccountGRPCHandler struct {
	accountv1.UnimplementedAccountServiceServer
	service ports.AccountService
}

func NewAccountGRPCHandler(service ports.AccountService) *AccountGRPCHandler {
	return &AccountGRPCHandler{service: service}
}

func (h *AccountGRPCHandler) GetProfile(ctx context.Context, req *accountv1.GetProfileRequest) (*accountv1.GetProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	profile, defaultAddr, err := h.service.GetProfile(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	res := &accountv1.GetProfileResponse{
		UserId:    profile.UserID,
		Email:     profile.Email,
		Name:      profile.Name,
		Phone:     profile.Phone,
		AvatarUrl: profile.AvatarURL,
		CreatedAt: timestamppb.New(profile.CreatedAt),
		UpdatedAt: timestamppb.New(profile.UpdatedAt),
	}

	if defaultAddr != nil {
		res.DefaultShippingAddress = &accountv1.Address{
			Id:          defaultAddr.ID,
			UserId:      defaultAddr.UserID,
			Street:      defaultAddr.Street,
			City:        defaultAddr.City,
			State:       defaultAddr.State,
			Country:     defaultAddr.Country,
			ZipCode:     defaultAddr.ZipCode,
			IsDefault:   defaultAddr.IsDefault,
			AddressType: defaultAddr.AddressType,
		}
	}

	return res, nil
}

func (h *AccountGRPCHandler) UpdateProfile(ctx context.Context, req *accountv1.UpdateProfileRequest) (*accountv1.UpdateProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := h.service.UpdateProfile(ctx, req.GetUserId(), req.GetName(), req.GetPhone(), req.GetAvatarUrl())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &accountv1.UpdateProfileResponse{Success: true, UserId: req.GetUserId()}, nil
}

func (h *AccountGRPCHandler) CreateAddress(ctx context.Context, req *accountv1.CreateAddressRequest) (*accountv1.CreateAddressResponse, error) {
	addr := &domain.Address{
		UserID:      req.GetUserId(),
		Street:      req.GetStreet(),
		City:        req.GetCity(),
		State:       req.GetState(),
		Country:     req.GetCountry(),
		ZipCode:     req.GetZipCode(),
		IsDefault:   req.GetIsDefault(),
		AddressType: req.GetAddressType(),
	}

	created, err := h.service.CreateAddress(ctx, addr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &accountv1.CreateAddressResponse{
		Address: &accountv1.Address{
			Id:          created.ID,
			UserId:      created.UserID,
			Street:      created.Street,
			City:        created.City,
			State:       created.State,
			Country:     created.Country,
			ZipCode:     created.ZipCode,
			IsDefault:   created.IsDefault,
			AddressType: created.AddressType,
		},
	}, nil
}

func (h *AccountGRPCHandler) ListAddresses(ctx context.Context, req *accountv1.ListAddressesRequest) (*accountv1.ListAddressesResponse, error) {
	addresses, err := h.service.ListAddresses(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbAddrs []*accountv1.Address
	for _, a := range addresses {
		pbAddrs = append(pbAddrs, &accountv1.Address{
			Id:          a.ID,
			UserId:      a.UserID,
			Street:      a.Street,
			City:        a.City,
			State:       a.State,
			Country:     a.Country,
			ZipCode:     a.ZipCode,
			IsDefault:   a.IsDefault,
			AddressType: a.AddressType,
		})
	}

	return &accountv1.ListAddressesResponse{Addresses: pbAddrs}, nil
}

func (h *AccountGRPCHandler) DeleteAddress(ctx context.Context, req *accountv1.DeleteAddressRequest) (*accountv1.DeleteAddressResponse, error) {
	if req.GetUserId() == "" || req.GetAddressId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and address_id are required")
	}

	err := h.service.DeleteAddress(ctx, req.GetUserId(), req.GetAddressId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &accountv1.DeleteAddressResponse{Success: true}, nil
}

func (h *AccountGRPCHandler) SetDefaultAddress(ctx context.Context, req *accountv1.SetDefaultAddressRequest) (*accountv1.SetDefaultAddressResponse, error) {
	if req.GetUserId() == "" || req.GetAddressId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and address_id are required")
	}

	addrType := req.GetAddressType()
	if addrType == "" {
		addrType = "SHIPPING"
	}

	err := h.service.SetDefaultAddress(ctx, req.GetUserId(), req.GetAddressId(), addrType)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &accountv1.SetDefaultAddressResponse{Success: true}, nil
}
