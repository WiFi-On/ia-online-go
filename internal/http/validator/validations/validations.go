package validations

import (
	"ia-online-golang/internal/dto"
	"regexp"

	"github.com/go-playground/validator/v10"
	_ "github.com/go-playground/validator/v10/translations/ru"
)

func PasswordValidation(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Проверка на минимум 1 цифру
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	// Проверка на минимум 1 заглавную букву
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// Проверка на минимум 1 спецсимвол
	hasSpecial := regexp.MustCompile(`[!@#\$%\^&\*\(\)_\+\-=\[\]\{\};:'",<>\./?\\|]`).MatchString(password)

	// Проверка на минимум 8 символов
	return len(password) >= 8 && hasDigit && hasUpper && hasSpecial
}

func AtLeastOneServiceEnabled(fl validator.FieldLevel) bool {
	obj := fl.Parent().Interface().(dto.CreateLeadDTO)
	return obj.IsInternet || obj.IsShipping || obj.IsCleaning
}

func NewPasswordStructValidation(sl validator.StructLevel) {
	dto := sl.Current().Interface().(dto.NewPasswordDTO)

	if dto.OldPassword == dto.NewPassword {
		// Регистрируем ошибку на поле NewPassword, можно изменить сообщение в переводах
		sl.ReportError(dto.NewPassword, "NewPassword", "new_password", "nefield", "OldPassword")
	}
}
