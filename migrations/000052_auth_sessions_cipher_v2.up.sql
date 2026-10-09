-- Tokens passam a ser cifrados pelo retech-authkit/sessioncrypto (HKDF + AAD = id da sessão).
-- Linhas antigas não decifram mais: sessões ativas são encerradas e cada usuário faz login de novo uma vez.
DELETE FROM auth_sessions;
