// Package session: contratos da sessão do gateway. A implementação (cookie
// opaco, tokens cifrados, refresh, JWT verificado a cada request, logout no
// auth) é o retech-authkit/session — idêntica à do CashFlowfy.
package session

import (
	"github.com/theretechlabs/retech-authkit/authclient"
	"github.com/theretechlabs/retech-authkit/session"
)

// Session é a sessão do browser (ID = sha256 do token do cookie).
type Session = session.Session

// Tokens são os tokens do auth central guardados na sessão.
type Tokens = session.Tokens

// UserInfo é o usuário autenticado pelo auth central.
type UserInfo = session.User

// Authenticator é o auth central (ou um fake nos testes).
type Authenticator = session.Authenticator

// Store persiste sessões.
type Store = session.Store

// Erros (do kit), mapeados pelo HTTP para o formato de erro do MeuFin.
var (
	ErrInvalidCredentials = authclient.ErrInvalidCredentials
	ErrUserInactive       = authclient.ErrInactive
	ErrAuthUnavailable    = authclient.ErrUnavailable
	ErrRefreshRejected    = authclient.ErrRefreshRejected
	ErrNotFound           = session.ErrNotFound
	ErrTokenInvalid       = session.ErrTokenInvalid
)
