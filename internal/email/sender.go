package email

import (
	gomail "github.com/wneessen/go-mail"
)

type Sender struct {
	host string
	port int
	user string
	pass string
	from string
}

func New(host, user, pass, from string, port int) *Sender {
	return &Sender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *Sender) Send(to, subject, body string) error {
	m := gomail.NewMsg()
	if err := m.From(s.from); err != nil {
		return err
	}
	if err := m.To(to); err != nil {
		return err
	}
	m.Subject(subject)
	m.SetBodyString(gomail.TypeTextPlain, body)

	opts := []gomail.Option{
		gomail.WithPort(s.port),
		gomail.WithTLSPolicy(gomail.NoTLS),
	}
	if s.user != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(s.user),
			gomail.WithPassword(s.pass),
		)
	}

	c, err := gomail.NewClient(s.host, opts...)
	if err != nil {
		return err
	}
	return c.DialAndSend(m)
}
