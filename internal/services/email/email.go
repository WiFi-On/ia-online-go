package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Структура EmailService для хранения настроек SMTP
type EmailService struct {
	SMTPServer string
	SMTPPort   string
	Email      string
	Password   string
}

type EmailServiceI interface {
	SendEmail(ctx context.Context, toAddress, subject, body string) error
	SendActivationLink(ctx context.Context, toAddress string, activationLink string) error
	SendNewPassword(ctx context.Context, toAddress string, new_password string) error
}

// Конструктор для создания нового экземпляра EmailService
func New(smtpServer, smtpPort, email, password string) *EmailService {
	return &EmailService{
		SMTPServer: smtpServer,
		SMTPPort:   smtpPort,
		Email:      email,
		Password:   password,
	}
}

// Функция для отправки письма
func (e *EmailService) SendEmail(ctx context.Context, toAddress, subject, body string) error {
	op := "EmailService.SendEmail"

	serverAddr := e.SMTPServer + ":" + e.SMTPPort

	// --- 1. Пробуем установить соединение ---
	var client *smtp.Client
	var err error

	switch e.SMTPPort {
	case "465": // SSL (чистый TLS)
		// Подключение по TLS
		tlsConfig := &tls.Config{
			ServerName:         e.SMTPServer,
			InsecureSkipVerify: false, // true — только если внутренний сервер с самоподписанным сертификатом
		}

		conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
		if err != nil {
			return fmt.Errorf("%s: ошибка TLS-подключения (465): %w", op, err)
		}

		client, err = smtp.NewClient(conn, e.SMTPServer)
		if err != nil {
			return fmt.Errorf("%s: ошибка создания SMTP клиента: %w", op, err)
		}

	default: // 587 и прочие
		// Подключение без TLS, потом STARTTLS
		client, err = smtp.Dial(serverAddr)
		if err != nil {
			return fmt.Errorf("%s: ошибка подключения (587): %w", op, err)
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName:         e.SMTPServer,
				InsecureSkipVerify: false,
			}
			if err = client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("%s: ошибка запуска STARTTLS: %w", op, err)
			}
		}
	}
	defer func() { _ = client.Quit() }()

	// --- 2. Аутентификация ---
	auth := smtp.PlainAuth("", e.Email, e.Password, e.SMTPServer)
	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%s: ошибка аутентификации SMTP: %w", op, err)
		}
	}

	// --- 3. Отправка письма ---
	if err := client.Mail(e.Email); err != nil {
		return fmt.Errorf("%s: ошибка MAIL FROM: %w", op, err)
	}

	if err := client.Rcpt(toAddress); err != nil {
		return fmt.Errorf("%s: ошибка RCPT TO: %w", op, err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("%s: ошибка открытия потока для письма: %w", op, err)
	}

	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n%s",
		e.Email, toAddress, subject, body,
	)

	if _, err = w.Write([]byte(message)); err != nil {
		return fmt.Errorf("%s: ошибка записи письма: %w", op, err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("%s: ошибка закрытия потока письма: %w", op, err)
	}

	return nil
}

func (e *EmailService) SendActivationLink(ctx context.Context, toAddress string, activationLink string) error {
	op := "EmailService.SendActivationLink"

	htmlBody := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Подтверждение регистрации</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f0f0f0;
            color: #333;
            padding: 0;
            margin: 0;
        }
        .container {
            max-width: 600px;
            margin: 40px auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
        }
        .header {
            background-color: #7ed956;
            padding: 20px;
            text-align: center;
            color: white;
            font-size: 24px;
        }
        .content {
            display: flex;
            align-items: center;
            flex-direction: column;
            padding: 30px;
        }
        .button {
            display: inline-block;
            margin-top: 20px;
            padding: 12px 24px;
            background-color: #7ed956;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            font-weight: bold;
        }
        .footer {
            margin-top: 40px;
            font-size: 12px;
            color: #999;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">Добро пожаловать!</div>
        <div class="content">
            <h2>Спасибо за регистрацию 🎉</h2>
            <p>Чтобы активировать ваш аккаунт, нажмите на кнопку ниже:</p>
            <a class="button" href="{{.ActivationLink}}">Подтвердить аккаунт</a>
            <p class="footer">Если вы не регистрировались, просто проигнорируйте это письмо.</p>
        </div>
    </div>
</body>
</html>
`
	htmlBody = strings.Replace(htmlBody, "{{.ActivationLink}}", activationLink, -1)

	err := e.SendEmail(ctx, toAddress, "Подтверждение регестрации", htmlBody)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (e *EmailService) SendNewPassword(ctx context.Context, toAddress string, new_password string) error {
	op := "EmailService.SendPasswordCode"

	htmlBody := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Новый пароль</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f0f0f0;
            color: #333;
            padding: 0;
            margin: 0;
        }
        .container {
            max-width: 600px;
            margin: 40px auto;
            background-color: #ffffff;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
        }
        .header {
            background-color: #7ed956;
            padding: 20px;
            text-align: center;
            color: white;
            font-size: 24px;
        }
        .content {
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 30px;
        }
        .password-box {
            margin-top: 20px;
            background-color: #f0f0f0;
            padding: 15px;
            font-size: 18px;
            font-weight: bold;
            border-radius: 6px;
            word-break: break-all;
            text-align: center;
        }
        .footer {
            margin-top: 40px;
            font-size: 12px;
            color: #999;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">Временный пароль</div>
        <div class="content">
            <h2>Ваш новый временный пароль:</h2>
            <div class="password-box">{{.NewPassword}}</div>
            <p class="footer">Рекомендуем сменить его сразу после входа.</p>
        </div>
    </div>
</body>
</html>
`
	htmlBody = strings.Replace(htmlBody, "{{.NewPassword}}", new_password, -1)

	err := e.SendEmail(ctx, toAddress, "Новый временный пароль", htmlBody)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
