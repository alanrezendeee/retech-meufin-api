package finance

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// fakeDocRepo implementa dom.FinanceDocumentRepository em memória.
type fakeDocRepo struct {
	docs map[uuid.UUID]*dom.FinanceDocument
}

func newFakeDocRepo() *fakeDocRepo { return &fakeDocRepo{docs: map[uuid.UUID]*dom.FinanceDocument{}} }

func (f *fakeDocRepo) Create(_ context.Context, d *dom.FinanceDocument) error {
	f.docs[d.ID] = d
	return nil
}
func (f *fakeDocRepo) GetByID(_ context.Context, ws, id uuid.UUID) (*dom.FinanceDocument, error) {
	d, ok := f.docs[id]
	if !ok || d.WorkspaceID != ws {
		return nil, dom.ErrNotFound
	}
	return d, nil
}
func (f *fakeDocRepo) List(_ context.Context, ws uuid.UUID, filter dom.FinanceDocumentFilter, _, _ int) ([]dom.FinanceDocument, int64, error) {
	var out []dom.FinanceDocument
	for _, d := range f.docs {
		if d.WorkspaceID != ws {
			continue
		}
		if filter.Kind != nil && d.Kind != *filter.Kind {
			continue
		}
		if filter.EntryID != nil && (d.EntryID == nil || *d.EntryID != *filter.EntryID) {
			continue
		}
		out = append(out, *d)
	}
	return out, int64(len(out)), nil
}
func (f *fakeDocRepo) UpdateExtraction(_ context.Context, d *dom.FinanceDocument) error {
	f.docs[d.ID] = d
	return nil
}
func (f *fakeDocRepo) SoftDelete(_ context.Context, _, id uuid.UUID) error {
	delete(f.docs, id)
	return nil
}

// fakeStorage guarda os bytes enviados por objectKey.
type fakeStorage struct{ objects map[string][]byte }

func newFakeStorage() *fakeStorage { return &fakeStorage{objects: map[string][]byte{}} }

