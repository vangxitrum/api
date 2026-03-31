package mails

import (
	"bytes"
	"context"
	"html/template"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
)

type MailHelper interface {
	SendEmail(
		ctx context.Context,
		userEmail []string,
		mailType string,
		data map[string]interface{},
	) error
}

func FormatEmail(email string) (string, error) {
	rs, err := mail.ParseAddress(email)
	if err != nil {
		return "", err
	}

	// abc...@xyz -> abc, xyz
	i := strings.LastIndexByte(rs.Address, '@')
	// If no @ present, not a valid email.
	beforeEmail := rs.Address[:i]
	afterEmail := rs.Address[i+1:]

	// lower email
	beforeEmail = strings.ToLower(beforeEmail)

	//	@googlemail.com	-> @gmail.com
	if afterEmail == "googlemail.com" {
		afterEmail = "gmail.com"
	}

	if strings.HasSuffix(afterEmail, ".com") {
		beforeEmail = strings.ReplaceAll(
			beforeEmail,
			".",
			"",
		)
	}

	// remove email tag
	// abc+cde@xyz -> abc@xyz
	beforeEmail = strings.Split(
		beforeEmail,
		"+",
	)[0]

	return strings.Join(
		[]string{
			beforeEmail,
			afterEmail,
		}, "@",
	), nil
}

func parseTemplateDir(dir string) (
	*template.Template, error,
) {
	var paths []string
	err := filepath.Walk(
		dir,
		func(
			path string, info os.FileInfo,
			err error,
		) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				paths = append(paths, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return template.ParseFiles(paths...)
}

func generateEmailBody(
	data map[string]any, templateDir string,
) (string, error) {
	template, err := parseTemplateDir(templateDir)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	if err := template.Execute(
		&body,
		data,
	); err != nil {
		return "", err
	}

	return body.String(), nil
}
