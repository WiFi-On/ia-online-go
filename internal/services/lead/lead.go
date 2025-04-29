package lead

import (
	"context"
	"errors"
	"fmt"
	"ia-online-golang/internal/dto"
	"ia-online-golang/internal/http/context_keys"
	"ia-online-golang/internal/models"
	"ia-online-golang/internal/services/bitrix"
	"ia-online-golang/internal/services/comment"
	"ia-online-golang/internal/services/user"
	"ia-online-golang/internal/storage"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LeadService struct {
	log                *logrus.Logger
	CommentService     comment.CommentServiceI
	UserService        user.UserServiceI
	BitrixService      bitrix.BitrixServiceI
	LeadRepository     storage.LeadRepositoryI
	ReferralRepository storage.ReferralRepositoryI
}

type LeadServiceI interface {
	Leads(ctx context.Context, filterDTO dto.LeadFilterDTO) ([]dto.LeadDTO, error)
	GetUserPaymentStatistic(ctx context.Context, userID int64, startDate *time.Time, endDate *time.Time) (dto.UserStatistic, error)
	SaveLead(ctx context.Context, lead dto.CreateLeadDTO) error
	EditDeal(ctx context.Context, arrInfoBitrix []string) error
}

func New(
	log *logrus.Logger,
	commentService comment.CommentServiceI,
	leadRepository storage.LeadRepositoryI,
	userService user.UserServiceI,
	referralRepository storage.ReferralRepositoryI,
	bitrixService bitrix.BitrixServiceI,
) *LeadService {
	return &LeadService{
		log:                log,
		CommentService:     commentService,
		LeadRepository:     leadRepository,
		UserService:        userService,
		ReferralRepository: referralRepository,
		BitrixService:      bitrixService,
	}
}

func (l *LeadService) Leads(ctx context.Context, filterDTO dto.LeadFilterDTO) ([]dto.LeadDTO, error) {
	const op = "LeadService.Leads"

	if filterDTO.UserID == nil {
		userIDValue := ctx.Value(context_keys.UserIDKey)
		userID, ok := userIDValue.(int64)
		if !ok {
			return []dto.LeadDTO{}, fmt.Errorf("%s: error receiving userID", op)
		}
		filterDTO.UserID = &userID
	}

	leads, err := l.LeadRepository.Leads(
		ctx,
		filterDTO.StatusID,
		filterDTO.StartDate,
		filterDTO.EndDate,
		filterDTO.Limit,
		filterDTO.Offset,
		filterDTO.UserID,
		filterDTO.Search,
		filterDTO.IsInternet,
		filterDTO.IsShipping,
		filterDTO.IsCleaning,
	)
	if err != nil {
		if errors.Is(err, storage.ErrLeadsNotFound) {
			return []dto.LeadDTO{}, nil
		}

		l.log.Error("Error fetching leads", err)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var result []dto.LeadDTO

	for _, lead := range leads {
		leadResult := dto.LeadDTO{
			ID:             lead.ID,
			FIO:            lead.FIO,
			Address:        lead.Address,
			StatusID:       lead.StatusID,
			PhoneNumber:    lead.PhoneNumber,
			Internet:       lead.Internet,
			Cleaning:       lead.Cleaning,
			Shipping:       lead.Shipping,
			CreatedAt:      lead.CreatedAt,
			CompletedAt:    lead.CompletedAt,
			PaymentAt:      lead.PaymentAt,
			RewardInternet: lead.RewardInternet,
			RewardCleaning: lead.RewardCleaning,
			RewardShipping: lead.RewardShipping,
		}

		comments, err := l.CommentService.Comments(ctx, lead.ID)
		if err != nil {
			if errors.Is(err, comment.ErrCommentsNotFound) {
				leadResult.Comments = []dto.CommentDTO{} // Пустой список, если нет комментариев
			} else {
				return nil, fmt.Errorf("%s: %w", op, err)
			}
		} else {
			leadResult.Comments = comments
		}

		result = append(result, leadResult)
	}

	return result, nil
}

func (l *LeadService) SaveLead(ctx context.Context, lead dto.CreateLeadDTO) error {
	const op = "LeadService.SaveLead"

	userIDValue := ctx.Value(context_keys.UserIDKey)
	userID, ok := userIDValue.(int64)
	if !ok {
		return fmt.Errorf("%s: %v", op, "user id not found")
	}

	user, err := l.UserService.UserById(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	bitrix_result, err := l.BitrixService.SendDeal(ctx, lead, user)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	leadDB := models.Lead{
		ID:             int64(bitrix_result.Result),
		UserID:         userID,
		FIO:            lead.Name,
		Address:        lead.Address,
		StatusID:       0,
		PhoneNumber:    lead.PhoneNumber,
		Internet:       lead.IsInternet,
		Cleaning:       lead.IsCleaning,
		Shipping:       lead.IsShipping,
		RewardInternet: lead.RewardInternet,
		RewardCleaning: lead.RewardCleaning,
		RewardShipping: lead.RewardShipping,
	}

	err = l.LeadRepository.CreateLead(ctx, &leadDB)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	if lead.Comment != "" {
		_, err = l.CommentService.SaveComment(ctx, int64(bitrix_result.Result), lead.Comment)
		if err != nil {
			return fmt.Errorf("%s: %v", op, err)
		}
	}

	return nil
}

func (l *LeadService) EditDeal(ctx context.Context, arrInfoBitrix []string) error {
	const op = "LeadService.EditDeal"

	idDealStr := arrInfoBitrix[2]
	idStr := strings.Split(idDealStr, "_")[1]
	idDeal, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	infoDeal, err := l.BitrixService.GetLead(ctx, idDeal)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	statuses := map[string]int64{
		"C42:NEW":               0,
		"C42:PREPARATION":       1,
		"C42:PREPAYMENT_INVOIC": 2,
		"C42:EXECUTING":         7,
		"C42:FINAL_INVOICE":     3,
		"C42:1":                 4,
		"C42:LOSE":              6,
		"C42:WON":               5,
	}

	status, ok := statuses[infoDeal.Result.Status]
	if !ok {
		return fmt.Errorf("%s: статус не найден для %s", op, infoDeal.Result.Status)
	}

	var paymentAt *time.Time
	var completedAt *time.Time
	if status == 4 {
		now := time.Now()
		completedAt = &now
	}
	if status == 5 {
		now := time.Now()
		paymentAt = &now
	}

	// Преобразуем строки в float64
	internetPayment, err := strconv.ParseFloat(infoDeal.Result.InternetPayment, 64)
	if err != nil {
		internetPayment = 0
	}

	cleaningPayment, err := strconv.ParseFloat(infoDeal.Result.CleaningPayment, 64)
	if err != nil {
		cleaningPayment = 0
	}

	shippingPayment, err := strconv.ParseFloat(infoDeal.Result.ShippingPayment, 64)
	if err != nil {
		shippingPayment = 0
	}

	l.LeadRepository.UpdateLead(
		ctx,
		&idDeal,
		nil,
		&status,
		&internetPayment,
		&cleaningPayment,
		&shippingPayment,
		nil, nil, nil, nil, nil, nil, nil, completedAt, paymentAt,
	)

	return nil
}

func (l *LeadService) GetUserPaymentStatistic(ctx context.Context, userID int64, startDate *time.Time, endDate *time.Time) (dto.UserStatistic, error) {
	const op = "LeadService.GetUserPaymentStatistic"

	leads, err := l.LeadRepository.Leads(ctx, nil, startDate, endDate, 0, 0, &userID, nil, nil, nil, nil)
	if err != nil {
		if errors.Is(err, storage.ErrLeadsNotFound) {

		} else {
			return dto.UserStatistic{}, fmt.Errorf("%s: %v", op, err)
		}
	}

	user, err := l.UserService.UserById(ctx, userID)
	if err != nil {
		return dto.UserStatistic{}, fmt.Errorf("%s: %v", op, err)
	}

	result := dto.UserStatistic{}

	referrals, err := l.ReferralRepository.ActiveReferralsByReferralId(ctx, user.ReferralCode)
	if err != nil {
		if errors.Is(err, storage.ErrReferralsNotFound) {

		} else {
			return dto.UserStatistic{}, fmt.Errorf("%s: %v", op, err)
		}
	}

	for _, lead := range leads {
		result.Internet += lead.RewardInternet
		result.Shipping += lead.RewardShipping
		result.Cleaning += lead.RewardCleaning
	}

	for _, referral := range referrals {
		result.Referrals += referral.Cost
	}

	result.Total = result.Cleaning + result.Referrals + result.Internet + result.Shipping

	return result, nil
}