func (fakeStorage) Enabled() bool { return true }
func (f *fakeStorage) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.objects[key] = b
	return nil
}
func (fakeStorage) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://minio.test/" + key, nil
}
func (f *fakeStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := f.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// fakeQR devolve um payload fixo para qualquer imagem.
type fakeQR struct {
	payload string
	ok      bool
	calls   int
}

func (q *fakeQR) DecodeNFCe(_ []byte) (string, bool) {
	q.calls++
	return q.payload, q.ok
}

const pixPayload = "00020126580014br.gov.bcb.pix0136123e4567-e12b-12d1-a456-4266554400005204000053039865802BR5913Fulano de Tal6008BRASILIA62070503***63041D3D"

func newAttachmentSvc(t *testing.T, qr qrDecoder) (*FinanceDocumentService, *fakeDocRepo, *fakeStorage) {
	t.Helper()
	repo := newFakeDocRepo()
	st := newFakeStorage()
	svc := NewFinanceDocumentService(repo, st, 5)
	if qr != nil {
		svc.SetQRDecoder(qr)
	}
	return svc, repo, st
}

func baseInput(entryID uuid.UUID, typ dom.AttachmentType, mime, body string) UploadAttachmentInput {
	return UploadAttachmentInput{
		WorkspaceID:      uuid.New(),
		UploadedByUserID: uuid.New(),
		EntryID:          entryID,
		Type:             typ,
		OriginalFileName: "arquivo.bin",
		MimeType:         mime,
		Size:             int64(len(body)),
		Content:          strings.NewReader(body),
	}
}

func TestNormalizePaymentCode(t *testing.T) {
	cases := []struct {
		name    string
		typ     dom.AttachmentType
		raw     string
		want    string
		wantErr bool
	}{
		{"vazio ok", dom.AttachmentBoleto, "  ", "", false},
		{"boleto 47 com pontuação", dom.AttachmentBoleto,
			"23793.38128 60000.000003 00000.000400 1 84340000010000",
			"23793381286000000000300000000400184340000010000", false},
		{"boleto arrecadação 48", dom.AttachmentBoleto,
			"836900000015 999900481000 000000000000 000000000000",
			"836900000015999900481000000000000000000000000000", false},
		{"boleto curto", dom.AttachmentBoleto, "1234567890", "", true},
		{"pix válido", dom.AttachmentPixQRCode, " " + pixPayload + " ", pixPayload, false},
		{"pix inválido", dom.AttachmentPixQRCode, "chave@email.com", "", true},
		{"outro livre", dom.AttachmentOutro, "ref 123", "ref 123", false},
		{"outro longo", dom.AttachmentOutro, strings.Repeat("x", 513), "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizePaymentCode(tc.typ, tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestUploadAttachment_UserCodeWinsOverQR(t *testing.T) {
	qr := &fakeQR{payload: pixPayload, ok: true}
	svc, repo, st := newAttachmentSvc(t, qr)
	entryID := uuid.New()
	in := baseInput(entryID, dom.AttachmentBoleto, "image/png", "png-bytes")
	in.PaymentCode = "23793.38128 60000.000003 00000.000400 1 84340000010000"
	in.Note = "vence dia 10"

	res, err := svc.UploadAttachment(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	doc := res.Doc
	if qr.calls != 0 {
		t.Fatalf("QR não deveria ser lido quando o usuário informa o código; calls=%d", qr.calls)
	}
	meta, ok := doc.AttachmentMeta()
	if !ok {
		t.Fatal("meta ausente")
	}
	if meta.Type != dom.AttachmentBoleto || meta.PaymentCode == nil || *meta.PaymentCode != "23793381286000000000300000000400184340000010000" {
		t.Fatalf("meta inesperada: %+v", meta)
	}
	if meta.PaymentCodeSource == nil || *meta.PaymentCodeSource != "user" || meta.Note == nil || *meta.Note != "vence dia 10" {
		t.Fatalf("source/note inesperados: %+v", meta)
	}
	if doc.Kind != dom.DocumentAttachment || doc.EntryID == nil || *doc.EntryID != entryID {
		t.Fatalf("doc inesperado: kind=%s entry=%v", doc.Kind, doc.EntryID)
	}
	if !strings.Contains(doc.ObjectKey, "/finance/attachments/"+entryID.String()+"/") {
		t.Fatalf("objectKey inesperada: %s", doc.ObjectKey)
	}
	if string(st.objects[doc.ObjectKey]) != "png-bytes" {
		t.Fatalf("conteúdo não chegou íntegro ao storage")
	}
	if _, ok := repo.docs[doc.ID]; !ok {
		t.Fatal("doc não persistido")
	}
}

func TestUploadAttachment_ReadsPixFromQRCode(t *testing.T) {
	qr := &fakeQR{payload: pixPayload, ok: true}
	svc, _, st := newAttachmentSvc(t, qr)
	in := baseInput(uuid.New(), dom.AttachmentPixQRCode, "image/jpeg", "jpeg-bytes")

	res, err := svc.UploadAttachment(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	doc := res.Doc
	if qr.calls != 1 {
		t.Fatalf("QR deveria ser lido uma vez; calls=%d", qr.calls)
	}
	meta, _ := doc.AttachmentMeta()
	if meta.PaymentCode == nil || *meta.PaymentCode != pixPayload {
		t.Fatalf("payload não capturado: %+v", meta)
	}
	if meta.PaymentCodeSource == nil || *meta.PaymentCodeSource != "qrcode" {
		t.Fatalf("source deveria ser qrcode: %+v", meta)
	}
	// Os bytes foram bufferizados para o decode; o upload deve seguir íntegro.
	if string(st.objects[doc.ObjectKey]) != "jpeg-bytes" {
		t.Fatalf("conteúdo não chegou íntegro ao storage após o decode")
	}
}

func TestUploadAttachment_IgnoresNonPixQR(t *testing.T) {
	qr := &fakeQR{payload: "https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx?p=4321", ok: true}
	svc, _, _ := newAttachmentSvc(t, qr)
	res, err := svc.UploadAttachment(context.Background(), baseInput(uuid.New(), dom.AttachmentPixQRCode, "image/png", "x"))
	if err != nil {
		t.Fatal(err)
	}
	meta, _ := res.Doc.AttachmentMeta()
	if meta.PaymentCode != nil {
		t.Fatalf("QR de NFC-e não é código de pagamento: %+v", meta)
	}
}

func TestUploadAttachment_SkipsQRForPDFAndOtherTypes(t *testing.T) {
	qr := &fakeQR{payload: pixPayload, ok: true}
	svc, _, _ := newAttachmentSvc(t, qr)
	if _, err := svc.UploadAttachment(context.Background(), baseInput(uuid.New(), dom.AttachmentPixQRCode, "application/pdf", "%PDF")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadAttachment(context.Background(), baseInput(uuid.New(), dom.AttachmentContrato, "image/png", "png")); err != nil {
		t.Fatal(err)
	}
	if qr.calls != 0 {
		t.Fatalf("QR não deveria ser tentado em PDF nem em tipos sem código; calls=%d", qr.calls)
	}
}

func TestUploadAttachment_Validation(t *testing.T) {
	svc, _, _ := newAttachmentSvc(t, nil)
	ctx := context.Background()
	var ve *dom.ValidationError

	in := baseInput(uuid.New(), dom.AttachmentType("cheque"), "image/png", "x")
	if _, err := svc.UploadAttachment(ctx, in); !errors.As(err, &ve) {
		t.Fatalf("tipo inválido deveria falhar com ValidationError; err=%v", err)
	}
	in = baseInput(uuid.New(), dom.AttachmentBoleto, "text/plain", "x")
	if _, err := svc.UploadAttachment(ctx, in); !errors.As(err, &ve) {
		t.Fatalf("mime inválido deveria falhar; err=%v", err)
	}
	in = baseInput(uuid.Nil, dom.AttachmentBoleto, "image/png", "x")
	if _, err := svc.UploadAttachment(ctx, in); !errors.As(err, &ve) {
		t.Fatalf("entry nil deveria falhar; err=%v", err)
	}
	in = baseInput(uuid.New(), dom.AttachmentBoleto, "image/png", "x")
	in.PaymentCode = "123"
	if _, err := svc.UploadAttachment(ctx, in); !errors.As(err, &ve) {
		t.Fatalf("linha digitável curta deveria falhar; err=%v", err)
	}
}

func TestListAttachments_OnlyAttachmentsOfEntry(t *testing.T) {
	svc, _, _ := newAttachmentSvc(t, nil)
	ctx := context.Background()
	ws := uuid.New()
	entryA, entryB := uuid.New(), uuid.New()

	mk := func(entry uuid.UUID, typ dom.AttachmentType) {
		in := baseInput(entry, typ, "application/pdf", "pdf")
		in.WorkspaceID = ws
		if _, err := svc.UploadAttachment(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	mk(entryA, dom.AttachmentBoleto)
	mk(entryA, dom.AttachmentContrato)
	mk(entryB, dom.AttachmentOutro)
	// Um comprovante no mesmo lançamento não pode vazar para a lista de anexos.
	rin := baseInput(entryA, "", "application/pdf", "pdf")
	rin.WorkspaceID = ws
	if _, err := svc.UploadReceipt(ctx, UploadFinanceDocInput{
		WorkspaceID: ws, UploadedByUserID: rin.UploadedByUserID, OriginalFileName: "r.pdf",
		MimeType: "application/pdf", Size: 3, Content: strings.NewReader("pdf"),
	}, entryA); err != nil {
		t.Fatal(err)
	}

	res, err := svc.ListAttachments(ctx, ws, entryA, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 || len(res.Items) != 2 {
		t.Fatalf("esperado 2 anexos em A; total=%d len=%d", res.Total, len(res.Items))
	}
	rec, err := svc.ListReceipts(ctx, ws, entryA, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Total != 1 {
		t.Fatalf("esperado 1 comprovante em A; total=%d", rec.Total)
	}
}

// pixDynamicPayload tem o campo 01 = "12" (Point of Initiation dinâmico).
const pixDynamicPayload = "00020101021226850014br.gov.bcb.pix2563pix.example.com/qr/v2/9d36b84f-c70b-478f-b95c-12729b90ca255204000053039865802BR5913Fulano de Tal6008BRASILIA62070503***63041D3D"

func TestPixIsDynamic(t *testing.T) {
	if PixIsDynamic(pixPayload) {
		t.Fatal("payload sem campo 01 é estático")
	}
	if !PixIsDynamic(pixDynamicPayload) {
		t.Fatal("campo 01=12 deveria ser dinâmico")
	}
	static11 := "000201010211" + pixPayload[6:]
	if PixIsDynamic(static11) {
		t.Fatal("campo 01=11 é estático")
	}
	if PixIsDynamic("0002") {
		t.Fatal("payload truncado não pode ser dinâmico")
	}
}

// seriesFixture cria uma série de 5 parcelas mensais: #1 realizada (passada),
// #2 alvo, #3 e #5 previstas futuras, #4 cancelada. Retorna o repo e os ids.
func seriesFixture(ws uuid.UUID) (*fakeEntryRepo, uuid.UUID, []uuid.UUID) {
	repo := newFakeEntryRepo()
	group := uuid.New()
	base := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	statuses := []dom.Status{dom.StatusRealizada, dom.StatusPrevista, dom.StatusPrevista, dom.StatusCancelada, dom.StatusPrevista}
	ids := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		ids[i] = uuid.New()
		repo.entries[ids[i]] = &dom.FinancialEntry{
			ID: ids[i], WorkspaceID: ws, RecurrenceGroupID: &group,
			DueDate: base.AddDate(0, i, 0), Status: statuses[i], Kind: dom.KindDebit,
		}
	}
	return repo, ids[1], []uuid.UUID{ids[2], ids[4]}
}

func TestUploadAttachment_ReplicatesPixToFuturePrevistas(t *testing.T) {
	svc, docRepo, st := newAttachmentSvc(t, nil)
	ws := uuid.New()
	entries, target, wantFuture := seriesFixture(ws)
	svc.SetEntryRepo(entries)

	in := baseInput(target, dom.AttachmentPixQRCode, "application/pdf", "pdf")
	in.WorkspaceID = ws
	in.PaymentCode = pixPayload
	in.ApplyToFuture = true

	res, err := svc.UploadAttachment(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if res.ReplicatedTo != 2 {
		t.Fatalf("esperado replicar em 2 futuras previstas (pula passada, realizada e cancelada); got %d", res.ReplicatedTo)
	}
	if len(docRepo.docs) != 3 {
		t.Fatalf("esperado 3 linhas (alvo + 2 cópias); got %d", len(docRepo.docs))
	}
	if len(st.objects) != 1 {
		t.Fatalf("cópias devem apontar para o MESMO objeto; objetos=%d", len(st.objects))
	}
	seen := map[uuid.UUID]bool{}
	for _, d := range docRepo.docs {
		if d.ObjectKey != res.Doc.ObjectKey {
			t.Fatalf("object_key divergente: %s vs %s", d.ObjectKey, res.Doc.ObjectKey)
		}
		m, _ := d.AttachmentMeta()
		if d.ID == res.Doc.ID {
			if m.ReplicatedFromEntryID != nil {
				t.Fatal("o original não é réplica")
			}
			continue
		}
		seen[*d.EntryID] = true
		if m.ReplicatedFromEntryID == nil || *m.ReplicatedFromEntryID != target {
			t.Fatalf("réplica deve apontar para o lançamento de origem: %+v", m)
		}
		if m.PaymentCode == nil || *m.PaymentCode != pixPayload {
			t.Fatalf("réplica deve carregar o mesmo código: %+v", m)
		}
	}
	for _, id := range wantFuture {
		if !seen[id] {
			t.Fatalf("parcela futura %s não recebeu cópia", id)
		}
	}
	// Listagem por parcela enxerga a cópia.
	lst, err := svc.ListAttachments(context.Background(), ws, wantFuture[0], 10, 0)
	if err != nil || lst.Total != 1 {
		t.Fatalf("parcela futura deveria listar 1 anexo; total=%d err=%v", lst.Total, err)
	}
}

func TestUploadAttachment_BoletoNeverReplicates(t *testing.T) {
	svc, docRepo, _ := newAttachmentSvc(t, nil)
	ws := uuid.New()
	entries, target, _ := seriesFixture(ws)
	svc.SetEntryRepo(entries)

	for _, typ := range []dom.AttachmentType{dom.AttachmentBoleto, dom.AttachmentNotaFiscal, dom.AttachmentFatura, dom.AttachmentOutro} {
		in := baseInput(target, typ, "application/pdf", "pdf")
		in.WorkspaceID = ws
		in.ApplyToFuture = true
		res, err := svc.UploadAttachment(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		if res.ReplicatedTo != 0 {
			t.Fatalf("%s não pode replicar mesmo com apply_to=future; got %d", typ, res.ReplicatedTo)
		}
	}
	if len(docRepo.docs) != 4 {
		t.Fatalf("esperado 4 linhas (uma por upload, sem cópias); got %d", len(docRepo.docs))
	}
}

func TestUploadAttachment_ReplicationNoopOutsideSeriesOrWithoutRepo(t *testing.T) {
	ws := uuid.New()

	// Sem repo de lançamentos: replicação silenciosamente desligada.
	svc, _, _ := newAttachmentSvc(t, nil)
	in := baseInput(uuid.New(), dom.AttachmentContrato, "application/pdf", "pdf")
	in.WorkspaceID = ws
	in.ApplyToFuture = true
	res, err := svc.UploadAttachment(context.Background(), in)
	if err != nil || res.ReplicatedTo != 0 {
		t.Fatalf("sem repo: replicated=0 esperado; got %d err=%v", res.ReplicatedTo, err)
	}

	// Lançamento avulso (sem grupo): nada a replicar.
	svc2, _, _ := newAttachmentSvc(t, nil)
	entries := newFakeEntryRepo()
	solo := &dom.FinancialEntry{ID: uuid.New(), WorkspaceID: ws, DueDate: time.Now(), Status: dom.StatusPrevista, Kind: dom.KindDebit}
	entries.entries[solo.ID] = solo
	svc2.SetEntryRepo(entries)
	in = baseInput(solo.ID, dom.AttachmentContrato, "application/pdf", "pdf")
	in.WorkspaceID = ws
	in.ApplyToFuture = true
	res, err = svc2.UploadAttachment(context.Background(), in)
	if err != nil || res.ReplicatedTo != 0 {
		t.Fatalf("avulso: replicated=0 esperado; got %d err=%v", res.ReplicatedTo, err)
	}

	// Lançamento inexistente com apply_to=future: falha antes de gravar.
	svc3, docRepo3, st3 := newAttachmentSvc(t, nil)
	svc3.SetEntryRepo(newFakeEntryRepo())
	in = baseInput(uuid.New(), dom.AttachmentPixQRCode, "application/pdf", "pdf")
	in.WorkspaceID = ws
	in.ApplyToFuture = true
	if _, err := svc3.UploadAttachment(context.Background(), in); !errors.Is(err, dom.ErrNotFound) {
		t.Fatalf("esperado ErrNotFound; got %v", err)
	}
	if len(docRepo3.docs) != 0 || len(st3.objects) != 0 {
		t.Fatal("nada deveria ter sido gravado")
	}
}

func TestUploadAttachment_MarksPixDynamic(t *testing.T) {
	svc, _, _ := newAttachmentSvc(t, nil)
	in := baseInput(uuid.New(), dom.AttachmentPixQRCode, "application/pdf", "pdf")
	in.PaymentCode = pixDynamicPayload
	res, err := svc.UploadAttachment(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := res.Doc.AttachmentMeta()
	if m.PixDynamic == nil || !*m.PixDynamic {
		t.Fatalf("pix_dynamic deveria ser true: %+v", m)
	}
	in = baseInput(uuid.New(), dom.AttachmentPixQRCode, "application/pdf", "pdf")
	in.PaymentCode = pixPayload
	res, _ = svc.UploadAttachment(context.Background(), in)
	m, _ = res.Doc.AttachmentMeta()
	if m.PixDynamic == nil || *m.PixDynamic {
		t.Fatalf("pix_dynamic deveria ser false para estático: %+v", m)
	}
}
