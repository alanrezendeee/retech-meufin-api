// Package qrdecode lê o QR Code de imagens de cupom fiscal no servidor.
// Complementa a leitura client-side (navegador): serve para canais sem browser
// no meio (ex.: WhatsApp) e para a recuperação de jobs, onde só temos os bytes
// da imagem.
package qrdecode

import (
	"bytes"
	"image"
	_ "image/jpeg" // registra o decoder JPEG
	_ "image/png"  // registra o decoder PNG
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// Decoder lê QR Codes de imagens. Sem estado; seguro para uso concorrente.
type Decoder struct{}

func New() Decoder { return Decoder{} }

// DecodeNFCe tenta ler o QR Code de uma imagem de cupom e devolve seu conteúdo
// (tipicamente a URL da SEFAZ com a chave de acesso + hash). ok=false quando não
// há QR legível ou o formato não é suportado (ex.: HEIC não é decodificado aqui).
func (Decoder) DecodeNFCe(content []byte) (string, bool) {
	if len(content) == 0 {
		return "", false
	}
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return "", false
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", false
	}
	// TRY_HARDER melhora a taxa de acerto em fotos (perspectiva/ruído), ao custo
	// de mais processamento — aceitável num fluxo assíncrono.
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		return "", false
	}
	text := strings.TrimSpace(res.GetText())
	return text, text != ""
}
