package mails

import (
	"context"
	"fmt"
	"time"

	"github.com/resend/resend-go/v2"
)

const (
	LoginMailType                   = "login"
	SignUpMailType                  = "SignUp"
	LowBalanceMailType              = "low_balance"
	OutOfBalanceMailType            = "out_of_balance"
	MonthlyReceiptType              = "monthly_receipt"
	RequestJoinExclusiveProgramType = "request_join_exlusive_program"
)

type ResendHelper struct {
	resendClient     *resend.Client
	domain           string
	emailAddress     string
	emailName        string
	templateDir      string
	emailFeedback    string
	termOfServiceUrl string
	githubUrl        string
	telegramUrl      string
	twitterUrl       string
}

func NewResendHelper(apiKey, from, templateDir string) MailHelper {
	client := resend.NewClient(apiKey)
	return &ResendHelper{
		resendClient:     client,
		domain:           from,
		templateDir:      templateDir,
		emailFeedback:    "admin@aioz.tube",
		termOfServiceUrl: "https://aioz.network/",
		githubUrl:        "https://github.com/aioznetwork",
		telegramUrl:      "https://t.me/aiozofficial",
		twitterUrl:       "https://twitter.com/AIOZNetwork",
	}
}

func (r ResendHelper) SendEmail(
	ctx context.Context,
	userEmail []string,
	mailType string,
	data map[string]any,
) error {
	var (
		emailTitle, emailBody string
		err                   error
	)
	defaultData := map[string]any{
		"Days":             "7",
		"EmailFeedback":    r.emailFeedback,
		"TermOfServiceUrl": r.termOfServiceUrl,
		"TelegramUrl":      r.telegramUrl,
		"TwitterUrl":       r.twitterUrl,
		"GithubUrl":        r.githubUrl,
		"CopyrightYear":    time.Now().Year(),
	}
	for k, v := range defaultData {
		data[k] = v
	}
	switch mailType {
	case LoginMailType:
		{
			emailTitle = "Login"
			data["Subject"] = "Login"
			emailBody, err = generateEmailBody(
				data,
				fmt.Sprintf(
					"%s/login_code.html",
					r.templateDir,
				),
			)
			if err != nil {
				return err
			}
		}
	case SignUpMailType:
		{
			emailTitle = "Sign up"
			data["Subject"] = "Sign up"
			emailBody, err = generateEmailBody(
				data,
				fmt.Sprintf(
					"%s/signup_code.html",
					r.templateDir,
				),
			)
			if err != nil {
				return err
			}
		}
	case LowBalanceMailType:
		{
			emailTitle = "Low wallet's balance"
			data["Subject"] = "Low balance"
			emailBody, err = generateEmailBody(
				data,
				fmt.Sprintf(
					"%s/low_wallet_balance.html",
					r.templateDir,
				),
			)
			if err != nil {
				return err
			}
		}
	case OutOfBalanceMailType:
		{
			emailTitle = "Out of balance"
			data["Subject"] = "Out of balance"
			emailBody, err = generateEmailBody(
				data,
				fmt.Sprintf(
					"%s/out_of_balance.html",
					r.templateDir,
				),
			)
			if err != nil {
				return err
			}
		}
	case MonthlyReceiptType:
		{
			emailTitle = "Monthly receipt"
			data["Subject"] = "Monthly receipt"
			emailBody, err = generateEmailBody(
				data,
				fmt.Sprintf(
					"%s/monthly_receipt.html",
					r.templateDir,
				),
			)
			if err != nil {
				return err
			}
		}
	case RequestJoinExclusiveProgramType:
		{
			emailTitle = "Request join exclusive program"
			data["Subject"] = "Monthly receipt"
			emailBody, err = generateEmailBody(
				data,
				fmt.Sprintf(
					"%s/request_join_exclusive_program.html",
					r.templateDir,
				),
			)
			if err != nil {
				return err
			}
		}
	default:
		{
			return fmt.Errorf(
				"mail type %s not supported",
				mailType,
			)
		}
	}
	params := &resend.SendEmailRequest{
		From: fmt.Sprintf(
			"AIOZ STREAM <%s>", r.domain,
		),
		To:      userEmail,
		Subject: emailTitle,
		Html:    emailBody,
	}

	timeoutCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	if _, err := r.resendClient.Emails.SendWithContext(
		timeoutCtx,
		params,
	); err != nil {
		return err
	}
	return nil
}
