-- Extração v3 transcreve referência/método verbatim do laudo: tabelas por
-- sexo/idade e metas por risco passam fácil de 255 chars (LDL ~500). VARCHAR
-- curto estourava "value too long" no confirm da importação.
ALTER TABLE health_exam_result_items ALTER COLUMN reference_text TYPE TEXT;
ALTER TABLE health_exam_result_items ALTER COLUMN method TYPE TEXT;
ALTER TABLE health_exam_result_items ALTER COLUMN material TYPE TEXT;
ALTER TABLE health_markers ALTER COLUMN default_ref_text TYPE TEXT;
