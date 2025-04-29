package comment

import (
	"context"
	"errors"
	"fmt"
	"ia-online-golang/internal/dto"
	"ia-online-golang/internal/http/context_keys"
	"ia-online-golang/internal/models"
	"ia-online-golang/internal/services/bitrix"
	"ia-online-golang/internal/storage"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Структура EmailService для хранения настроек SMTP
type CommentService struct {
	log               *logrus.Logger
	id_funnel         string
	BitrixService     bitrix.BitrixServiceI
	LeadRepository    storage.LeadRepositoryI
	CommentRepository storage.CommentsRepositoryI
}

type CommentServiceI interface {
	SaveComment(ctx context.Context, id_lead int64, text string) (dto.CommentDTO, error)
	SaveCommentFromBitrix(ctx context.Context, id_comment int64) error
	Comments(ctx context.Context, leadID int64) ([]dto.CommentDTO, error)
}

var (
	ErrLeadNotFound                    = errors.New("lead not found")
	ErrLeadDoesNotBelongToUser         = errors.New("lead does not belong to user")
	ErrCommentDoesNotBelongToTheFunnel = errors.New("comment does not belong to the funnel")
	ErrCommentsNotFound                = errors.New("comments not found")
)

// Конструктор для создания нового экземпляра EmailService
func New(log *logrus.Logger, id_funnel string, bitrixService bitrix.BitrixServiceI, leadRepository storage.LeadRepositoryI, commentRepository storage.CommentsRepositoryI) *CommentService {
	return &CommentService{
		log:               log,
		id_funnel:         id_funnel,
		BitrixService:     bitrixService,
		LeadRepository:    leadRepository,
		CommentRepository: commentRepository,
	}
}

func (c *CommentService) SaveComment(ctx context.Context, id_lead int64, text string) (dto.CommentDTO, error) {
	const op = "CommentService.SaveComment"

	userIDValue := ctx.Value(context_keys.UserIDKey)
	userID, ok := userIDValue.(int64)
	if !ok {
		return dto.CommentDTO{}, fmt.Errorf("%s: %v", op, "user id not found")
	}

	lead, err := c.LeadRepository.LeadByID(ctx, id_lead)
	if err != nil {
		if errors.Is(err, storage.ErrLeadNotFound) {
			return dto.CommentDTO{}, ErrLeadNotFound
		}

		return dto.CommentDTO{}, fmt.Errorf("%s: %v", op, err)
	}

	if userID != lead.UserID {
		return dto.CommentDTO{}, ErrLeadDoesNotBelongToUser
	}

	commentBitrix, err := c.BitrixService.SendComment(ctx, id_lead, text)
	if err != nil {
		return dto.CommentDTO{}, fmt.Errorf("%s: %v", op, err)
	}

	commentObj := models.Comment{
		ID:     int64(commentBitrix.Result),
		LeadID: id_lead,
		UserID: userID,
		Text:   text,
	}

	comment, err := c.CommentRepository.SaveComment(ctx, commentObj)
	if err != nil {
		return dto.CommentDTO{}, fmt.Errorf("%s: %v", op, err)
	}

	var result dto.CommentDTO
	if comment.UserID == 228 {
		result = dto.CommentDTO{
			ID:        comment.ID,
			Manager:   true,
			Text:      comment.Text,
			CreatedAt: comment.CreatedAt,
		}
	} else {
		result = dto.CommentDTO{
			ID:        comment.ID,
			Manager:   false,
			Text:      comment.Text,
			CreatedAt: comment.CreatedAt,
		}
	}

	return result, nil
}

func (c *CommentService) SaveCommentFromBitrix(ctx context.Context, id_comment int64) error {
	const op = "CommentService.SaveCommentFromBitrix"

	comment, err := c.BitrixService.GetComment(ctx, id_comment)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	entityID, err := strconv.ParseInt(comment.Result.EntityID, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: invalid EntityID: %v", op, err)
	}

	lead, err := c.BitrixService.GetLead(ctx, entityID)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	if lead.Result.CategoryID != c.id_funnel {
		return ErrCommentDoesNotBelongToTheFunnel
	}

	leadID, err := strconv.ParseInt(lead.Result.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: invalid LeadID: %v", op, err)
	}

	commentObj := models.Comment{
		ID:     id_comment,
		LeadID: leadID,
		UserID: 228,
		Text:   comment.Result.Comment,
	}

	if _, err := c.CommentRepository.SaveComment(ctx, commentObj); err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	return nil
}

func (c *CommentService) Comments(ctx context.Context, leadID int64) ([]dto.CommentDTO, error) {
	const op = "CommentService.Comments"

	comments, err := c.CommentRepository.Comments(ctx, leadID)
	if err != nil {
		if errors.Is(err, storage.ErrCommentsNotFound) {
			return []dto.CommentDTO{}, ErrCommentsNotFound
		}

		return []dto.CommentDTO{}, fmt.Errorf("%s: %v", op, err)
	}

	var result []dto.CommentDTO
	for _, comment := range comments {
		if comment.UserID == 228 {
			result = append(result, dto.CommentDTO{
				ID:        comment.ID,
				Manager:   true,
				Text:      comment.Text,
				CreatedAt: comment.CreatedAt,
			})
		} else {
			result = append(result, dto.CommentDTO{
				ID:        comment.ID,
				Manager:   false,
				Text:      comment.Text,
				CreatedAt: comment.CreatedAt,
			})
		}
	}

	return result, nil
}
