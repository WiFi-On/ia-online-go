package utils

import (
	"crypto/rand"
	"fmt"
	"ia-online-golang/internal/dto"
	"ia-online-golang/internal/http/responses"
	"ia-online-golang/internal/models"
	"math/big"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

func FormatValidationErrors(err error) string {
	var messages []string
	for _, e := range err.(validator.ValidationErrors) {
		messages = append(messages, fmt.Sprintf("Field '%s' is invalid", e.Field()))
	}
	return strings.Join(messages, ", ")
}

func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	responses.SendError(w, http.StatusNotFound, "По таким путям мы не работаем")
}

func UserToDTO(user models.User) dto.UserDTO {
	return dto.UserDTO{
		ID:           &user.ID,
		Roles:        user.Roles,
		ReferralCode: user.ReferralCode,
		Email:        user.Email,
		Name:         user.Name,
		PhoneNumber:  user.PhoneNumber,
		City:         user.City,
		Telegram:     user.Telegram,
	}
}

func DtoToUser(user dto.UserDTO) models.User {
	return models.User{
		ID:           derefInt64(user.ID),
		Roles:        user.Roles,
		ReferralCode: user.ReferralCode,
		Email:        user.Email,
		Name:         user.Name,
		PhoneNumber:  user.PhoneNumber,
		City:         user.City,
		Telegram:     user.Telegram,
	}
}

func UserToReferralDTO(user models.User, status bool) dto.ReferralDTO {
	return dto.ReferralDTO{
		ID:          user.ID,
		Name:        user.Name,
		PhoneNumber: user.PhoneNumber,
		City:        user.City,
		Active:      status,
	}
}

func GeneratePasswordCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

func Contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func derefInt64(i *int64) int64 {
	if i != nil {
		return *i
	}
	return 0
}

func GenerateValidPassword(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8 characters")
	}

	digits := "0123456789"
	uppers := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specials := "!@#$%^&*()_+-=[]{};:'\",.<>/?\\|"
	all := digits + uppers + specials + "abcdefghijklmnopqrstuvwxyz"

	// Гарантируем хотя бы по одному символу каждого типа
	password := make([]byte, length)
	var err error

	password[0], err = randomChar(digits)
	if err != nil {
		return "", err
	}
	password[1], err = randomChar(uppers)
	if err != nil {
		return "", err
	}
	password[2], err = randomChar(specials)
	if err != nil {
		return "", err
	}

	// Остальные символы
	for i := 3; i < length; i++ {
		password[i], err = randomChar(all)
		if err != nil {
			return "", err
		}
	}

	// Перемешиваем пароль
	shuffle(password)

	return string(password), nil
}

func randomChar(charset string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, err
	}
	return charset[n.Int64()], nil
}

func shuffle(data []byte) {
	for i := range data {
		jBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(data))))
		j := int(jBig.Int64())
		data[i], data[j] = data[j], data[i]
	}
}
