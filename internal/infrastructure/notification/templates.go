package notification

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
)

// Identidade visual do MeuFin (espelho do tema do admin: colorTemplate.ts).
const (
	brandNeon     = "#00e676" // primary.main
	brandNeonDark = "#00c853" // primary.dark
	brandBgApp    = "#050506" // background.app
	brandBgCard   = "#0c0e12" // background.elevated
	brandBorder   = "#1c1f26"
	brandText     = "#e4e4e7"
	brandTextDim  = "#8b8f98"
)

// baseEmailTmpl é o layout base de todos os e-mails do MeuFin: fundo escuro,
// card central, faixa neon no topo e botão de ação. Tudo inline-style —
// clientes de e-mail não carregam CSS externo.
var baseEmailTmpl = template.Must(template.New("base").Parse(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="color-scheme" content="dark">
  <meta name="supported-color-schemes" content="dark">
  <title>{{.Subject}}</title>
</head>
<body style="margin:0;padding:0;background-color:` + brandBgApp + `;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;">{{.Preheader}}</div>
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:` + brandBgApp + `;padding:40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;">
          <!-- Logo -->
          <tr>
            <td align="center" style="padding-bottom:28px;">
              <table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 auto;">
                <tr>
                  {{if .LogoURL}}
                  <td style="vertical-align:middle;padding-right:12px;">
                    <img src="{{.LogoURL}}" width="44" height="44" alt="MeuFin" style="display:block;border:0;border-radius:12px;">
                  </td>
                  {{end}}
                  <td style="vertical-align:middle;text-align:left;">
                    <span style="font-size:26px;font-weight:800;letter-spacing:-0.5px;color:` + brandText + `;">meu<span style="color:` + brandNeon + `;">fin</span></span>
                    <div style="font-size:11px;color:` + brandTextDim + `;letter-spacing:2px;text-transform:uppercase;margin-top:2px;">gestão familiar completa</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <!-- Card -->
          <tr>
            <td style="background-color:` + brandBgCard + `;border:1px solid ` + brandBorder + `;border-radius:16px;overflow:hidden;">
              <!-- Faixa neon -->
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0">
                <tr><td style="height:4px;background:linear-gradient(90deg,` + brandNeonDark + `,` + brandNeon + `);font-size:0;line-height:0;">&nbsp;</td></tr>
              </table>
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0">
                <tr>
                  <td style="padding:36px 36px 32px 36px;">
                    <h1 style="margin:0 0 8px 0;font-size:22px;line-height:1.3;color:` + brandText + `;">{{.Title}}</h1>
                    <p style="margin:0 0 20px 0;font-size:15px;line-height:1.6;color:` + brandTextDim + `;">Olá, <strong style="color:` + brandText + `;">{{.Name}}</strong> 👋</p>
                    {{range .Paragraphs}}
                    <p style="margin:0 0 16px 0;font-size:15px;line-height:1.6;color:` + brandTextDim + `;">{{.}}</p>
                    {{end}}
                    {{if .ButtonURL}}
                    <table role="presentation" cellpadding="0" cellspacing="0" style="margin:28px auto 8px auto;">
                      <tr>
                        <td align="center" style="border-radius:12px;background:linear-gradient(180deg,` + brandNeon + `,` + brandNeonDark + `);box-shadow:0 0 24px rgba(0,230,118,0.35);">
                          <a href="{{.ButtonURL}}" target="_blank" style="display:inline-block;padding:14px 36px;font-size:15px;font-weight:700;color:#000000;text-decoration:none;border-radius:12px;">{{.ButtonLabel}}</a>
                        </td>
                      </tr>
                    </table>
                    <p style="margin:20px 0 0 0;font-size:12px;line-height:1.6;color:` + brandTextDim + `;text-align:center;">Se o botão não funcionar, copie e cole este link no navegador:<br>
                      <a href="{{.ButtonURL}}" style="color:` + brandNeon + `;word-break:break-all;">{{.ButtonURL}}</a>
                    </p>
                    {{end}}
                    {{if .FooterNote}}
                    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-top:28px;">
                      <tr>
                        <td style="border-top:1px solid ` + brandBorder + `;padding-top:18px;font-size:12px;line-height:1.6;color:` + brandTextDim + `;">{{.FooterNote}}</td>
                      </tr>
                    </table>
                    {{end}}
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <!-- Rodapé -->
          <tr>
            <td align="center" style="padding-top:24px;">
              <p style="margin:0;font-size:11px;line-height:1.6;color:` + brandTextDim + `;">Você recebeu este e-mail porque uma ação foi solicitada na sua conta MeuFin.<br>© MeuFin · The Retech — todos os direitos reservados.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

// emailData alimenta o layout base — reutilizável por qualquer notificação.
type emailData struct {
	Subject     string
	Preheader   string
	Title       string
	Name        string
	Paragraphs  []string
	ButtonURL   string
	ButtonLabel string
	FooterNote  template.HTML
	LogoURL     string
}

// logoURL lê MAIL_LOGO_URL — imagem pública do ícone da marca (PNG hospedado
// no admin: {ADMIN_BASE_URL}/logo-email.png). Vazio = header só com wordmark.
func logoURL() string {
	return strings.TrimSpace(os.Getenv("MAIL_LOGO_URL"))
}

func renderBase(data emailData) (string, error) {
	if data.LogoURL == "" {
		data.LogoURL = logoURL()
	}
	var buf bytes.Buffer
	if err := baseEmailTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("notification: renderizar template: %w", err)
	}
	return buf.String(), nil
}

// PasswordResetEmail monta o e-mail de redefinição de senha.
func PasswordResetEmail(name, resetURL string, ttlMinutes int) (Email, error) {
	if name == "" {
		name = "tudo bem?"
	}
	html, err := renderBase(emailData{
		Subject:   "Redefinição de senha — MeuFin",
		Preheader: "Recebemos um pedido para redefinir a sua senha. O link expira em breve.",
		Title:     "Redefinir a sua senha",
		Name:      name,
		Paragraphs: []string{
			"Recebemos um pedido para redefinir a senha da sua conta no MeuFin.",
			fmt.Sprintf("Clique no botão abaixo para escolher uma nova senha. Por segurança, o link expira em %d minutos e só pode ser usado uma vez.", ttlMinutes),
		},
		ButtonURL:   resetURL,
		ButtonLabel: "Redefinir senha",
		FooterNote:  template.HTML("Se você <strong>não</strong> pediu esta redefinição, ignore este e-mail — a sua senha continua a mesma e nenhuma ação é necessária."),
	})
	if err != nil {
		return Email{}, err
	}
	text := fmt.Sprintf(
		"Olá, %s\n\nRecebemos um pedido para redefinir a senha da sua conta no MeuFin.\n"+
			"Acesse o link abaixo para escolher uma nova senha (expira em %d minutos, uso único):\n\n%s\n\n"+
			"Se você não pediu esta redefinição, ignore este e-mail.\n\n— MeuFin",
		name, ttlMinutes, resetURL)
	return Email{
		Subject: "Redefinição de senha — MeuFin",
		HTML:    html,
		Text:    text,
	}, nil
}
