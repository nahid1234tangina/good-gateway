//nolint:nolintlint,dupl
package good

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	constant "github.com/NpoolPlatform/good-middleware/pkg/message/const"
	commontracer "github.com/NpoolPlatform/good-middleware/pkg/tracer"

	"go.opentelemetry.io/otel"
	scodes "go.opentelemetry.io/otel/codes"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NpoolPlatform/go-service-framework/pkg/logger"
	mgrpb "github.com/NpoolPlatform/message/npool/good/mgr/v1/good"

	npool "github.com/NpoolPlatform/message/npool/good/gw/v1/good"

	goodmwcli "github.com/NpoolPlatform/good-middleware/pkg/client/good"

	"github.com/google/uuid"

	coininfocli "github.com/NpoolPlatform/sphinx-coininfo/pkg/client"
)

// nolint
func (s *Server) CreateGood(ctx context.Context, in *npool.CreateGoodRequest) (*npool.CreateGoodResponse, error) {
	var err error

	_, span := otel.Tracer(constant.ServiceName).Start(ctx, "CreateGood")
	defer span.End()

	defer func() {
		if err != nil {
			span.SetStatus(scodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	// TODO: Check if device exist
	// TODO: Check inherit from good exist
	// TODO: Check vendor location exist

	if _, err := uuid.Parse(in.GetInfo().GetCoinTypeID()); err != nil {
		logger.Sugar().Errorw("CreateGood", "CoinTypeID", in.GetInfo().GetCoinTypeID(), "error", err)
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	if in.GetInfo().GetDurationDays() <= 0 {
		logger.Sugar().Errorw("CreateGood", "DurationDays", in.GetInfo().GetDurationDays())
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "DurationDays is invalid")
	}

	if price, err := decimal.NewFromString(in.GetInfo().GetPrice()); err != nil || price.Cmp(decimal.NewFromInt(0)) <= 0 {
		logger.Sugar().Errorw("CreateGood", "Price", in.GetInfo().GetPrice(), "error", err)
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "Price is invalid")
	}

	switch in.GetInfo().GetBenefitType() {
	case mgrpb.BenefitType_BenefitTypePlatform:
	case mgrpb.BenefitType_BenefitTypePool:
	default:
		logger.Sugar().Errorw("CreateGood", "BenefitType", in.GetInfo().GetBenefitType())
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "BenefitType is invalid")
	}

	switch in.GetInfo().GetGoodType() {
	case mgrpb.GoodType_GoodTypeClassicMining:
	case mgrpb.GoodType_GoodTypeUnionMining:
	case mgrpb.GoodType_GoodTypeTechniqueFee:
	case mgrpb.GoodType_GoodTypeElectricityFee:
	default:
		logger.Sugar().Errorw("CreateGood", "GoodType", in.GetInfo().GetGoodType())
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "GoodType is invalid")
	}

	if in.GetInfo().GetTitle() == "" {
		logger.Sugar().Errorw("CreateGood", "Title", in.GetInfo().GetTitle())
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "Title is invalid")
	}

	if in.GetInfo().GetUnitAmount() <= 0 {
		logger.Sugar().Errorw("CreateGood", "UnitAmount", in.GetInfo().GetUnitAmount())
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "UnitAmount is invalid")
	}

	for _, coinTypeID := range in.GetInfo().GetSupportCoinTypeIDs() {
		if _, err := uuid.Parse(coinTypeID); err != nil {
			logger.Sugar().Errorw("CreateGood", "SupportCoinTypeIDs", in.GetInfo().GetSupportCoinTypeIDs(), "error", err)
			return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	now := uint32(time.Now().Unix())
	if in.GetInfo().GetDeliveryAt() <= now {
		logger.Sugar().Errorw("CreateGood", "DeliveryAt", in.GetInfo().GetDeliveryAt(), "now", now)
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "DeliveryAt is invalid")
	}

	if in.GetInfo().GetStartAt() <= now {
		logger.Sugar().Errorw("CreateGood", "StartAt", in.GetInfo().GetStartAt(), "now", now)
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "StartAt is invalid")
	}

	if in.GetInfo().GetTotal() <= 0 {
		logger.Sugar().Errorw("CreateGood", "Total", in.GetInfo().GetTotal())
		return &npool.CreateGoodResponse{}, status.Error(codes.InvalidArgument, "Total is invalid")
	}

	span = commontracer.TraceInvoker(span, "Good", "mw", "CreateGood")

	info, err := goodmwcli.CreateGood(ctx, in.GetInfo())
	if err != nil {
		logger.Sugar().Errorw("CreateGood", "Good", in.GetInfo(), "error", err)
		return &npool.CreateGoodResponse{}, status.Error(codes.Internal, err.Error())
	}

	coinType, err := coininfocli.GetCoinInfo(ctx, info.CoinTypeID)
	if err != nil {
		logger.Sugar().Errorw("CreateGood", "Good", in.GetInfo(), "error", err)
		return &npool.CreateGoodResponse{}, status.Error(codes.Internal, err.Error())
	}

	supportCoins := []*npool.Good_CoinInfo{}
	for _, val := range info.SupportCoinTypeIDs {
		coinTypeInfo, err := coininfocli.GetCoinInfo(ctx, val)
		if err != nil {
			logger.Sugar().Errorw("CreateGood", "Good", in.GetInfo(), "error", err)
			return &npool.CreateGoodResponse{}, status.Error(codes.Internal, err.Error())
		}
		supportCoins = append(supportCoins, &npool.Good_CoinInfo{
			CoinTypeID:  info.CoinTypeID,
			CoinLogo:    coinTypeInfo.Logo,
			CoinName:    coinTypeInfo.Name,
			CoinUnit:    coinTypeInfo.Unit,
			CoinPreSale: coinTypeInfo.PreSale,
		})
	}

	return &npool.CreateGoodResponse{
		Info: &npool.Good{
			ID:                         info.ID,
			DeviceInfoID:               info.DeviceInfoID,
			DeviceType:                 info.DeviceType,
			DeviceManufacturer:         info.DeviceManufacturer,
			DevicePowerComsuption:      info.DevicePowerComsuption,
			DeviceShipmentAt:           info.DeviceShipmentAt,
			DevicePosters:              info.DevicePosters,
			DurationDays:               info.DurationDays,
			CoinTypeID:                 info.CoinTypeID,
			CoinLogo:                   coinType.Logo,
			CoinName:                   coinType.Name,
			CoinUnit:                   coinType.Unit,
			CoinPreSale:                coinType.PreSale,
			InheritFromGoodID:          info.InheritFromGoodID,
			InheritFromGoodName:        info.InheritFromGoodName,
			InheritFromGoodType:        info.InheritFromGoodType,
			InheritFromGoodBenefitType: info.InheritFromGoodBenefitType,
			VendorLocationID:           info.VendorLocationID,
			VendorLocationCountry:      info.VendorLocationCountry,
			VendorLocationProvince:     info.VendorLocationProvince,
			VendorLocationCity:         info.VendorLocationCity,
			VendorLocationAddress:      info.VendorLocationAddress,
			GoodType:                   info.GoodType,
			BenefitType:                info.BenefitType,
			Price:                      info.Price,
			Title:                      info.Title,
			Unit:                       info.Unit,
			UnitAmount:                 info.UnitAmount,
			TestOnly:                   info.TestOnly,
			Posters:                    info.Posters,
			Labels:                     info.Labels,
			VoteCount:                  info.VoteCount,
			Rating:                     info.Rating,
			SupportCoins:               supportCoins,
			GoodStockID:                info.GoodStockID,
			GoodTotal:                  info.GoodTotal,
			GoodLocked:                 info.GoodLocked,
			GoodInService:              info.GoodInService,
			GoodSold:                   info.GoodSold,
			DeliveryAt:                 info.DeliveryAt,
			StartAt:                    info.StartAt,
			CreatedAt:                  info.CreatedAt,
			UpdatedAt:                  info.UpdatedAt,
		},
	}, nil
}
